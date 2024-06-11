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

package controlplane

import (
	"context"

	v1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/pkg/controller"
	"github.com/crossplane/crossplane-runtime/pkg/event"
	"github.com/crossplane/crossplane-runtime/pkg/meta"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	cmp "github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	uperrors "github.com/upbound/up-sdk-go/errors"
	"github.com/upbound/up-sdk-go/service/configurations"
	"github.com/upbound/up-sdk-go/service/controlplanes"
	"k8s.io/utils/pointer"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/upbound/provider-upbound/apis/mcp/v1alpha1"
	apisv1alpha1 "github.com/upbound/provider-upbound/apis/v1alpha1"
	upclient "github.com/upbound/provider-upbound/internal/client"
	"github.com/upbound/provider-upbound/internal/client/controlplane"
	"github.com/upbound/provider-upbound/internal/features"
)

const (
	errNotControlPlane = "managed resource is not a ControlPlane custom resource"
	errTrackPCUsage    = "cannot track ProviderConfig usage"

	errNewClient = "cannot create new Service"
)

// Setup adds a controller that reconciles Robot managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v1alpha1.ControlPlaneGroupKind)
	initializers := []managed.Initializer{managed.NewNameAsExternalName(mgr.GetClient())}
	cps := []managed.ConnectionPublisher{managed.NewAPISecretPublisher(mgr.GetClient(), mgr.GetScheme())}
	reconcilerOpts := []managed.ReconcilerOption{
		managed.WithExternalConnecter(&connector{
			kube:  mgr.GetClient(),
			usage: resource.NewProviderConfigUsageTracker(mgr.GetClient(), &apisv1alpha1.ProviderConfigUsage{}),
		}),
		managed.WithConnectionPublishers(cps...),
		managed.WithPollInterval(o.PollInterval),
		managed.WithReferenceResolver(managed.NewAPISimpleReferenceResolver(mgr.GetClient())),
		managed.WithInitializers(initializers...),
		managed.WithLogger(o.Logger.WithValues("controller", name)),
		managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name))),
	}

	if o.Features.Enabled(features.EnableAlphaManagementPolicies) {
		reconcilerOpts = append(reconcilerOpts, managed.WithManagementPolicies())
	}

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v1alpha1.ControlPlaneGroupVersionKind),
		reconcilerOpts...)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o.ForControllerRuntime()).
		WithEventFilter(resource.DesiredStateChanged()).
		For(&v1alpha1.ControlPlane{}).
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
	cr, ok := mg.(*v1alpha1.ControlPlane)
	if !ok {
		return nil, errors.New(errNotControlPlane)
	}

	if err := c.usage.Track(ctx, mg); err != nil {
		return nil, errors.Wrap(err, errTrackPCUsage)
	}

	cfg, _, err := upclient.NewConfig(ctx, c.kube, cr)
	if err != nil {
		return nil, errors.Wrap(err, errNewClient)
	}

	return &external{
		controlPlane:      controlplanes.NewClient(cfg),
		configurations:    configurations.NewClient(cfg),
		localControlPlane: controlplane.NewClient(cfg),
		kube:              c.kube,
	}, nil
}

// An ExternalClient observes, then either creates, updates, or deletes an
// external resource to ensure it reflects the managed resource's desired state.
type external struct {
	controlPlane      *controlplanes.Client
	configurations    *configurations.Client
	localControlPlane *controlplane.Client
	kube              client.Client
}

func (c *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) { //nolint:gocyclo
	cr, ok := mg.(*v1alpha1.ControlPlane)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotControlPlane)
	}

	if meta.GetExternalName(cr) == "" {
		return managed.ExternalObservation{}, nil
	}

	resp, err := c.controlPlane.Get(ctx, cr.Spec.ForProvider.OrganizationName, meta.GetExternalName(cr))
	if err != nil {
		return managed.ExternalObservation{}, errors.Wrap(resource.Ignore(uperrors.IsNotFound, err), "cannot get control plane")
	}

	o, err := c.configurations.Get(ctx, cr.Spec.ForProvider.OrganizationName, pointer.StringDeref(cr.Spec.ForProvider.Configuration, ""))
	if err != nil {
		return managed.ExternalObservation{}, errors.Wrap(resource.Ignore(uperrors.IsNotFound, err), "cannot get configuration")
	}

	cr.Status.AtProvider = controlplane.StatusFromResponse(resp, o.LatestVersion)
	up2date := pointer.StringDeref(cr.Status.AtProvider.ControlPlane.Configuration.LatestAvailableVersion, "") == pointer.StringDeref(cr.Spec.ForProvider.Version, "")
	cr.Status.AtProvider.ControlPlane.VersionUpToDate = &up2date

	// lateInit
	current := cr.Spec.ForProvider.DeepCopy()
	if resp.ControlPlane.Configuration.CurrentVersion != nil && cr.Spec.ForProvider.Version == nil {
		cr.Spec.ForProvider.Version = resp.ControlPlane.Configuration.CurrentVersion
	}

	// autoUpdate true
	if cr.Spec.ForProvider.AutoUpdate != nil && *cr.Spec.ForProvider.AutoUpdate && cr.Status.AtProvider.ControlPlane.Configuration.Status == v1alpha1.ConfigurationReady {
		version := pointer.StringDeref(cr.Status.AtProvider.ControlPlane.Configuration.LatestAvailableVersion, "")
		currentVersion := pointer.StringDeref(resp.ControlPlane.Configuration.CurrentVersion, "")

		if controlplane.CompareVersions(version, currentVersion) == 1 {
			// set new version
			cr.Spec.ForProvider.Version = &version
			return managed.ExternalObservation{
				ResourceExists:          true,
				ResourceUpToDate:        false,
				ResourceLateInitialized: !cmp.Equal(current, &cr.Spec.ForProvider),
			}, nil
		}
	}

	// version update
	if cr.Spec.ForProvider.Version != nil && cr.Status.AtProvider.ControlPlane.Configuration.Status == v1alpha1.ConfigurationReady {
		version := pointer.StringDeref(cr.Spec.ForProvider.Version, "")
		currentVersion := pointer.StringDeref(resp.ControlPlane.Configuration.CurrentVersion, "")

		if controlplane.CompareVersions(version, currentVersion) != 0 {
			return managed.ExternalObservation{
				ResourceExists:   true,
				ResourceUpToDate: false,
			}, nil
		}
	}

	cr.Status.SetConditions(v1.Available())
	return managed.ExternalObservation{
		ResourceExists:          true,
		ResourceUpToDate:        true,
		ResourceLateInitialized: !cmp.Equal(current, &cr.Spec.ForProvider),
	}, nil
}

func (c *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1alpha1.ControlPlane)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotControlPlane)
	}
	configuration := pointer.StringDeref(cr.Spec.ForProvider.Configuration, "")
	var configurationUuid uuid.UUID

	o, err := c.configurations.Get(ctx, cr.Spec.ForProvider.OrganizationName, configuration)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, "cannot get configuration")
	}
	configurationUuid = o.ID

	resp, err := c.controlPlane.Create(ctx, cr.Spec.ForProvider.OrganizationName, &controlplanes.ControlPlaneCreateParameters{
		Name:            meta.GetExternalName(cr),
		Description:     pointer.StringDeref(cr.Spec.ForProvider.Description, ""),
		ConfigurationID: configurationUuid,
	})
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, "cannot create controlPlane")
	}
	meta.SetExternalName(cr, resp.ControlPlane.Name)
	return managed.ExternalCreation{}, nil
}

func (c *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1alpha1.ControlPlane)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotControlPlane)
	}
	params := &controlplane.ApplyParameters{
		Name:           meta.GetExternalName(cr),
		Organization:   cr.Spec.ForProvider.OrganizationName,
		DesiredVersion: pointer.StringDeref(cr.Spec.ForProvider.Version, ""),
	}
	return managed.ExternalUpdate{}, errors.Wrap(c.localControlPlane.Apply(ctx, params), "cannot apply control plane update")
}

func (c *external) Delete(ctx context.Context, mg resource.Managed) error {
	cr, ok := mg.(*v1alpha1.ControlPlane)
	if !ok {
		return errors.New(errNotControlPlane)
	}

	if cr.Status.AtProvider.Status == v1alpha1.StatusDeleting {
		return nil
	}

	return errors.Wrap(resource.Ignore(uperrors.IsNotFound, c.controlPlane.Delete(ctx, cr.Spec.ForProvider.OrganizationName, meta.GetExternalName(cr))), "cannot delete controlPlane")
}
