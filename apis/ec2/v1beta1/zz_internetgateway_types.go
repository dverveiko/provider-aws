/*
Copyright 2022 Upbound Inc.
*/

// Code generated by upjet. DO NOT EDIT.

package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	v1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
)

type InternetGatewayObservation_2 struct {

	// The ARN of the Internet Gateway.
	Arn *string `json:"arn,omitempty" tf:"arn,omitempty"`

	// The ID of the Internet Gateway.
	ID *string `json:"id,omitempty" tf:"id,omitempty"`

	// The ID of the AWS account that owns the internet gateway.
	OwnerID *string `json:"ownerId,omitempty" tf:"owner_id,omitempty"`

	// A map of tags assigned to the resource, including those inherited from the provider default_tags configuration block.
	TagsAll map[string]*string `json:"tagsAll,omitempty" tf:"tags_all,omitempty"`
}

type InternetGatewayParameters_2 struct {

	// Region is the region you'd like your resource to be created in.
	// +upjet:crd:field:TFTag=-
	// +kubebuilder:validation:Required
	Region *string `json:"region" tf:"-"`

	// Key-value map of resource tags.
	// +kubebuilder:validation:Optional
	Tags map[string]*string `json:"tags,omitempty" tf:"tags,omitempty"`

	// The VPC ID to create in.  See the aws_internet_gateway_attachment resource for an alternate way to attach an Internet Gateway to a VPC.
	// +crossplane:generate:reference:type=github.com/upbound/provider-aws/apis/ec2/v1beta1.VPC
	// +kubebuilder:validation:Optional
	VPCID *string `json:"vpcId,omitempty" tf:"vpc_id,omitempty"`

	// Reference to a VPC in ec2 to populate vpcId.
	// +kubebuilder:validation:Optional
	VPCIDRef *v1.Reference `json:"vpcIdRef,omitempty" tf:"-"`

	// Selector for a VPC in ec2 to populate vpcId.
	// +kubebuilder:validation:Optional
	VPCIDSelector *v1.Selector `json:"vpcIdSelector,omitempty" tf:"-"`
}

// InternetGatewaySpec defines the desired state of InternetGateway
type InternetGatewaySpec struct {
	v1.ResourceSpec `json:",inline"`
	ForProvider     InternetGatewayParameters_2 `json:"forProvider"`
}

// InternetGatewayStatus defines the observed state of InternetGateway.
type InternetGatewayStatus struct {
	v1.ResourceStatus `json:",inline"`
	AtProvider        InternetGatewayObservation_2 `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true

// InternetGateway is the Schema for the InternetGateways API. Provides a resource to create a VPC Internet Gateway.
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,categories={crossplane,managed,aws}
type InternetGateway struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              InternetGatewaySpec   `json:"spec"`
	Status            InternetGatewayStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// InternetGatewayList contains a list of InternetGateways
type InternetGatewayList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []InternetGateway `json:"items"`
}

// Repository type metadata.
var (
	InternetGateway_Kind             = "InternetGateway"
	InternetGateway_GroupKind        = schema.GroupKind{Group: CRDGroup, Kind: InternetGateway_Kind}.String()
	InternetGateway_KindAPIVersion   = InternetGateway_Kind + "." + CRDGroupVersion.String()
	InternetGateway_GroupVersionKind = CRDGroupVersion.WithKind(InternetGateway_Kind)
)

func init() {
	SchemeBuilder.Register(&InternetGateway{}, &InternetGatewayList{})
}
