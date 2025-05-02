/*
Copyright 2023 Upbound Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	"reflect"

	v1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
)

type ControlPlaneAuthParameters struct {

	// ControlPlaneName is the name of the ControlPlane you'd like to fetch Kubeconfig of.
	// Either ControlPlaneName, ControlPlaneNameRef or ControlPlaneNameSelector has to be given.
	// +crossplane:generate:reference:type=ControlPlane
	ControlPlaneName string `json:"controlPlaneName,omitempty"`

	// Reference to a ControlPlane to populate controlPlaneName.
	// Either ControlPlaneName, ControlPlaneRef or ControlPlaneSelector has to be given.
	// +kubebuilder:validation:Optional
	ControlPlaneRef *v1.Reference `json:"controlPlaneRef,omitempty"`

	// Selector for a ControlPlane to populate controlPlaneName.
	// Either ControlPlaneName, ControlPlaneRef or ControlPlaneSelector has to be given.
	// +kubebuilder:validation:Optional
	ControlPlaneSelector *v1.Selector `json:"controlPlaneSelector,omitempty"`

	// OrganizationName is the name of the organization to which the control plane
	// belongs.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	OrganizationName string `json:"organizationName"`

	// A Token ConnectionSecret is referenced to serve as
	// the authentication token for a KubeConfig
	// +optional
	TokenSecretRef *xpv1.SecretKeySelector `json:"tokenSecretRef,omitempty"`
}

type ControlPlaneAuthObservation struct{}

// ControlPlaneAuthSpec defines the desired state of ControlPlaneAuth
type ControlPlaneAuthSpec struct {
	v1.ResourceSpec `json:",inline"`
	ForProvider     ControlPlaneAuthParameters `json:"forProvider"`
}

// ControlPlaneAuthStatus defines the observed state of ControlPlaneAuth.
type ControlPlaneAuthStatus struct {
	v1.ResourceStatus `json:",inline"`
	AtProvider        ControlPlaneAuthObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true

// ControlPlaneAuth is used to retrieve Kubeconfig of given ControlPlane.
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,categories={crossplane,managed,upbound}
type ControlPlaneAuth struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              ControlPlaneAuthSpec   `json:"spec"`
	Status            ControlPlaneAuthStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ControlPlaneAuthList contains a list of ControlPlaneAuths
type ControlPlaneAuthList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ControlPlaneAuth `json:"items"`
}

// ControlPlaneAuth type metadata.
var (
	ControlPlaneAuthKind             = reflect.TypeOf(ControlPlaneAuth{}).Name()
	ControlPlaneAuthGroupKind        = schema.GroupKind{Group: Group, Kind: ControlPlaneAuthKind}.String()
	ControlPlaneAuthKindAPIVersion   = ControlPlaneAuthKind + "." + SchemeGroupVersion.String()
	ControlPlaneAuthGroupVersionKind = SchemeGroupVersion.WithKind(ControlPlaneAuthKind)
)

func init() {
	SchemeBuilder.Register(&ControlPlaneAuth{}, &ControlPlaneAuthList{})
}
