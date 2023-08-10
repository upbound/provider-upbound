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

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
)

// ControlPlanePermissionParameters are the configurable fields of a ControlPlanePermission.
type ControlPlanePermissionParameters struct {
	// OrganizationName is the name of the organization to which the control plane
	// belongs.
	// +kubebuilder:validation:Required
	OrganizationName string `json:"organizationName"`

	// TeamID is the name of the team the control plane permission will be
	// granted to.
	// +crossplane:generate:reference:type=github.com/upbound/provider-upbound/apis/iam/v1alpha1.Team
	TeamID *string `json:"teamId,omitempty"`

	// TeamIDRef references a Team to retrieve its name to populate TeamID.
	TeamIDRef *xpv1.Reference `json:"teamIdRef,omitempty"`

	// TeamIDSelector selects a reference to a Team to populate TeamIDRef.
	TeamIDSelector *xpv1.Selector `json:"teamIdSelector,omitempty"`

	// ControlPlaneName is the name of the control plane to which the permission
	// will be granted.
	// +crossplane:generate:reference:type=ControlPlane
	ControlPlaneName string `json:"controlPlaneName,omitempty"`

	// ControlPlaneNameRef references a Team to retrieve its name to populate ControlPlaneName.
	ControlPlaneNameRef *xpv1.Reference `json:"controlPlaneNameRef,omitempty"`

	// ControlPlaneNameSelector selects a reference to a Team to populate ControlPlaneNameDRef.
	ControlPlaneNameSelector *xpv1.Selector `json:"controlPlaneNameSelector,omitempty"`

	// Permission is the permission to grant to the team.
	// +kubebuilder:validation:Enum=editor;viewer;owner
	// +kubebuilder:validation:Required
	Permission string `json:"permission"`
}

// ControlPlanePermissionObservation are the observable fields of a ControlPlanePermission.
type ControlPlanePermissionObservation struct {
	// CreatedAt is the time the control plane permission was created.
	CreatedAt *metav1.Time `json:"createdAt,omitempty"`

	// UpdatedAt is the time the control plane permission was last updated.
	UpdatedAt *metav1.Time `json:"updatedAt,omitempty"`

	// AccountID is the ID of the account that the team belongs to, i.e.
	// organization account.
	AccountID uint `json:"accountId,omitempty"`

	// CreatorID is the ID of the user that created the control plane permission.
	CreatorID uint `json:"creatorId,omitempty"`
}

// A ControlPlanePermissionSpec defines the desired state of a ControlPlanePermission.
type ControlPlanePermissionSpec struct {
	xpv1.ResourceSpec `json:",inline"`
	ForProvider       ControlPlanePermissionParameters `json:"forProvider"`
}

// A ControlPlanePermissionStatus represents the observed state of a ControlPlanePermission.
type ControlPlanePermissionStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          ControlPlanePermissionObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true

// A ControlPlanePermission is used to grant control plane permissions to a
// team.
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,categories={crossplane,managed,upbound}
type ControlPlanePermission struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ControlPlanePermissionSpec   `json:"spec"`
	Status ControlPlanePermissionStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ControlPlanePermissionList contains a list of ControlPlanePermission
type ControlPlanePermissionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ControlPlanePermission `json:"items"`
}

// ControlPlanePermission type metadata.
var (
	ControlPlanePermissionKind             = reflect.TypeOf(ControlPlanePermission{}).Name()
	ControlPlanePermissionGroupKind        = schema.GroupKind{Group: Group, Kind: ControlPlanePermissionKind}.String()
	ControlPlanePermissionKindAPIVersion   = ControlPlanePermissionKind + "." + SchemeGroupVersion.String()
	ControlPlanePermissionGroupVersionKind = SchemeGroupVersion.WithKind(ControlPlanePermissionKind)
)

func init() {
	SchemeBuilder.Register(&ControlPlanePermission{}, &ControlPlanePermissionList{})
}
