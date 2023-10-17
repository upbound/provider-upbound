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
	"regexp"

	"github.com/upbound/up-sdk-go/service/controlplanes"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"context"
	"fmt"
	"net/http"

	"github.com/upbound/up-sdk-go"

	v1alpha1 "github.com/upbound/provider-upbound/apis/mcp/v1alpha1"
)

const (
	basePathFmt = "/v1/controlPlanes/%s/"
)

func NewClient(cfg *up.Config) *Client {
	return &Client{
		Config: cfg,
	}
}

type Client struct {
	*up.Config
}

func (c *Client) Apply(ctx context.Context, params *ApplyParameters) error {
	patches := &ControlPlanePatch{
		Patches: []SetDesiredVersion{
			{
				SetDesiredVersion: params.DesiredVersion,
			},
		},
	}

	req, err := c.Client.NewRequest(ctx, http.MethodPatch, fmt.Sprintf(basePathFmt, params.Organization), params.Name, patches)
	if err != nil {
		return err
	}
	if err := c.Client.Do(req, nil); err != nil {
		return err
	}
	return nil
}

// GetSecretValue fetches the referenced input secret key reference
func StatusFromResponse(resp *controlplanes.ControlPlaneResponse, latestAvailableVersion *string) v1alpha1.ControlPlaneResponse {

	status := v1alpha1.ControlPlaneResponse{
		ControlPlane: v1alpha1.ControlPlaneObs{
			Configuration: v1alpha1.ControlPlaneConfiguration{},
		},
	}

	if latestAvailableVersion != nil {
		status.ControlPlane.Configuration.LatestAvailableVersion = latestAvailableVersion
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

// CompareVersions checks if current and desired version for configurations matches
func CompareVersions(version1, version2 string) int { //nolint:gocyclo
	re := regexp.MustCompile(`v(\d+)\.(\d+)\.(\d+)\+(\d+)`)

	matches1 := re.FindStringSubmatch(version1)
	matches2 := re.FindStringSubmatch(version2)

	if len(matches1) == 0 || len(matches2) == 0 {
		// Invalid versions, can't compare
		return 0
	}

	// Extract version components
	major1 := parseVersionPart(matches1[1])
	minor1 := parseVersionPart(matches1[2])
	patch1 := parseVersionPart(matches1[3])
	numericPart1 := parseVersionPart(matches1[4])

	major2 := parseVersionPart(matches2[1])
	minor2 := parseVersionPart(matches2[2])
	patch2 := parseVersionPart(matches2[3])
	numericPart2 := parseVersionPart(matches2[4])

	// Compare version components
	if major1 > major2 {
		return 1
	} else if major1 < major2 {
		return -1
	}

	if minor1 > minor2 {
		return 1
	} else if minor1 < minor2 {
		return -1
	}

	if patch1 > patch2 {
		return 1
	} else if patch1 < patch2 {
		return -1
	}

	if numericPart1 > numericPart2 {
		return 1
	} else if numericPart1 < numericPart2 {
		return -1
	}

	return 0
}

func parseVersionPart(part string) int {
	var num int
	_, err := fmt.Sscanf(part, "%d", &num)
	if err != nil {
		num = 0
	}
	return num
}
