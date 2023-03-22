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

package controlplanepermission

import (
	"context"
	"fmt"
	"net/http"

	"k8s.io/utils/pointer"

	"github.com/upbound/up-sdk-go"
	uperrors "github.com/upbound/up-sdk-go/errors"
	"github.com/upbound/up-sdk-go/service/robots"
)

const (
	// account_name, team_name, control_plane_name
	basePathFmt = "v1/controlPlanePermissions/%s/teams/%s"
)

func NewClient(cfg *up.Config) *Client {
	return &Client{
		Config:      cfg,
		robotClient: robots.NewClient(cfg),
	}
}

type Client struct {
	*up.Config
	robotClient *robots.Client
}

func (c *Client) Get(ctx context.Context, params *GetParameters) (*PermissionResponse, error) {
	req, err := c.Client.NewRequest(ctx, http.MethodGet, fmt.Sprintf(basePathFmt, params.AccountName, params.TeamID), "", nil)
	if err != nil {
		return nil, err
	}
	perms := &GetResponse{}
	if err := c.Client.Do(req, perms); err != nil {
		return nil, err
	}
	for i := range perms.Permissions {
		if perms.Permissions[i].TeamID == params.TeamID {
			return &perms.Permissions[i], nil
		}
	}
	return nil, &uperrors.Error{Status: http.StatusNotFound, Title: "NotFound", Detail: pointer.String("permission not found")}
}

func (c *Client) Apply(ctx context.Context, params *ApplyParameters) error {
	req, err := c.Client.NewRequest(ctx, http.MethodPut, fmt.Sprintf(basePathFmt, params.AccountName, params.TeamID), params.ControlPlaneName, params)
	if err != nil {
		return err
	}
	return c.Client.Do(req, nil)
}

func (c *Client) Delete(ctx context.Context, params *DeleteParameters) error {
	req, err := c.Client.NewRequest(ctx, http.MethodDelete, fmt.Sprintf(basePathFmt, params.AccountName, params.TeamID), params.ControlPlaneName, nil)
	if err != nil {
		return err
	}
	return c.Client.Do(req, nil)
}
