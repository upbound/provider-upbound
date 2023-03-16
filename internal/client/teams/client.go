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

package teams

import (
	"context"
	"net/http"

	"github.com/upbound/up-sdk-go"
)

const (
	basePath = "v1/teams"
)

func NewClient(cfg *up.Config) *Client {
	return &Client{
		Config: cfg,
	}
}

type Client struct {
	*up.Config
}

func (c *Client) Get(ctx context.Context, id string) (*GetResponse, error) {
	req, err := c.Client.NewRequest(ctx, http.MethodGet, basePath, id, nil)
	if err != nil {
		return nil, err
	}
	resp := &GetResponse{}
	if err := c.Client.Do(req, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *Client) Create(ctx context.Context, params *CreateParameters) (*CreateResponse, error) {
	req, err := c.Client.NewRequest(ctx, http.MethodPost, basePath, "", params)
	if err != nil {
		return nil, err
	}
	resp := &CreateResponse{}
	if err := c.Client.Do(req, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *Client) Delete(ctx context.Context, id string) error {
	req, err := c.Client.NewRequest(ctx, http.MethodDelete, basePath, id, nil)
	if err != nil {
		return err
	}
	return c.Client.Do(req, nil)
}
