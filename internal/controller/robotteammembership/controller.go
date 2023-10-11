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

	v1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/pkg/controller"
	"github.com/crossplane/crossplane-runtime/pkg/event"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	"github.com/pkg/errors"
	"k8s.io/utils/pointer"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	uperrors "github.com/upbound/up-sdk-go/errors"

	"github.com/upbound/provider-upbound/apis/iam/v1alpha1"
	apisv1alpha1 "github.com/upbound/provider-upbound/apis/v1alpha1"
	upclient "github.com/upbound/provider-upbound/internal/client"
	"github.com/upbound/provider-upbound/internal/client/robotteammembership"
	"github.com/upbound/provider-upbound/internal/features"
)

const (
	errNotRobotTeamMembership = "managed resource is not a RobotTeamMembership custom resource"
	errTrackPCUsage           = "cannot track ProviderConfig usage"

	errNewClient = "cannot create new Service"
)

// Setup adds a controller that reconciles RobotTeamMembership managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v1alpha1.RobotTeamMembershipKindAPIVersion)
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
		resource.ManagedKind(v1alpha1.RobotTeamMembershipGroupVersionKind),
		reconcilerOpts...)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o.ForControllerRuntime()).
		WithEventFilter(resource.DesiredStateChanged()).
		For(&v1alpha1.RobotTeamMembership{}).
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
	cr, ok := mg.(*v1alpha1.RobotTeamMembership)
	if !ok {
		return nil, errors.New(errNotRobotTeamMembership)
	}

	if err := c.usage.Track(ctx, mg); err != nil {
		return nil, errors.Wrap(err, errTrackPCUsage)
	}

	cfg, _, err := upclient.NewConfig(ctx, c.kube, cr)
	if err != nil {
		return nil, errors.Wrap(err, errNewClient)
	}

	return &external{
		robotTeamMemberships: robotteammembership.NewClient(cfg),
	}, nil
}

// An ExternalClient observes, then either creates, updates, or deletes an
// external resource to ensure it reflects the managed resource's desired state.
type external struct {
	robotTeamMemberships *robotteammembership.Client
}

func (c *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v1alpha1.RobotTeamMembership)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotRobotTeamMembership)
	}

	// NOTE(muvaf): External name annotation is not used since this is an attribute
	// that binds to resources and the id of both are already under spec.

	tid := pointer.StringDeref(cr.Spec.ForProvider.TeamID, "")
	rid := pointer.StringDeref(cr.Spec.ForProvider.RobotID, "")
	if err := c.robotTeamMemberships.Get(ctx, rid, tid); err != nil {
		return managed.ExternalObservation{}, errors.Wrap(resource.Ignore(uperrors.IsNotFound, err), "cannot get robot team ids")
	}
	cr.Status.SetConditions(v1.Available())

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: true,
	}, nil
}

func (c *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1alpha1.RobotTeamMembership)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotRobotTeamMembership)
	}
	tid := pointer.StringDeref(cr.Spec.ForProvider.TeamID, "")
	rid := pointer.StringDeref(cr.Spec.ForProvider.RobotID, "")
	if err := c.robotTeamMemberships.Create(ctx, rid, &robotteammembership.ResourceIdentifier{
		ID:   tid,
		Type: robotteammembership.RobotMembershipTypeTeam,
	}); err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, "cannot create team membership for the robot")
	}
	return managed.ExternalCreation{}, nil
}

func (c *external) Update(_ context.Context, _ resource.Managed) (managed.ExternalUpdate, error) {
	// There is no Update endpoints in robotTeamMemberships API.
	return managed.ExternalUpdate{}, nil
}

func (c *external) Delete(ctx context.Context, mg resource.Managed) error {
	cr, ok := mg.(*v1alpha1.RobotTeamMembership)
	if !ok {
		return errors.New(errNotRobotTeamMembership)
	}
	err := c.robotTeamMemberships.Delete(ctx, pointer.StringDeref(cr.Spec.ForProvider.RobotID, ""), &robotteammembership.DeleteParameters{
		ID:   pointer.StringDeref(cr.Spec.ForProvider.TeamID, ""),
		Type: robotteammembership.RobotMembershipTypeTeam,
	})
	return errors.Wrap(resource.Ignore(uperrors.IsNotFound, err), "cannot delete robot team membership")
}
