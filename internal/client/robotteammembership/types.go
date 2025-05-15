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

package robotteammembership

// Robot scope types.
const (
	RobotMembershipTypeTeam string = "teams"
)

// A RelationshipList represents JSON API relationships.
// https://jsonapi.org/format/#document-resource-object-relationships
type RelationshipList struct {
	Data []ResourceIdentifier `json:"data"`
}

type ResourceIdentifier struct {
	Type string `json:"type"`
	ID   string `json:"id"`
}

type DeleteParameters struct {
	Type string `json:"type"`
	ID   string `json:"id"`
}
