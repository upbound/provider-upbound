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

package robot

import (
	"context"
	"strconv"

	v1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	uperrors "github.com/upbound/up-sdk-go/errors"
	"github.com/upbound/up-sdk-go/service/organizations"
	"github.com/upbound/up-sdk-go/service/robots"

	iamv1alpha1 "github.com/upbound/provider-upbound/apis/namespaced/iam/v1alpha1"
	upclient "github.com/upbound/provider-upbound/internal/client"
	"github.com/upbound/provider-upbound/internal/controller/namespaced/config"
)

const (
	errNotRobot  = "managed resource is not a Robot custom resource"
	errNewClient = "cannot create new Service"
)

// A connector is expected to produce an ExternalClient when its Connect method
// is called.
type connector struct {
	kube client.Client
}

// Connect typically produces an ExternalClient by:
// 1. Tracking that the managed resource is using a ProviderConfig.
// 2. Getting the managed resource's ProviderConfig.
// 3. Getting the credentials specified by the ProviderConfig.
// 4. Using the credentials to form a client.
func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	cr, ok := mg.(*iamv1alpha1.Robot)
	if !ok {
		return nil, errors.New(errNotRobot)
	}

	cfg, _, err := upclient.NewConfig(ctx, c.kube, config.GetProviderConfigSpecFn(cr))
	if err != nil {
		return nil, errors.Wrap(err, errNewClient)
	}

	return &external{
		robots:        robots.NewClient(cfg),
		organizations: organizations.NewClient(cfg),
	}, nil
}

func (e *external) Disconnect(_ context.Context) error {
	// If there's nothing special to clean up, just return nil.
	return nil
}

// An ExternalClient observes, then either creates, updates, or deletes an
// external resource to ensure it reflects the managed resource's desired state.
type external struct {
	robots        *robots.Client
	organizations *organizations.Client
}

func (e *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*iamv1alpha1.Robot)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotRobot)
	}

	if meta.GetExternalName(cr) == "" {
		return managed.ExternalObservation{}, nil
	}
	id, err := uuid.Parse(meta.GetExternalName(cr))
	if err != nil {
		return managed.ExternalObservation{}, errors.Wrap(err, "cannot parse external name as a uuid")
	}
	resp, err := e.robots.Get(ctx, id)
	if err != nil {
		return managed.ExternalObservation{}, errors.Wrap(resource.Ignore(uperrors.IsNotFound, err), "cannot get robot")
	}
	cr.Status.SetConditions(v1.Available())
	cr.Status.AtProvider.ID = resp.ID.String()

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: true,
	}, nil
}

func (e *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*iamv1alpha1.Robot)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotRobot)
	}
	id := ptr.Deref(cr.Spec.ForProvider.Owner.ID, "")
	if id == "" {
		if cr.Spec.ForProvider.Owner.Name == nil {
			return managed.ExternalCreation{}, errors.New("organization name or id must be specified")
		}
		o, err := e.organizations.GetOrgID(ctx, *cr.Spec.ForProvider.Owner.Name)
		if err != nil {
			return managed.ExternalCreation{}, errors.Wrap(err, "cannot get organization id")
		}
		id = strconv.FormatUint(uint64(o), 10)
	}

	resp, err := e.robots.Create(ctx, &robots.RobotCreateParameters{
		Attributes: robots.RobotAttributes{
			Name:        cr.Spec.ForProvider.Name,
			Description: cr.Spec.ForProvider.Description,
		},
		Relationships: robots.RobotRelationships{
			Owner: robots.RobotOwner{
				Data: robots.RobotOwnerData{
					Type: robots.RobotOwnerOrganization,
					ID:   id,
				},
			},
		},
	})
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, "cannot create robot")
	}
	meta.SetExternalName(cr, resp.ID.String())
	return managed.ExternalCreation{}, nil
}

func (e *external) Update(_ context.Context, _ resource.Managed) (managed.ExternalUpdate, error) {
	// There is no Update endpoints in robots API.
	return managed.ExternalUpdate{}, nil
}

func (e *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*iamv1alpha1.Robot)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotRobot)
	}

	id, err := uuid.Parse(meta.GetExternalName(cr))
	if err != nil {
		return managed.ExternalDelete{}, errors.Wrap(err, "cannot parse external name as a uuid")
	}
	return managed.ExternalDelete{}, errors.Wrap(resource.Ignore(uperrors.IsNotFound, e.robots.Delete(ctx, id)), "cannot delete robot")
}
