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
	"time"

	"github.com/crossplane/crossplane-runtime/v2/pkg/event"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
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

	errGetConnectionSecret         = "cannot get connection secret"
	errConnectionSecretUnavailable = "token exists upstream but connection secret is missing or empty; the token value cannot be recovered via the API - delete the upstream token to allow automatic recreation, or set spec.connectionSecretRecovery: Recreate"
	errDeleteToken                 = "cannot delete the upstream token to recreate it"
	errRecreateNotAllowed          = "connection secret is missing or empty, but spec.connectionSecretRecovery: Recreate needs management policies that allow Create and Delete"
	errRecreatedRecently           = "connection secret is missing or empty, but the token was created less than 10 minutes ago; it is recreated at most once per 10 minutes"
	msgFmtRecreatingToken          = "connection secret is missing or empty: deleted upstream token %s so that a new token is created and published"

	reasonRecreatingToken event.Reason = "RecreatingToken"

	// recreateCooldown is how long after creating a token the provider won't
	// recreate it, so that a new token whose secret can't be published or
	// is lost again right away isn't recreated on every reconcile.
	recreateCooldown = 10 * time.Minute
)

// A connector is expected to produce an ExternalClient when its Connect method
// is called.
type connector struct {
	kube                      client.Client
	apiReader                 client.Reader
	recorder                  event.Recorder
	managementPoliciesEnabled bool
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
		kube:                      c.kube,
		apiReader:                 c.apiReader,
		recorder:                  c.recorder,
		managementPoliciesEnabled: c.managementPoliciesEnabled,
		tokens:                    tokens.NewClient(cfg),
		accounts:                  accounts.NewClient(cfg),
		robots:                    robots.NewClient(cfg),
	}, nil
}

func (e *external) Disconnect(_ context.Context) error {
	// If there's nothing special to clean up, just return nil.
	return nil
}

// An ExternalClient observes, then either creates, updates, or deletes an
// external resource to ensure it reflects the managed resource's desired state.
type external struct {
	kube client.Client
	// apiReader bypasses the cache to confirm a missing connection secret.
	apiReader client.Reader
	recorder  event.Recorder
	// managementPoliciesEnabled is true if the managed reconciler honours
	// spec.managementPolicies.
	managementPoliciesEnabled bool
	tokens                    *tokens.Client
	accounts                  *accounts.Client
	robots                    *robots.Client
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
	cr.Status.SetConditions(xpv2.Available())

	// The token JWT is returned by the API only on creation (tokens.Create), not
	// on subsequent reads (tokens.Get). If the MR was adopted without its
	// connection secret — e.g., restored from a backup that excluded secrets, or
	// re-applied from Git — the token value is unrecoverable. Erroring here
	// prevents the reconciler from calling PublishConnection with an empty map,
	// which would otherwise write a DATA=0 secret that silently breaks consumers.
	if ref := cr.GetWriteConnectionSecretToReference(); ref != nil && ref.Name != "" && !meta.WasDeleted(cr) {
		key := types.NamespacedName{Namespace: cr.GetNamespace(), Name: ref.Name}
		s := &corev1.Secret{}
		err := e.kube.Get(ctx, key, s)
		if kerrors.IsNotFound(err) || (err == nil && len(s.Data["token"]) == 0) {
			if cr.Spec.ConnectionSecretRecovery != iamv1alpha1.ConnectionSecretRecoveryRecreate {
				return managed.ExternalObservation{}, errors.New(errConnectionSecretUnavailable)
			}
			// A deleted token is reported as gone, so that the managed
			// reconciler creates a new one and publishes its connection secret.
			deleted, rerr := e.recreate(ctx, cr, uid, key)
			if rerr != nil || deleted {
				return managed.ExternalObservation{}, rerr
			}
		} else if err != nil {
			return managed.ExternalObservation{}, errors.Wrap(err, errGetConnectionSecret)
		}
	}

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: resp.AttributeSet["name"] == cr.Spec.ForProvider.Name,
	}, nil
}

// recreate deletes the upstream token whose connection secret is missing or
// empty and returns true, once an uncached read confirms the miss and the
// policies allow it. It returns false and no error if the secret is present.
func (e *external) recreate(ctx context.Context, cr *iamv1alpha1.Token, uid uuid.UUID, key types.NamespacedName) (bool, error) {
	// Never revoke a working token because of a lagging cache.
	s := &corev1.Secret{}
	if err := e.apiReader.Get(ctx, key, s); resource.IgnoreNotFound(err) != nil {
		return false, errors.Wrap(err, errGetConnectionSecret)
	}
	if len(s.Data["token"]) > 0 {
		return false, nil
	}
	p := managed.NewManagementPoliciesResolver(e.managementPoliciesEnabled, cr.GetManagementPolicies())
	if !p.ShouldCreate() || !p.ShouldDelete() {
		return false, errors.New(errRecreateNotAllowed)
	}
	if meta.ExternalCreateSucceededDuring(cr, recreateCooldown) {
		return false, errors.New(errRecreatedRecently)
	}
	if err := resource.Ignore(uperrors.IsNotFound, e.tokens.Delete(ctx, uid)); err != nil {
		return false, errors.Wrap(err, errDeleteToken)
	}
	e.recorder.Event(cr, event.Warning(reasonRecreatingToken, errors.Errorf(msgFmtRecreatingToken, uid)))
	return true, nil
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
