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

// Owner defines the owner of the token.
type Owner struct {
	// Type of the owner account, like user or organization.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=users;controlPlanes;robots
	// +immutable
	Type string `json:"type"`

	// ID of the owner. It can be UUID or integer. Takes precedence over Name
	// field.
	// +immutable
	ID *string `json:"id,omitempty"`

	// IDRef references a Robot, User or a ControlPlane, depending on value of
	// Type field, to retrieve its ID.
	// +optional
	IDRef *xpv1.Reference `json:"idRef,omitempty"`

	// IDSelector selects a reference to a Robot, User or a ControlPlane,
	// depending on value of Type field, to retrieve its ID.
	// +optional
	IDSelector *xpv1.Selector `json:"idSelector,omitempty"`
}

// TokenParameters are the configurable fields of a Token.
type TokenParameters struct {
	// Name of the Token. This is different from the ID which is assigned by the
	// Upbound API.
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Owner of the Token.
	// +kubebuilder:validation:Required
	Owner Owner `json:"owner"`
}

// TokenObservation are the observable fields of a Token.
type TokenObservation struct{}

// A TokenSpec defines the desired state of a Token.
type TokenSpec struct {
	xpv1.ResourceSpec `json:",inline"`
	ForProvider       TokenParameters `json:"forProvider"`
}

// A TokenStatus represents the observed state of a Token.
type TokenStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          TokenObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true

// A Token is an Upbound token that can be used to access Upbound services.
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,categories={crossplane,managed,upbound}
type Token struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TokenSpec   `json:"spec"`
	Status TokenStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// TokenList contains a list of Token
type TokenList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Token `json:"items"`
}

// Token type metadata.
var (
	TokenKind             = reflect.TypeOf(Token{}).Name()
	TokenGroupKind        = schema.GroupKind{Group: Group, Kind: TokenKind}.String()
	TokenKindAPIVersion   = TokenKind + "." + SchemeGroupVersion.String()
	TokenGroupVersionKind = SchemeGroupVersion.WithKind(TokenKind)
)

func init() {
	SchemeBuilder.Register(&Token{}, &TokenList{})
}
