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

	v1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/event"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	"github.com/pkg/errors"
	uperrors "github.com/upbound/up-sdk-go/errors"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	repov1alpha1cluster "github.com/upbound/provider-upbound/apis/cluster/repository/v1alpha1"
	apisv1alpha1cluster "github.com/upbound/provider-upbound/apis/cluster/v1alpha1"
	upclient "github.com/upbound/provider-upbound/internal/client"
	"github.com/upbound/provider-upbound/internal/client/repositorypermission"
	"github.com/upbound/provider-upbound/internal/features"
)

const (
	errNotPermission = "managed resource is not a RepositoryPermission custom resource"
	errTrackPCUsage  = "cannot track ProviderConfig usage"

	errNewClient = "cannot create new client"
)

// Setup adds a controller that reconciles Permission managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(repov1alpha1cluster.PermissionGroupKind)
	reconcilerOpts := []managed.ReconcilerOption{
		managed.WithExternalConnecter(&connector{
			kube:  mgr.GetClient(),
			usage: resource.NewLegacyProviderConfigUsageTracker(mgr.GetClient(), &apisv1alpha1cluster.ProviderConfigUsage{}),
		}),
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
		resource.ManagedKind(repov1alpha1cluster.PermissionGroupVersionKind),
		reconcilerOpts...)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o.ForControllerRuntime()).
		WithEventFilter(resource.DesiredStateChanged()).
		For(&repov1alpha1cluster.Permission{}).
		Complete(r)
}

// A connector is expected to produce an ExternalClient when its Connect method
// is called.
type connector struct {
	kube  client.Client
	usage *resource.LegacyProviderConfigUsageTracker
}

// Connect typically produces an ExternalClient by:
// 1. Tracking that the managed resource is using a ProviderConfig.
// 2. Getting the managed resource's ProviderConfig.
// 3. Getting the credentials specified by the ProviderConfig.
// 4. Using the credentials to form a client.
func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	cr, ok := mg.(*repov1alpha1cluster.Permission)
	if !ok {
		return nil, errors.New(errNotPermission)
	}

	if err := c.usage.Track(ctx, mg.(resource.LegacyManaged)); err != nil {
		return nil, errors.Wrap(err, errTrackPCUsage)
	}

	cfg, _, err := upclient.NewConfig(ctx, c.kube, cr)
	if err != nil {
		return nil, errors.Wrap(err, errNewClient)
	}

	return &external{
		repositorypermission: repositorypermission.NewClient(cfg),
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
	// would be something like an Upbound SDK client.
	repositorypermission *repositorypermission.Client
}

func (c *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*repov1alpha1cluster.Permission)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotPermission)
	}

	err := c.repositorypermission.Get(ctx, &repositorypermission.GetParameters{
		Repository:   ptr.Deref(cr.Spec.ForProvider.Repository, ""),
		Organization: cr.Spec.ForProvider.OrganizationName,
		TeamID:       ptr.Deref(cr.Spec.ForProvider.TeamID, ""),
	})

	if err != nil {
		return managed.ExternalObservation{}, errors.Wrap(resource.Ignore(uperrors.IsNotFound, err), "failed to get team")
	}
	cr.Status.SetConditions(v1.Available())
	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: true,
	}, nil
}

func (c *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*repov1alpha1cluster.Permission)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotPermission)
	}

	err := c.repositorypermission.Create(ctx, &repositorypermission.CreateParameters{
		Repository:   ptr.Deref(cr.Spec.ForProvider.Repository, ""),
		Organization: cr.Spec.ForProvider.OrganizationName,
		TeamID:       ptr.Deref(cr.Spec.ForProvider.TeamID, ""),
		Permission:   cr.Spec.ForProvider.Permission,
	})
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, "failed to create repository permission")
	}

	meta.SetExternalName(cr, ptr.Deref(cr.Spec.ForProvider.Repository, ""))

	return managed.ExternalCreation{}, nil
}

func (c *external) Update(_ context.Context, _ resource.Managed) (managed.ExternalUpdate, error) {
	return managed.ExternalUpdate{}, nil
}

func (c *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*repov1alpha1cluster.Permission)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotPermission)
	}

	err := c.repositorypermission.Delete(ctx, &repositorypermission.GetParameters{
		Repository:   ptr.Deref(cr.Spec.ForProvider.Repository, ""),
		Organization: cr.Spec.ForProvider.OrganizationName,
		TeamID:       ptr.Deref(cr.Spec.ForProvider.TeamID, ""),
	})
	return managed.ExternalDelete{}, errors.Wrap(resource.Ignore(uperrors.IsNotFound, err), "cannot delete repositroy permission")

}
