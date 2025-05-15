// Copyright 2024 Upbound Inc
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package controlplanegroup

import (
	"context"
	"github.com/upbound/up-sdk-go/service/common"
	"net/http"
	"path"

	"github.com/upbound/up-sdk-go"
	"github.com/upbound/up-sdk-go/apis/spaces/v1beta1"
)

const (
	basePath = "org"
)

// Client is a control planes group client.
type Client struct {
	*up.Config
}

// NewClient build a control planes group client from the passed config.
func NewClient(cfg *up.Config) *Client {
	return &Client{
		cfg,
	}
}

// Create a control plane group on Upbound.
func (c *Client) Create(ctx context.Context, account string, space string, groupName string, params v1beta1.ControlPlane) (*v1beta1.ControlPlane, error) {
	// https://api.upbound.io/org/upbound/space/upbound-gcp-us-west-1/apis/spaces.upbound.io/v1beta1/namespaces/20241203-dalorion-test/controlplanes
	req, err := c.Client.NewRequest(ctx, http.MethodPost, basePath, path.Join(account, "space", space, "apis/spaces.upbound.io/v1beta1/namespaces", groupName, "/controlplanes"), params)
	if err != nil {
		return nil, err
	}
	cp := &v1beta1.ControlPlane{}
	err = c.Client.Do(req, &cp)
	if err != nil {
		return nil, err
	}
	return cp, nil
}

// Get a control plane group on Upbound.
func (c *Client) Get(ctx context.Context, account string, space string, groupName string, name string) (*v1beta1.ControlPlane, error) { // nolint:interfacer
	req, err := c.Client.NewRequest(ctx, http.MethodGet, basePath, path.Join(account, "space", space, "apis/spaces.upbound.io/v1beta1/namespaces", groupName, "/controlplanes", name), nil)
	if err != nil {
		return nil, err
	}
	cp := &v1beta1.ControlPlane{}
	err = c.Client.Do(req, &cp)
	if err != nil {
		return nil, err
	}
	return cp, nil
}

// List all control planes groups in the given account on Upbound.
func (c *Client) List(ctx context.Context, account string, space string, groupName string, opts ...common.ListOption) (*v1beta1.ControlPlaneList, error) {
	req, err := c.Client.NewRequest(ctx, http.MethodGet, basePath, path.Join(account, "space", space, "apis/spaces.upbound.io/v1beta1/namespaces", groupName, "/controlplanes"), nil)
	if err != nil {
		return nil, err
	}
	for _, o := range opts {
		o(req)
	}
	cp := &v1beta1.ControlPlaneList{}
	err = c.Client.Do(req, cp)
	if err != nil {
		return nil, err
	}
	return cp, nil
}

// Delete a control plane on Upbound.
func (c *Client) Delete(ctx context.Context, account string, space string, groupName string, name string) error { // nolint:interfacer
	req, err := c.Client.NewRequest(ctx, http.MethodDelete, basePath, path.Join(account, "space", space, "apis/spaces.upbound.io/v1beta1/namespaces", groupName, "/controlplanes", name), nil)
	if err != nil {
		return err
	}
	return c.Client.Do(req, nil)
}
