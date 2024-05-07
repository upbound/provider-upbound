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
	"strconv"
	"strings"

	"github.com/Masterminds/semver"
	"github.com/upbound/up-sdk-go/service/controlplanes"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"context"
	"fmt"
	"net/http"

	"github.com/upbound/up-sdk-go"

	v1alpha1 "github.com/upbound/provider-upbound/apis/mcp/v1alpha1"
)

const basePathFmt = "/v1/controlPlanes/%s/"

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
			Configuration: &v1alpha1.ControlPlaneConfiguration{},
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
	semver1, err1 := semver.NewVersion(version1)
	semver2, err2 := semver.NewVersion(version2)

	if err1 != nil || err2 != nil {
		// Invalid versions, can't compare
		return 0
	}

	if versionCmp := semver1.Compare(semver2); versionCmp != 0 {
		return versionCmp
	}

	// Versions are equal based on semantic versioning, compare the package
	// metadata parts, specifically looking at commit numbers as commit
	// SHAs can change due to rebasing. This is at best a "best effort"
	// approach to derive differences between versions given in the case
	// of rebasing commit numbers could change in addition to SHAs (e.g.
	// dropping commits, etc).
	commitNum1 := getCommitNumber(version1)
	commitNum2 := getCommitNumber(version2)
	return compare(commitNum1, commitNum2)
}

// getCommitNumber will derive the commit number (int) from the given semver
// metadata.
func getCommitNumber(version string) int {
	parts := strings.SplitN(version, "+", 2)
	if len(parts) != 2 {
		return -1
	}
	parts = strings.SplitN(parts[1], ".", 2)
	if len(parts) != 2 {
		return -1
	}
	res, err := strconv.Atoi(parts[0])
	if err != nil {
		return -1
	}
	return res
}

func compare(a, b int) int {
	if a == b {
		return 0
	}
	if a > b {
		return 1
	}
	return -1
}
