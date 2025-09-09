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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	xpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
)

// TeamParameters are the configurable fields of a Team.
type TeamParameters struct {
	// Name of the Team. This is different from the ID which is assigned by the
	// Upbound API.
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// OrganizationID of the Team. Takes precedence over the OrganizationName
	// field. Either one of them must be specified.
	OrganizationID *int `json:"organizationId,omitempty"`

	// OrganizationName of the Team. It is used to lookup the OrganizationID.
	// OrganizationID takes precedence over this field. Either one of them must
	// be specified.
	OrganizationName *string `json:"organizationName,omitempty"`
}

// TeamObservation are the observable fields of a Team.
type TeamObservation struct {
	// ID of the Team.
	ID string `json:"id,omitempty"`
}

// A TeamSpec defines the desired state of a Team.
type TeamSpec struct {
	xpv1.ResourceSpec `json:",inline"`
	ForProvider       TeamParameters `json:"forProvider"`
}

// A TeamStatus represents the observed state of a Team.
type TeamStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          TeamObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true

// A Team is an Upbound team that can be used to access Upbound services.
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,categories={crossplane,managed,upbound}
type Team struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TeamSpec   `json:"spec"`
	Status TeamStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// TeamList contains a list of Team
type TeamList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Team `json:"items"`
}

// Team type metadata.
var (
	TeamKind             = reflect.TypeOf(Team{}).Name()
	TeamGroupKind        = schema.GroupKind{Group: Group, Kind: TeamKind}.String()
	TeamKindAPIVersion   = TeamKind + "." + SchemeGroupVersion.String()
	TeamGroupVersionKind = SchemeGroupVersion.WithKind(TeamKind)
)

func init() {
	SchemeBuilder.Register(&Team{}, &TeamList{})
}
