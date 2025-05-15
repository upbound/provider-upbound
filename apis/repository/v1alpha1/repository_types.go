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
	sdk "github.com/upbound/up-sdk-go/service/repositories"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// RepositoryParameters are the configurable fields of a Repository.
type RepositoryParameters struct {
	// Name of this Repository.
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// OrganizationName is the name of the organization to which the repository
	// belongs.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	OrganizationName string `json:"organizationName"`

	// Public determines the visibility of the repository
	// +kubebuilder:validation:Required
	// +kubebuilder:default=false
	Public bool `json:"public"`

	// Publish enables Upbound Marketplace listing page for the new repository
	// +kubebuilder:validation:Required
	// +kubebuilder:default=false
	Publish bool `json:"publish"`
}

// RepositoryObservation are the observable fields of a Repository.
type RepositoryObservation struct {
	Name           string              `json:"name"`
	RepositoryID   uint                `json:"repositoryId"`
	AccountID      uint                `json:"accountId"`
	Type           *sdk.RepositoryType `json:"type,omitempty"`
	Public         bool                `json:"public"`
	Publish        *sdk.PublishPolicy  `json:"publishPolicy"`
	Official       bool                `json:"official"`
	CurrentVersion *string             `json:"currentVersion,omitempty"`
	CreatedAt      metav1.Time         `json:"createdAt"`
	UpdatedAt      *metav1.Time        `json:"updatedAt,omitempty"`
}

// A RepositorySpec defines the desired state of a Repository.
type RepositorySpec struct {
	xpv1.ResourceSpec `json:",inline"`
	ForProvider       RepositoryParameters `json:"forProvider"`
}

// A RepositoryStatus represents the observed state of a Repository.
type RepositoryStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          RepositoryObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true

// A Repository is an API type.
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,categories={crossplane,managed,upbound}
type Repository struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RepositorySpec   `json:"spec"`
	Status RepositoryStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// RepositoryList contains a list of Repository
type RepositoryList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Repository `json:"items"`
}

// Repository type metadata.
var (
	RepositoryKind             = reflect.TypeOf(Repository{}).Name()
	RepositoryGroupKind        = schema.GroupKind{Group: Group, Kind: RepositoryKind}.String()
	RepositoryKindAPIVersion   = RepositoryKind + "." + SchemeGroupVersion.String()
	RepositoryGroupVersionKind = SchemeGroupVersion.WithKind(RepositoryKind)
)

func init() {
	SchemeBuilder.Register(&Repository{}, &RepositoryList{})
}
