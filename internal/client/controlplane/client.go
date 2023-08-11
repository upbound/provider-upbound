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

package controlplane

import (
	"github.com/upbound/up-sdk-go/service/controlplanes"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1alpha1 "github.com/upbound/provider-upbound/apis/mcp/v1alpha1"
)

// GetSecretValue fetches the referenced input secret key reference
func StatusFromResponse(resp *controlplanes.ControlPlaneResponse) v1alpha1.ControlPlaneResponse {

	status := v1alpha1.ControlPlaneResponse{
		ControlPlane: v1alpha1.ControlPlaneObs{
			Configuration: v1alpha1.ControlPlaneConfiguration{},
		},
	}

	status.ControlPlane.ID = resp.ControlPlane.ID.String()
	status.ControlPlane.Name = resp.ControlPlane.Name
	status.ControlPlane.Description = resp.ControlPlane.Description
	status.ControlPlane.CreatorID = resp.ControlPlane.CreatorID
	status.ControlPlane.Reserved = resp.ControlPlane.Reserved
	status.ControlPlane.ExpiresAt = metav1.Time{Time: resp.ControlPlane.ExpiresAt}
	status.ControlPlane.Configuration.ID = resp.ControlPlane.Configuration.ID.String()
	status.ControlPlane.Configuration.Status = v1alpha1.ConfigurationStatus(resp.ControlPlane.Configuration.Status)
	status.Status = v1alpha1.Status(resp.Status)
	status.Permission = v1alpha1.PermissionGroup(resp.Permission)

	if resp.ControlPlane.CreatedAt != nil {
		status.ControlPlane.CreatedAt = &metav1.Time{Time: *resp.ControlPlane.CreatedAt}
	}

	if resp.ControlPlane.UpdatedAt != nil {
		status.ControlPlane.UpdatedAt = &metav1.Time{Time: *resp.ControlPlane.UpdatedAt}
	}

	if resp.ControlPlane.Configuration.Name != nil {
		status.ControlPlane.Configuration.Name = resp.ControlPlane.Configuration.Name
	}

	if resp.ControlPlane.Configuration.DesiredVersion != nil {
		status.ControlPlane.Configuration.DesiredVersion = resp.ControlPlane.Configuration.DesiredVersion
	}

	if resp.ControlPlane.Configuration.CurrentVersion != nil {
		status.ControlPlane.Configuration.CurrentVersion = resp.ControlPlane.Configuration.CurrentVersion
	}

	if resp.ControlPlane.Configuration.SyncedAt != nil {
		status.ControlPlane.Configuration.SyncedAt = &metav1.Time{Time: *resp.ControlPlane.Configuration.SyncedAt}
	}

	if resp.ControlPlane.Configuration.DeployedAt != nil {
		status.ControlPlane.Configuration.DeployedAt = &metav1.Time{Time: *resp.ControlPlane.Configuration.DeployedAt}
	}

	return status
}
