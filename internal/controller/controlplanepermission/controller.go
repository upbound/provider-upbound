/*
Copyright 2022 The Crossplane Authors.

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

package controlplanepermission

import (
	"context"

	v1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/pkg/connection"
	"github.com/crossplane/crossplane-runtime/pkg/controller"
	"github.com/crossplane/crossplane-runtime/pkg/event"
	"github.com/crossplane/crossplane-runtime/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	"github.com/pkg/errors"
	"k8s.io/utils/pointer"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/upbound/provider-upbound/apis/mcp/v1alpha1"
	apisv1alpha1 "github.com/upbound/provider-upbound/apis/v1alpha1"
	upclient "github.com/upbound/provider-upbound/internal/client"
	"github.com/upbound/provider-upbound/internal/client/controlplanepermission"
	"github.com/upbound/provider-upbound/internal/controller/features"
	uperrors "github.com/upbound/up-sdk-go/errors"
)

const (
	errNotControlPlanePermission = "managed resource is not a ControlPlanePermission custom resource"
	errTrackPCUsage              = "cannot track ProviderConfig usage"

	errNewClient = "cannot create new Service"
)

// Setup adds a controller that reconciles ControlPlanePermission managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v1alpha1.ControlPlanePermissionGroupKind)

	cps := []managed.ConnectionPublisher{managed.NewAPISecretPublisher(mgr.GetClient(), mgr.GetScheme())}
	if o.Features.Enabled(features.EnableAlphaExternalSecretStores) {
		cps = append(cps, connection.NewDetailsManager(mgr.GetClient(), apisv1alpha1.StoreConfigGroupVersionKind))
	}

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v1alpha1.ControlPlanePermissionGroupVersionKind),
		managed.WithExternalConnecter(&connector{
			kube:  mgr.GetClient(),
			usage: resource.NewProviderConfigUsageTracker(mgr.GetClient(), &apisv1alpha1.ProviderConfigUsage{}),
		}),
		managed.WithLogger(o.Logger.WithValues("controller", name)),
		managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name))),
		managed.WithInitializers(),
		managed.WithConnectionPublishers(cps...))

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o.ForControllerRuntime()).
		For(&v1alpha1.ControlPlanePermission{}).
		Complete(ratelimiter.NewReconciler(name, r, o.GlobalRateLimiter))
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
	cr, ok := mg.(*v1alpha1.ControlPlanePermission)
	if !ok {
		return nil, errors.New(errNotControlPlanePermission)
	}

	if err := c.usage.Track(ctx, mg); err != nil {
		return nil, errors.Wrap(err, errTrackPCUsage)
	}

	cfg, err := upclient.NewConfig(ctx, c.kube, cr)
	if err != nil {
		return nil, errors.Wrap(err, errNewClient)
	}

	return &external{
		controlPlanePermissions: controlplanepermission.NewClient(cfg),
	}, nil
}

// An ExternalClient observes, then either creates, updates, or deletes an
// external resource to ensure it reflects the managed resource's desired state.
type external struct {
	controlPlanePermissions *controlplanepermission.Client
}

func (c *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v1alpha1.ControlPlanePermission)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotControlPlanePermission)
	}

	// NOTE(muvaf): External name annotation is not used since this is an attribute
	// that binds to resources and the id of both are already under spec.

	params := &controlplanepermission.GetParameters{
		AccountName: cr.Spec.ForProvider.OrganizationName,
		TeamID:      pointer.StringDeref(cr.Spec.ForProvider.TeamID, ""),
	}
	resp, err := c.controlPlanePermissions.Get(ctx, params)
	if err != nil {
		return managed.ExternalObservation{}, errors.Wrapf(resource.Ignore(uperrors.IsNotFound, err), "cannot get control plane permissions of team %s in org %s", params.TeamID, cr.Spec.ForProvider.OrganizationName)
	}
	if resp.Privilege != cr.Spec.ForProvider.Permission {
		return managed.ExternalObservation{
			ResourceExists:   true,
			ResourceUpToDate: false,
		}, nil
	}

	cr.Status.SetConditions(v1.Available())

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: true,
	}, nil
}

func (c *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1alpha1.ControlPlanePermission)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotControlPlanePermission)
	}
	params := &controlplanepermission.ApplyParameters{
		AccountName:      cr.Spec.ForProvider.OrganizationName,
		TeamID:           pointer.StringDeref(cr.Spec.ForProvider.TeamID, ""),
		ControlPlaneName: cr.Spec.ForProvider.ControlPlaneName,
		Permission:       cr.Spec.ForProvider.Permission,
	}
	return managed.ExternalCreation{}, errors.Wrap(c.controlPlanePermissions.Apply(ctx, params), "cannot apply control plane permission")
}

func (c *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1alpha1.ControlPlanePermission)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotControlPlanePermission)
	}
	params := &controlplanepermission.ApplyParameters{
		AccountName:      cr.Spec.ForProvider.OrganizationName,
		TeamID:           pointer.StringDeref(cr.Spec.ForProvider.TeamID, ""),
		ControlPlaneName: cr.Spec.ForProvider.ControlPlaneName,
		Permission:       cr.Spec.ForProvider.Permission,
	}
	return managed.ExternalUpdate{}, errors.Wrap(c.controlPlanePermissions.Apply(ctx, params), "cannot apply control plane permission")
}

func (c *external) Delete(ctx context.Context, mg resource.Managed) error {
	cr, ok := mg.(*v1alpha1.ControlPlanePermission)
	if !ok {
		return errors.New(errNotControlPlanePermission)
	}
	params := &controlplanepermission.DeleteParameters{
		AccountName:      cr.Spec.ForProvider.OrganizationName,
		TeamID:           pointer.StringDeref(cr.Spec.ForProvider.TeamID, ""),
		ControlPlaneName: cr.Spec.ForProvider.ControlPlaneName,
	}
	return errors.Wrap(resource.Ignore(uperrors.IsNotFound, c.controlPlanePermissions.Delete(ctx, params)), "cannot delete control plane permission")
}