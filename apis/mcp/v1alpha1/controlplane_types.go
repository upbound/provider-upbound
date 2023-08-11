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

// ControlPlaneParameters are the configurable fields of a ControlPlane.
type ControlPlaneParameters struct {
	// Description is the description of the the control plane
	Description *string `json:"description,omitempty"`

	// Configuration is the name of the predefined configuration
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Configuration string `json:"configuration"`

	// OrganizationName is the name of the organization to which the control plane
	// belongs.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	OrganizationName string `json:"organizationName"`
}

// A ControlPlaneSpec defines the desired state of a ControlPlane.
type ControlPlaneSpec struct {
	xpv1.ResourceSpec `json:",inline"`
	ForProvider       ControlPlaneParameters `json:"forProvider"`
}

// A ControlPlaneStatus represents the observed state of a ControlPlane.
type ControlPlaneStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          ControlPlaneResponse `json:"atProvider,omitempty"`
}

// ControlPlaneResponse is the HTTP body returned by the Upbound API when
// fetching control planes.
type ControlPlaneResponse struct {
	ControlPlane ControlPlaneObs `json:"controlPlane"`
	Status       Status          `json:"controlPlanestatus,omitempty"`
	Permission   PermissionGroup `json:"controlPlanePermission,omitempty"`
}

// ControlPlane describes a control plane.
type ControlPlaneObs struct {
	ID            string                    `json:"id,omitempty"`
	Name          string                    `json:"name,omitempty"`
	Description   string                    `json:"description,omitempty"`
	CreatorID     uint                      `json:"creatorId,omitempty"`
	Reserved      bool                      `json:"reserved"`
	CreatedAt     *metav1.Time              `json:"createdAt,omitempty"`
	UpdatedAt     *metav1.Time              `json:"updatedAt,omitempty"`
	ExpiresAt     metav1.Time               `json:"expiresAt,omitempty"`
	Configuration ControlPlaneConfiguration `json:"configuration"`
}

// ControlPlaneConfiguration represents an instance of a Configuration associated with a
// Managed Control Plane on Upbound.
type ControlPlaneConfiguration struct {
	ID             string              `json:"id"`
	Name           *string             `json:"name,omitempty"`
	CurrentVersion *string             `json:"currentVersion,omitempty"`
	DesiredVersion *string             `json:"desiredVersion,omitempty"`
	Status         ConfigurationStatus `json:"status"`
	SyncedAt       *metav1.Time        `json:"syncedAt,omitempty"`
	DeployedAt     *metav1.Time        `json:"deployedAt,omitempty"`
}

// ConfigurationStatus represents the different states of a Configuration relative to a Managed Control Plane.
type ConfigurationStatus string

const (
	// ConfigurationInstallationQueued means queued to begin installation in a Managed Control Plane
	ConfigurationInstallationQueued ConfigurationStatus = "installationQueued"
	// ConfigurationUpgradeQueued means queued to upgrade to a specified version in a Managed Control Plane
	ConfigurationUpgradeQueued ConfigurationStatus = "upgradeQueued"
	// ConfigurationInstalling means currently installing into the Managed Control Plane
	ConfigurationInstalling ConfigurationStatus = "installing"
	// ConfigurationReady means ready for use in the Managed Control Plane
	ConfigurationReady ConfigurationStatus = "ready"
	// ConfigurationUpgrading means currently upgrading to a specified version in the Managed Control Plane
	ConfigurationUpgrading ConfigurationStatus = "upgrading"
)

// Status is the status of a control plane on Upbound.
type Status string

// A control plane will always be in one of the following phases.
const (
	StatusProvisioning Status = "provisioning"
	StatusUpdating     Status = "updating"
	StatusReady        Status = "ready"
	StatusDeleting     Status = "deleting"
)

// +kubebuilder:object:root=true

// A ControlPlane is used to create a controlplane
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="ID",type="string",JSONPath=".status.atProvider.controlPlane.id"
// +kubebuilder:printcolumn:name="DEPLOYED-CONFIGURATION",type="string",JSONPath=".status.atProvider.controlPlane.configuration.name"
// +kubebuilder:printcolumn:name="CONFIGURATION-STATUS",type="string",JSONPath=".status.atProvider.controlPlane.configuration.status"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,categories={crossplane,managed,upbound}
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

// ControlPlane type metadata.
var (
	ControlPlaneKind             = reflect.TypeOf(ControlPlane{}).Name()
	ControlPlaneGroupKind        = schema.GroupKind{Group: Group, Kind: ControlPlaneKind}.String()
	ControlPlaneKindAPIVersion   = ControlPlaneKind + "." + SchemeGroupVersion.String()
	ControlPlaneGroupVersionKind = SchemeGroupVersion.WithKind(ControlPlaneKind)
)

func init() {
	SchemeBuilder.Register(&ControlPlane{}, &ControlPlaneList{})
}
