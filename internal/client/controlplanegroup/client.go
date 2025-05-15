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
	v1 "k8s.io/api/core/v1"
	"net/http"
	"path"

	"github.com/upbound/up-sdk-go"
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
func (c *Client) Create(ctx context.Context, account string, space string, params *v1.Namespace) (*v1.Namespace, error) {
	// https://api.upbound.io/org/upbound/space/upbound-gcp-us-west-1/api/v1/namespaces/
	req, err := c.Client.NewRequest(ctx, http.MethodPost, basePath, path.Join(account, "space", space, "api/v1/namespaces"), params)
	if err != nil {
		return nil, err
	}
	cp := &v1.Namespace{}
	err = c.Client.Do(req, &cp)
	if err != nil {
		return nil, err
	}
	return cp, nil
}

// Get a control plane group on Upbound.
func (c *Client) Get(ctx context.Context, account string, space string, name string) (*v1.Namespace, error) { // nolint:interfacer
	req, err := c.Client.NewRequest(ctx, http.MethodGet, basePath, path.Join(account, "space", space, "api/v1/namespaces", name), nil)
	if err != nil {
		return nil, err
	}
	cp := &v1.Namespace{}
	err = c.Client.Do(req, &cp)
	if err != nil {
		return nil, err
	}
	return cp, nil
}

// List all control planes groups in the given account on Upbound.
func (c *Client) List(ctx context.Context, account string, space string, opts ...common.ListOption) (*v1.NamespaceList, error) {
	req, err := c.Client.NewRequest(ctx, http.MethodGet, basePath, path.Join(account, "space", space, "api/v1/namespaces"), nil)
	if err != nil {
		return nil, err
	}
	for _, o := range opts {
		o(req)
	}
	cp := &v1.NamespaceList{}
	err = c.Client.Do(req, cp)
	if err != nil {
		return nil, err
	}
	return cp, nil
}

// Delete a control plane on Upbound.
func (c *Client) Delete(ctx context.Context, account string, space string, name string) error { // nolint:interfacer
	req, err := c.Client.NewRequest(ctx, http.MethodDelete, basePath, path.Join(account, "space", space, "api/v1/namespaces", name), nil)
	if err != nil {
		return err
	}
	return c.Client.Do(req, nil)
}
