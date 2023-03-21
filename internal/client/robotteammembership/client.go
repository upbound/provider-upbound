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

package robotteammembership

import (
	"context"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/pkg/errors"

	"github.com/upbound/up-sdk-go"
	uperrors "github.com/upbound/up-sdk-go/errors"
	"github.com/upbound/up-sdk-go/service/robots"
)

const (
	basePathFmt = "v1/robots/%s/relationships/teams"
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

func (c *Client) Get(ctx context.Context, robotId, teamId string) error {
	rid, err := uuid.Parse(robotId)
	if err != nil {
		return errors.Wrapf(err, "failed to parse robot id %s as uuid", robotId)
	}
	resp, err := c.robotClient.Get(ctx, rid)
	if err != nil {
		return errors.Wrapf(err, "failed to get robot with id %s", robotId)
	}
	teams := resp.DataSet.RelationshipSet["teams"].([]string)
	for _, team := range teams {
		if team == teamId {
			return nil
		}
	}
	return &uperrors.Error{Status: http.StatusNotFound}
}

func (c *Client) Create(ctx context.Context, robotId string, params *CreateParameters) error {
	req, err := c.Client.NewRequest(ctx, http.MethodPost, fmt.Sprintf(basePathFmt, robotId), "", params)
	if err != nil {
		return err
	}
	return c.Client.Do(req, nil)
}

func (c *Client) Delete(ctx context.Context, robotId string, params *DeleteParameters) error {
	req, err := c.Client.NewRequest(ctx, http.MethodDelete, fmt.Sprintf(basePathFmt, robotId), "", params)
	if err != nil {
		return err
	}
	return c.Client.Do(req, nil)
}
