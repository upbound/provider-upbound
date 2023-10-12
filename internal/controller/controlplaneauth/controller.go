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

package controlplaneauth

import (
	"context"
	"fmt"

	v1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/pkg/controller"
	"github.com/crossplane/crossplane-runtime/pkg/event"
	"github.com/crossplane/crossplane-runtime/pkg/meta"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	uperrors "github.com/upbound/up-sdk-go/errors"
	"github.com/upbound/up-sdk-go/service/accounts"
	"github.com/upbound/up-sdk-go/service/tokens"

	upclient "github.com/upbound/provider-upbound/internal/client"
	"github.com/upbound/provider-upbound/internal/client/controlplaneauth"

	"github.com/upbound/provider-upbound/apis/mcp/v1alpha1"
	apisv1alpha1 "github.com/upbound/provider-upbound/apis/v1alpha1"
	"github.com/upbound/provider-upbound/internal/features"
)

const (
	errNotControlPlaneAuth = "managed resource is not a ControlPlaneAuth custom resource"
	errTrackPCUsage        = "cannot track ProviderConfig usage"

	errNewClient = "cannot create new Service"
)

// Setup adds a controller that reconciles ControlPlaneAuth managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v1alpha1.ControlPlaneAuthGroupKind)
	cps := []managed.ConnectionPublisher{managed.NewAPISecretPublisher(mgr.GetClient(), mgr.GetScheme())}
	reconcilerOpts := []managed.ReconcilerOption{
		managed.WithExternalConnecter(&connector{
			kube:  mgr.GetClient(),
			usage: resource.NewProviderConfigUsageTracker(mgr.GetClient(), &apisv1alpha1.ProviderConfigUsage{}),
		}),
		managed.WithConnectionPublishers(cps...),
		managed.WithPollInterval(o.PollInterval),
		managed.WithReferenceResolver(managed.NewAPISimpleReferenceResolver(mgr.GetClient())),
		managed.WithInitializers(),
		managed.WithLogger(o.Logger.WithValues("controller", name)),
		managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name))),
	}

	if o.Features.Enabled(features.EnableAlphaManagementPolicies) {
		reconcilerOpts = append(reconcilerOpts, managed.WithManagementPolicies())
	}

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v1alpha1.ControlPlaneAuthGroupVersionKind),
		reconcilerOpts...)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o.ForControllerRuntime()).
		WithEventFilter(resource.DesiredStateChanged()).
		For(&v1alpha1.ControlPlaneAuth{}).
		Complete(r)
}

// A connector is expected to produce an ExternalClient when its Connect method
// is called.
type connector struct {
	kube  client.Client
	usage resource.Tracker
}

// Connect typically produces an ExternalClient by:
// 1. Tracking that the managed resource is using a ProviderConfig.
// 2. Getting the managed resource's ProviderConfig.
// 3. Getting the credentials specified by the ProviderConfig.
// 4. Using the credentials to form a client.
func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	cr, ok := mg.(*v1alpha1.ControlPlaneAuth)
	if !ok {
		return nil, errors.New(errNotControlPlaneAuth)
	}

	if err := c.usage.Track(ctx, mg); err != nil {
		return nil, errors.Wrap(err, errTrackPCUsage)
	}

	cfg, profile, err := upclient.NewConfig(ctx, c.kube, cr)
	if err != nil {
		return nil, errors.Wrap(err, errNewClient)
	}

	return &external{
		tokens:   tokens.NewClient(cfg),
		accounts: accounts.NewClient(cfg),
		profile:  profile,
		kube:     c.kube,
	}, nil
}

// An ExternalClient observes, then either creates, updates, or deletes an
// external resource to ensure it reflects the managed resource's desired state.
type external struct {
	kube     client.Client
	accounts *accounts.Client
	profile  upclient.Profile
	tokens   *tokens.Client
}

func (c *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v1alpha1.ControlPlaneAuth)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotControlPlaneAuth)
	}

	if meta.GetExternalName(cr) == "" {
		return managed.ExternalObservation{}, nil
	}
	uid, err := uuid.Parse(meta.GetExternalName(cr))
	if err != nil {
		return managed.ExternalObservation{}, errors.Wrap(err, fmt.Sprintf("failed to parse external name as UUID %s", meta.GetExternalName(cr)))
	}
	_, err = c.tokens.Get(ctx, uid)
	if err != nil {
		return managed.ExternalObservation{}, errors.Wrap(resource.Ignore(uperrors.IsNotFound, err), "failed to get token")
	}

	cr.Status.SetConditions(v1.Available())

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: true,
	}, nil
}

func (c *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1alpha1.ControlPlaneAuth)
	var kubeToken string
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotControlPlaneAuth)
	}

	userID, err := upclient.ExtractUserIDFromToken(c.profile.Session)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, "unable to retrieve userId from token")
	}

	t, err := c.tokens.Create(ctx, &tokens.TokenCreateParameters{
		Attributes: tokens.TokenAttributes{
			Name: fmt.Sprintf("%.64s", cr.Spec.ForProvider.ControlPlaneName+"-"+string(cr.UID)),
		},
		Relationships: tokens.TokenRelationships{
			Owner: tokens.TokenOwner{
				Data: tokens.TokenOwnerData{
					Type: tokens.TokenOwnerUser,
					ID:   userID,
				},
			},
		},
	})
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, "cannot create token")
	}
	kubeToken = t.DataSet.Meta["jwt"].(string)
	meta.SetExternalName(cr, t.DataSet.ID.String())

	config, err := controlplaneauth.BuildControlPlaneKubeconfig(
		cr.Spec.ForProvider.OrganizationName,
		cr.Spec.ForProvider.ControlPlaneName,
		kubeToken,
	)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, "cannot create kubeconfig")
	}

	return managed.ExternalCreation{
		ConnectionDetails: managed.ConnectionDetails{
			"kubeconfig": []byte(config),
		},
	}, nil
}

func (c *external) Update(_ context.Context, _ resource.Managed) (managed.ExternalUpdate, error) {
	return managed.ExternalUpdate{}, nil
}

func (c *external) Delete(ctx context.Context, mg resource.Managed) error {
	cr, ok := mg.(*v1alpha1.ControlPlaneAuth)
	if !ok {
		return errors.New(errNotControlPlaneAuth)
	}
	uid, err := uuid.Parse(meta.GetExternalName(cr))
	if err != nil {
		return errors.Wrap(err, "cannot parse external name as UUID")
	}
	return errors.Wrap(resource.Ignore(uperrors.IsNotFound, c.tokens.Delete(ctx, uid)), "failed to delete token")
}
