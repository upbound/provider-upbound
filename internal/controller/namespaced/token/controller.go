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

package token

import (
	"context"
	"fmt"

	v1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	uperrors "github.com/upbound/up-sdk-go/errors"
	"github.com/upbound/up-sdk-go/service/accounts"
	"github.com/upbound/up-sdk-go/service/robots"
	"github.com/upbound/up-sdk-go/service/tokens"

	iamv1alpha1 "github.com/upbound/provider-upbound/apis/namespaced/iam/v1alpha1"
	upclient "github.com/upbound/provider-upbound/internal/client"
	"github.com/upbound/provider-upbound/internal/controller/namespaced/config"
)

const (
	errNotToken  = "managed resource is not a Token custom resource"
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
	cr, ok := mg.(*iamv1alpha1.Token)
	if !ok {
		return nil, errors.New(errNotToken)
	}

	cfg, _, err := upclient.NewConfig(ctx, c.kube, config.GetProviderConfigSpecFn(cr))
	if err != nil {
		return nil, errors.Wrap(err, errNewClient)
	}

	return &external{
		tokens:   tokens.NewClient(cfg),
		accounts: accounts.NewClient(cfg),
		robots:   robots.NewClient(cfg),
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
	tokens   *tokens.Client
	accounts *accounts.Client
	robots   *robots.Client
}

func (e *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*iamv1alpha1.Token)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotToken)
	}

	if meta.GetExternalName(cr) == "" {
		return managed.ExternalObservation{}, nil
	}
	uid, err := uuid.Parse(meta.GetExternalName(cr))
	if err != nil {
		return managed.ExternalObservation{}, errors.Wrap(err, fmt.Sprintf("failed to parse external name as UUID %s", meta.GetExternalName(cr)))
	}
	resp, err := e.tokens.Get(ctx, uid)
	if err != nil {
		return managed.ExternalObservation{}, errors.Wrap(resource.Ignore(uperrors.IsNotFound, err), "failed to get token")
	}
	cr.Status.SetConditions(v1.Available())
	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: resp.AttributeSet["name"] == cr.Spec.ForProvider.Name,
	}, nil
}

func (e *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*iamv1alpha1.Token)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotToken)
	}
	resp, err := e.tokens.Create(ctx, &tokens.TokenCreateParameters{
		Attributes: tokens.TokenAttributes{
			Name: cr.Spec.ForProvider.Name,
		},
		Relationships: tokens.TokenRelationships{
			Owner: tokens.TokenOwner{
				Data: tokens.TokenOwnerData{
					Type: tokens.TokenOwnerType(cr.Spec.ForProvider.Owner.Type),
					ID:   ptr.Deref(cr.Spec.ForProvider.Owner.ID, ""),
				},
			},
		},
	})
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, "failed to create token")
	}
	meta.SetExternalName(cr, resp.ID.String())

	return managed.ExternalCreation{
		// Optionally return any details that may be required to connect to the
		// external resource. These will be stored as the connection secret.
		ConnectionDetails: managed.ConnectionDetails{
			"token": []byte(fmt.Sprint(resp.DataSet.Meta["jwt"])),
			"id":    []byte(fmt.Sprint(resp.ID.String())),
		},
	}, nil
}

func (e *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*iamv1alpha1.Token)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotToken)
	}

	_, err := e.tokens.Update(ctx, &tokens.TokenUpdateParameters{
		Attributes: tokens.TokenAttributes{
			Name: cr.Spec.ForProvider.Name,
		},
	})
	return managed.ExternalUpdate{}, errors.Wrap(err, "failed to update token")
}

func (e *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*iamv1alpha1.Token)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotToken)
	}
	uid, err := uuid.Parse(meta.GetExternalName(cr))
	if err != nil {
		return managed.ExternalDelete{}, errors.Wrap(err, "cannot parse external name as UUID")
	}
	return managed.ExternalDelete{}, errors.Wrap(resource.Ignore(uperrors.IsNotFound, e.tokens.Delete(ctx, uid)), "failed to delete token")
}
