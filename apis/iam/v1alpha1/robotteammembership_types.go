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

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// RobotTeamMembershipParameters are the configurable fields of a RobotTeamMembership.
type RobotTeamMembershipParameters struct {
	// RobotID of the robot to add to the team. Either robotId or robotIdRef or
	// robotIdSelector is required.
	// +crossplane:generate:reference:type=Robot
	RobotID *string `json:"robotId,omitempty"`

	// RobotIDRef references a Robot to and retrieves its robotId.
	RobotIDRef *xpv1.Reference `json:"robotIdRef,omitempty"`

	// RobotIDSelector selects a reference to a Robot in order to retrieve its
	// robotId.
	RobotIDSelector *xpv1.Selector `json:"robotIdSelector,omitempty"`

	// TeamID of the team to add the robot to. Either teamId or teamIdRef or
	// teamIdSelector is required.
	// +crossplane:generate:reference:type=Team
	TeamID *string `json:"teamId,omitempty"`

	// TeamIDRef references a Team to and retrieves its teamId.
	TeamIDRef *xpv1.Reference `json:"teamIdRef,omitempty"`

	// TeamIDSelector selects a reference to a Team in order to retrieve its
	// teamId.
	TeamIDSelector *xpv1.Selector `json:"teamIdSelector,omitempty"`
}

// RobotTeamMembershipObservation are the observable fields of a RobotTeamMembership.
type RobotTeamMembershipObservation struct{}

// A RobotTeamMembershipSpec defines the desired state of a RobotTeamMembership.
type RobotTeamMembershipSpec struct {
	xpv1.ResourceSpec `json:",inline"`
	ForProvider       RobotTeamMembershipParameters `json:"forProvider"`
}

// A RobotTeamMembershipStatus represents the observed state of a RobotTeamMembership.
type RobotTeamMembershipStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          RobotTeamMembershipObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true

// A RobotTeamMembership is an example API type.
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,categories={crossplane,managed,upbound}
type RobotTeamMembership struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RobotTeamMembershipSpec   `json:"spec"`
	Status RobotTeamMembershipStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// RobotTeamMembershipList contains a list of RobotTeamMembership
type RobotTeamMembershipList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RobotTeamMembership `json:"items"`
}

// RobotTeamMembership type metadata.
var (
	RobotTeamMembershipKind             = reflect.TypeOf(RobotTeamMembership{}).Name()
	RobotTeamMembershipGroupKind        = schema.GroupKind{Group: Group, Kind: RobotTeamMembershipKind}.String()
	RobotTeamMembershipKindAPIVersion   = RobotTeamMembershipKind + "." + SchemeGroupVersion.String()
	RobotTeamMembershipGroupVersionKind = SchemeGroupVersion.WithKind(RobotTeamMembershipKind)
)

func init() {
	SchemeBuilder.Register(&RobotTeamMembership{}, &RobotTeamMembershipList{})
}
