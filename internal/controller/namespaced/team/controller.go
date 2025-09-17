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

package team

import (
	"context"
	"fmt"

	v1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	"github.com/pkg/errors"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	uperrors "github.com/upbound/up-sdk-go/errors"
	"github.com/upbound/up-sdk-go/service/accounts"

	iamv1alpha1 "github.com/upbound/provider-upbound/apis/namespaced/iam/v1alpha1"
	upclient "github.com/upbound/provider-upbound/internal/client"
	"github.com/upbound/provider-upbound/internal/client/teams"
	"github.com/upbound/provider-upbound/internal/controller/namespaced/config"
)

const (
	errNotTeam   = "managed resource is not a Team custom resource"
	errNewClient = "cannot create new client"
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
	cr, ok := mg.(*iamv1alpha1.Team)
	if !ok {
		return nil, errors.New(errNotTeam)
	}

	cfg, _, err := upclient.NewConfig(ctx, c.kube, config.GetProviderConfigSpecFn(cr))
	if err != nil {
		return nil, errors.Wrap(err, errNewClient)
	}

	return &external{
		teams:    teams.NewClient(cfg),
		accounts: accounts.NewClient(cfg),
	}, nil
}

func (e *external) Disconnect(_ context.Context) error {
	// If there's nothing special to clean up, just return nil.
	return nil
}

// An ExternalClient observes, then either creates, updates, or deletes an
// external resource to ensure it reflects the managed resource's desired state.
type external struct {
	// A 'client' used to connect to the external resource API. In practice this
	// would be something like an AWS SDK client.
	teams    *teams.Client
	accounts *accounts.Client
}

func (e *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*iamv1alpha1.Team)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotTeam)
	}

	if meta.GetExternalName(cr) == "" {
		return managed.ExternalObservation{}, nil
	}
	_, err := e.teams.Get(ctx, meta.GetExternalName(cr))
	if err != nil {
		return managed.ExternalObservation{}, errors.Wrap(resource.Ignore(uperrors.IsNotFound, err), "failed to get team")
	}
	cr.Status.SetConditions(v1.Available())
	return managed.ExternalObservation{
		ResourceExists: true,
		// Name is not returned in the API.
		ResourceUpToDate: true,
	}, nil
}

func (e *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*iamv1alpha1.Team)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotTeam)
	}
	orgIdInt := ptr.Deref(cr.Spec.ForProvider.OrganizationID, 0)
	if orgIdInt < 0 {
		return managed.ExternalCreation{}, errors.New(fmt.Sprintf("invalid OrganizationID: cannot convert negative int %d to uint", orgIdInt))
	}
	orgId := uint(orgIdInt)
	if orgId == 0 {
		if cr.Spec.ForProvider.OrganizationName == nil {
			return managed.ExternalCreation{}, errors.New("either organizationName or organizationId must be specified")
		}
		o, err := e.accounts.Get(ctx, *cr.Spec.ForProvider.OrganizationName)
		if err != nil {
			return managed.ExternalCreation{}, errors.Wrapf(err, "failed to get account %s", *cr.Spec.ForProvider.OrganizationName)
		}
		if o.Account.Type != "organization" {
			return managed.ExternalCreation{}, errors.Errorf("given account %s is not an organization", *cr.Spec.ForProvider.OrganizationName)
		}
		orgId = o.Organization.ID
	}
	resp, err := e.teams.Create(ctx, &teams.CreateParameters{
		Name:           cr.Spec.ForProvider.Name,
		OrganizationID: orgId,
	})
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, "failed to create team")
	}
	meta.SetExternalName(cr, resp.ID)

	return managed.ExternalCreation{}, nil
}

func (e *external) Update(_ context.Context, _ resource.Managed) (managed.ExternalUpdate, error) {
	return managed.ExternalUpdate{}, nil
}

func (e *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	return managed.ExternalDelete{}, errors.Wrap(resource.Ignore(uperrors.IsNotFound, e.teams.Delete(ctx, meta.GetExternalName(mg))), "failed to delete team")
}
