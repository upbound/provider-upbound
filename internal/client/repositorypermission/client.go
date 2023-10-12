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

package repositorypermission

import (
	"context"
	"fmt"
	"net/http"

	"github.com/upbound/up-sdk-go"
)

const (
	basePathFmt = "/v1/repoPermissions/%s/teams/%s"
)

func NewClient(cfg *up.Config) *Client {
	return &Client{
		Config: cfg,
	}
}

type Client struct {
	*up.Config
}

func (c *Client) Get(ctx context.Context, params *GetParameters) error {
	req, err := c.Client.NewRequest(ctx, http.MethodGet, fmt.Sprintf(basePathFmt, params.Organization, params.TeamID), params.Repository, nil)
	if err != nil {
		return err
	}

	if err := c.Client.Do(req, nil); err != nil {
		return err
	}
	return nil
}

func (c *Client) Create(ctx context.Context, params *CreateParameters) error {
	req, err := c.Client.NewRequest(ctx, http.MethodPut, fmt.Sprintf(basePathFmt, params.Organization, params.TeamID), params.Repository, &SetPermission{
		Permission: params.Permission,
	})
	if err != nil {
		return err
	}
	if err := c.Client.Do(req, nil); err != nil {
		return err
	}
	return nil
}

func (c *Client) Delete(ctx context.Context, params *GetParameters) error {
	req, err := c.Client.NewRequest(ctx, http.MethodDelete, fmt.Sprintf(basePathFmt, params.Organization, params.TeamID), params.Repository, nil)
	if err != nil {
		return err
	}
	return c.Client.Do(req, nil)
}
