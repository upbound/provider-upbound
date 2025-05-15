// Copyright 2025 Upbound Inc
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package v1alpha1

import (
	"github.com/upbound/up-sdk-go/apis/spaces/v1beta1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"reflect"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
)

// ControlPlaneSpec defines the desired state of ControlPlane.
type ControlPlaneSpec struct {
	xpv1.ResourceSpec `json:",inline"`
	ForProvider       ControlPlaneParameters `json:"forProvider"`
}

type ControlPlaneParameters struct {
	Name  string `json:"name"`
	Space string `json:"space"`

	// ControlPlaneGroupName is the name of the ControlPlaneGroup you'd like to fetch Kubeconfig of.
	// Either ControlPlaneGroupName, ControlPlaneGroupRef or ControlPlaneGroupSelector has to be given.
	// +crossplane:generate:reference:type=github.com/upbound/provider-upbound/apis/spacesmgmt/v1alpha1.ControlPlaneGroup
	ControlPlaneGroupName string `json:"controlPlaneGroupName,omitempty"`

	// Reference to a ControlPlaneGroup to populate controlPlaneName.
	// Either ControlPlaneGroupName, ControlPlaneGroupRef or ControlPlaneGroupSelector has to be given.
	// +kubebuilder:validation:Optional
	ControlPlaneGroupNameRef *xpv1.Reference `json:"controlPlaneGroupNameRef,omitempty"`

	// Selector for a ControlPlane to populate controlPlaneName.
	// Either ControlPlaneGroupName, ControlPlaneGroupRef or ControlPlaneGroupSelector has to be given.
	// +kubebuilder:validation:Optional
	ControlPlaneGroupNameSelector *xpv1.Selector `json:"controlPlaneGroupNameSelector,omitempty"`

	Details v1beta1.ControlPlaneSpec `json:"details"`
}

// A ControlPlaneStatus represents the observed state of a ControlPlane.
type ControlPlaneStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          v1beta1.ControlPlaneStatus `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// +kubebuilder:printcolumn:name="Crossplane",type="string",JSONPath=".spec.crossplane.version"
// +kubebuilder:printcolumn:name="Ready",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="Message",type="string",JSONPath=`.status.message`
// +kubebuilder:printcolumn:name="Class",type="string",JSONPath=".spec.class",priority=1
// +kubebuilder:printcolumn:name="CPU Usage",type="string",JSONPath=".status.size.resourceUsage.cpu",priority=1
// +kubebuilder:printcolumn:name="Memory Usage",type="string",JSONPath=".status.size.resourceUsage.memory",priority=1
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,categories=spacesmgmt,shortName=ctp;ctps

// ControlPlane defines a managed Crossplane instance.
// +kubebuilder:validation:XValidation:rule="self.metadata.name.size() <= 63",message="control plane name cannot exceed 63 characters"
type ControlPlane struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ControlPlaneSpec   `json:"spec"`
	Status ControlPlaneStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ControlPlaneList contains a list of ControlPlane
type ControlPlaneList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ControlPlane `json:"items"`
}

// GetCondition of this ControlPlane.
func (mg *ControlPlane) GetCondition(ct xpv1.ConditionType) xpv1.Condition {
	return mg.Status.GetCondition(ct)
}

// SetConditions of this ControlPlane.
func (mg *ControlPlane) SetConditions(c ...xpv1.Condition) {
	mg.Status.SetConditions(c...)
}

// ManagedControlPlane type metadata.
var (
	// ControlPlaneKind is the kind of the ControlPlane.
	ControlPlaneKind = reflect.TypeOf(ControlPlane{}).Name()
	// ControlPlaneListKind is the kind of a list of ControlPlane.
	ControlPlaneListKind         = reflect.TypeOf(ControlPlaneList{}).Name()
	ControlPlaneGroupKind        = schema.GroupKind{Group: Group, Kind: ControlPlaneKind}.String()
	ControlPlaneKindAPIVersion   = ControlPlaneKind + "." + SchemeGroupVersion.String()
	ControlPlaneGroupVersionKind = SchemeGroupVersion.WithKind(ControlPlaneKind)
)

func init() {
	SchemeBuilder.Register(&ControlPlane{}, &ControlPlaneList{})
}
