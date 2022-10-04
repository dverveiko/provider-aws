/*
Copyright 2021 Upbound Inc.
*/

package main

import (
	"context"
	"os"
	"path/filepath"
	"time"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	xpcontroller "github.com/crossplane/crossplane-runtime/pkg/controller"
	"github.com/crossplane/crossplane-runtime/pkg/feature"
	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/crossplane/crossplane-runtime/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	tjcontroller "github.com/upbound/upjet/pkg/controller"
	"github.com/upbound/upjet/pkg/terraform"
	"gopkg.in/alecthomas/kingpin.v2"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"github.com/upbound/official-providers/provider-aws/apis"
	"github.com/upbound/official-providers/provider-aws/apis/v1alpha1"
	"github.com/upbound/official-providers/provider-aws/config"
	"github.com/upbound/official-providers/provider-aws/internal/clients"
	"github.com/upbound/official-providers/provider-aws/internal/controller"
	"github.com/upbound/official-providers/provider-aws/internal/features"
)

const upboundCTXEnv = "UPBOUND_CONTEXT"

func main() {
	var (
		app                = kingpin.New(filepath.Base(os.Args[0]), "AWS support for Crossplane.").DefaultEnvars()
		debug              = app.Flag("debug", "Run with debug logging.").Short('d').Bool()
		syncPeriod         = app.Flag("sync", "Controller manager sync period such as 300ms, 1.5h, or 2h45m").Short('s').Default("1h").Duration()
		leaderElection     = app.Flag("leader-election", "Use leader election for the controller manager.").Short('l').Default("false").OverrideDefaultFromEnvar("LEADER_ELECTION").Bool()
		terraformVersion   = app.Flag("terraform-version", "Terraform version.").Required().Envar("TERRAFORM_VERSION").String()
		providerSource     = app.Flag("terraform-provider-source", "Terraform provider source.").Required().Envar("TERRAFORM_PROVIDER_SOURCE").String()
		providerVersion    = app.Flag("terraform-provider-version", "Terraform provider version.").Required().Envar("TERRAFORM_PROVIDER_VERSION").String()
		nativeProviderPath = app.Flag("terraform-native-provider-path", "Terraform native provider path for shared execution.").Default("").Envar("TERRAFORM_NATIVE_PROVIDER_PATH").String()
		maxReconcileRate   = app.Flag("max-reconcile-rate", "The global maximum rate per second at which resources may checked for drift from the desired state.").Default("10").Int()

		namespace                  = app.Flag("namespace", "Namespace used to set as default scope in default secret store config.").Default("crossplane-system").Envar("POD_NAMESPACE").String()
		enableExternalSecretStores = app.Flag("enable-external-secret-stores", "Enable support for ExternalSecretStores.").Default("false").Envar("ENABLE_EXTERNAL_SECRET_STORES").Bool()
	)

	// If UPBOUND_CONTEXT is not set, we refuse to run.
	if _, set := os.LookupEnv(upboundCTXEnv); !set {
		kingpin.Fatalf("Running this provider outside of an Upbound distribution is prohibited by license. Please contact support@upbound.io for more information.")
	}

	kingpin.MustParse(app.Parse(os.Args[1:]))

	zl := zap.New(zap.UseDevMode(*debug))
	log := logging.NewLogrLogger(zl.WithName("provider-aws"))
	if *debug {
		// The controller-runtime runs with a no-op logger by default. It is
		// *very* verbose even at info level, so we only provide it a real
		// logger when we're running in debug mode.
		ctrl.SetLogger(zl)
	}

	log.Debug("Starting", "sync-period", syncPeriod.String())

	cfg, err := ctrl.GetConfig()
	kingpin.FatalIfError(err, "Cannot get API server rest config")

	mgr, err := ctrl.NewManager(ratelimiter.LimitRESTConfig(cfg, *maxReconcileRate), ctrl.Options{
		LeaderElection:             *leaderElection,
		LeaderElectionID:           "crossplane-leader-election-provider-aws",
		SyncPeriod:                 syncPeriod,
		LeaderElectionResourceLock: resourcelock.LeasesResourceLock,
		LeaseDuration:              func() *time.Duration { d := 60 * time.Second; return &d }(),
		RenewDeadline:              func() *time.Duration { d := 50 * time.Second; return &d }(),
	})
	kingpin.FatalIfError(err, "Cannot create controller manager")
	kingpin.FatalIfError(apis.AddToScheme(mgr.GetScheme()), "Cannot add AWS APIs to scheme")

	// if the native Terraform provider plugin's path is not configured via
	// the env. variable TERRAFORM_NATIVE_PROVIDER_PATH or
	// the `--terraform-native-provider-path` command-line option,
	// we do not use the shared gRPC server and default to the regular
	// Terraform CLI behaviour (of forking a plugin process per invocation).
	// This removes some complexity for setting up development environments.
	var runner terraform.ProviderRunner = terraform.NewNoOpProviderRunner()
	if len(*nativeProviderPath) != 0 {
		runner = terraform.NewSharedProvider(log, *nativeProviderPath, "registry.terraform.io/"+*providerSource)
	}

	o := tjcontroller.Options{
		Options: xpcontroller.Options{
			Logger:                  log,
			GlobalRateLimiter:       ratelimiter.NewGlobal(*maxReconcileRate),
			PollInterval:            1 * time.Minute,
			MaxConcurrentReconciles: 1,
			Features:                &feature.Flags{},
		},
		Provider:       config.GetProvider(),
		WorkspaceStore: terraform.NewWorkspaceStore(log, terraform.WithProviderRunner(runner)),
		SetupFn:        clients.TerraformSetupBuilder(*terraformVersion, *providerSource, *providerVersion),
	}

	if *enableExternalSecretStores {
		o.SecretStoreConfigGVK = &v1alpha1.StoreConfigGroupVersionKind
		log.Info("Alpha feature enabled", "flag", features.EnableAlphaExternalSecretStores)
		// Ensure default store config exists.
		kingpin.FatalIfError(resource.Ignore(kerrors.IsAlreadyExists, mgr.GetClient().Create(context.Background(), &v1alpha1.StoreConfig{
			TypeMeta: metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{
				Name: "default",
			},
			Spec: v1alpha1.StoreConfigSpec{
				// NOTE(turkenh): We only set required spec and expect optional
				// ones to properly be initialized with CRD level default values.
				SecretStoreConfig: xpv1.SecretStoreConfig{
					DefaultScope: *namespace,
				},
			},
			Status: v1alpha1.StoreConfigStatus{},
		})), "cannot create default store config")
	}

	kingpin.FatalIfError(controller.Setup(mgr, o), "Cannot setup AWS controllers")
	kingpin.FatalIfError(mgr.Start(ctrl.SetupSignalHandler()), "Cannot start controller manager")
}