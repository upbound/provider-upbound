/*
Copyright 2025 Upbound Inc.

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

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// PermissionParameters are the configurable fields of a Permission.
type PermissionParameters struct {
	// OrganizationName is the name of the organization to which the Permission
	// belongs.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	OrganizationName string `json:"organizationName"`

	// Permission is the permission to grant to the repository for an team.
	// +kubebuilder:validation:Enum=admin;read;write;view
	// +kubebuilder:validation:Required
	Permission string `json:"permission"`

	// TeamID of the team to add the robot to. Either teamId or teamIdRef or
	// teamIdSelector is required.
	// +crossplane:generate:reference:type=github.com/upbound/provider-upbound/apis/iam/v1alpha1.Team
	TeamID *string `json:"teamId,omitempty"`

	// TeamIDRef references a Team to and retrieves its teamId.
	TeamIDRef *xpv1.Reference `json:"teamIdRef,omitempty"`

	// TeamIDSelector selects a reference to a Team in order to retrieve its
	// teamId.
	TeamIDSelector *xpv1.Selector `json:"teamIdSelector,omitempty"`

	// Repository of the repository to add the permission to. Either repository or repositoryRef or
	// repositorySelector is required.
	// +crossplane:generate:reference:type=Repository
	Repository *string `json:"repository,omitempty"`

	// RepositoryRef references a Repository to and retrieves its name.
	RepositoryRef *xpv1.Reference `json:"repositoryRef,omitempty"`

	// RepositorySelector selects a reference to a Repository in order to retrieve its
	// name.
	RepositorySelector *xpv1.Selector `json:"repositorySelector,omitempty"`
}

// PermissionObservation are the observable fields of a Permission.
type PermissionObservation struct{}

// A PermissionSpec defines the desired state of a Permission.
type PermissionSpec struct {
	xpv1.ResourceSpec `json:",inline"`
	ForProvider       PermissionParameters `json:"forProvider"`
}

// A PermissionStatus represents the observed state of a Permission.
type PermissionStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          PermissionObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true

// A Permission is an API type.
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,categories={crossplane,managed,upbound}
type Permission struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PermissionSpec   `json:"spec"`
	Status PermissionStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// PermissionList contains a list of Permission
type PermissionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Permission `json:"items"`
}

// Permission type metadata.
var (
	PermissionKind             = reflect.TypeOf(Permission{}).Name()
	PermissionGroupKind        = schema.GroupKind{Group: Group, Kind: PermissionKind}.String()
	PermissionKindAPIVersion   = PermissionKind + "." + SchemeGroupVersion.String()
	PermissionGroupVersionKind = SchemeGroupVersion.WithKind(PermissionKind)
)

func init() {
	SchemeBuilder.Register(&Permission{}, &PermissionList{})
}
