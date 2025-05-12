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
	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Group represents a group of control planes.
// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,categories={crossplane,managed,upbound}
type ControlPlaneGroup struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ControlPlaneGroupSpec   `json:"spec,omitempty"`
	Status ControlPlaneGroupStatus `json:"status,omitempty"`
}

// ControlPlaneGroupList contains a list of ControlPlaneGroup.
//
// +kubebuilder:object:root=true
type ControlPlaneGroupList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ControlPlaneGroup `json:"items"`
}

// Objects return the list of items.
func (s *ControlPlaneGroupList) Objects() []client.Object {
	var objs = make([]client.Object, len(s.Items))
	for i := range s.Items {
		objs[i] = &s.Items[i]
	}
	return objs
}

type ControlPlaneGroupParameters struct {
	// Name is the name to use when creating a control plane group.
	// optional, if not set, Group name will be used.
	// When set, it is immutable.
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="value is immutable"
	// +kubebuilder:validation:MaxLength=253
	// +kubebuilder:validation:MinLength=1
	// +optional
	Name string `json:"name,omitempty"`

	// SpaceName is the name of the Space you'd like to fetch Kubeconfig of.
	// Either SpaceName, SpaceRef or SpaceSelector has to be given.
	//SpaceName string `json:"spaceName,omitempty"`

	// DeletionProtection enable deletion protection on the group
	// +kubebuilder:validation:Optional
	DeletionProtection bool `json:"deletionProtection,omitempty"`
	//// Reference to a Space to populate controlPlaneName.
	//// Either SpaceName, SpaceRef or SpaceSelector has to be given.
	//// +kubebuilder:validation:Optional
	//SpaceRef *xpv1.Reference `json:"spaceRef,omitempty"`
	//
	//// Selector for a Space to populate controlPlaneName.
	//// Either SpaceName, SpaceRef or SpaceSelector has to be given.
	//// +kubebuilder:validation:Optional
	//SpaceSelector *xpv1.Selector `json:"spacSelector,omitempty"`
}

// ControlPlaneGroupSpec defines the desired state of ControlPlaneGroup.
type ControlPlaneGroupSpec struct {
	xpv1.ResourceSpec `json:",inline"`
	ForProvider       ControlPlaneGroupParameters `json:"forProvider"`
}

// ControlPlaneGroupStatus defines the observed state of the ControlPlaneGroup.
type ControlPlaneGroupStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          ControlPlaneGroupObservation `json:"atProvider,omitempty"`
}

// ControlPlaneGroupObservation are the observable fields of a Repository.
type ControlPlaneGroupObservation struct {
	Name string `json:"name"`
	// it would be nice if we can get the space name/id...
	AccountID uint        `json:"controlplaneGroupName"`
	CreatedAt metav1.Time `json:"createdAt"`
}

var (
	// ControlPlaneGroupGroupKind is the kind of the ControlPlaneGroup.
	ControlPlaneGrpKind                 = reflect.TypeOf(ControlPlaneGroup{}).Name()
	ControlPlaneGrpGroupKind            = schema.GroupKind{Group: Group, Kind: ControlPlaneGrpKind}.String()
	ControlPlaneGrpKindAPIVersion       = ControlPlaneGrpGroupKind + "." + SchemeGroupVersion.String()
	ControlPlaneGrpGroupKindVersionKind = SchemeGroupVersion.WithKind(ControlPlaneGrpKind)
)

func init() {
	SchemeBuilder.Register(&ControlPlaneGroup{}, &ControlPlaneGroupList{})
}
