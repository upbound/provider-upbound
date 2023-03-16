/*
Copyright 2022 The Crossplane Authors.

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

type RobotOwner struct {
	// NOTE(muvaf): We don't have the Type field because there is only a single
	// possible value for that field, which is organization.

	// ID of the organization that owns this robot. Takes precedence over name.
	// Either name or id is required.
	ID *string `json:"id,omitempty"`

	// Name of the organization that owns this robot. Either name or id is required.
	// It is used to look up the ID of the organization. Id field takes precedence.
	Name *string `json:"name,omitempty"`
}

// RobotParameters are the configurable fields of a Robot.
type RobotParameters struct {
	// Name of this Robot.
	Name string `json:"name"`

	// Description for this Robot.
	Description *string `json:"description,omitempty"`

	// Organization that owns this robot.
	Organization RobotOwner `json:"organization"`
}

// RobotObservation are the observable fields of a Robot.
type RobotObservation struct {
	ID string `json:"id"`
}

// A RobotSpec defines the desired state of a Robot.
type RobotSpec struct {
	xpv1.ResourceSpec `json:",inline"`
	ForProvider       RobotParameters `json:"forProvider"`
}

// A RobotStatus represents the observed state of a Robot.
type RobotStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          RobotObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true

// A Robot is an example API type.
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,categories={crossplane,managed,upbound}
type Robot struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RobotSpec   `json:"spec"`
	Status RobotStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// RobotList contains a list of Robot
type RobotList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Robot `json:"items"`
}

// Robot type metadata.
var (
	RobotKind             = reflect.TypeOf(Robot{}).Name()
	RobotGroupKind        = schema.GroupKind{Group: Group, Kind: RobotKind}.String()
	RobotKindAPIVersion   = RobotKind + "." + SchemeGroupVersion.String()
	RobotGroupVersionKind = SchemeGroupVersion.WithKind(RobotKind)
)

func init() {
	SchemeBuilder.Register(&Robot{}, &RobotList{})
}
