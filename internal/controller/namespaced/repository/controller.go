/*
Copyright 2025 Upbound Inc.

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

package repository

import (
	"context"

	v1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
	xpcontroller "github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/event"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	"github.com/pkg/errors"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	uperrors "github.com/upbound/up-sdk-go/errors"
	"github.com/upbound/up-sdk-go/service/repositories"

	repov1alpha1 "github.com/upbound/provider-upbound/apis/namespaced/repository/v1alpha1"
	upclient "github.com/upbound/provider-upbound/internal/client"
	"github.com/upbound/provider-upbound/internal/client/repository"
	"github.com/upbound/provider-upbound/internal/controller/namespaced/config"
	"github.com/upbound/provider-upbound/internal/features"
)

const (
	errNotRepository = "managed resource is not a Repository custom resource"
	errNewClient     = "cannot create new Service"
)

// Setup adds a controller that reconciles Repository managed resources.
func Setup(mgr ctrl.Manager, o xpcontroller.Options) error {
	name := managed.ControllerName(repov1alpha1.RepositoryGroupKind)
	initializers := []managed.Initializer{managed.NewNameAsExternalName(mgr.GetClient())}
	reconcilerOpts := []managed.ReconcilerOption{
		managed.WithExternalConnector(&connector{
			kube: mgr.GetClient(),
		}),
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
		resource.ManagedKind(repov1alpha1.RepositoryGroupVersionKind),
		reconcilerOpts...)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o.ForControllerRuntime()).
		WithEventFilter(resource.DesiredStateChanged()).
		For(&repov1alpha1.Repository{}).
		Complete(r)
}

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
	cr, ok := mg.(*repov1alpha1.Repository)
	if !ok {
		return nil, errors.New(errNotRepository)
	}

	cfg, _, err := upclient.NewConfig(ctx, c.kube, config.GetProviderConfigSpecFn(cr))
	if err != nil {
		return nil, errors.Wrap(err, errNewClient)
	}

	return &external{
		repositories: &sdkClient{upstream: repositories.NewClient(cfg)},
	}, nil
}

func (e *external) Disconnect(_ context.Context) error {
	// If there's nothing special to clean up, just return nil.
	return nil
}

// An ExternalClient observes, then either creates, updates, or deletes an
// external resource to ensure it reflects the managed resource's desired state.
type external struct {
	repositories RepoClient
}

func (e *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*repov1alpha1.Repository)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotRepository)
	}

	if meta.GetExternalName(cr) == "" {
		return managed.ExternalObservation{}, nil
	}

	resp, err := e.repositories.Get(ctx, cr.Spec.ForProvider.OrganizationName, meta.GetExternalName(cr))
	if err != nil {
		return managed.ExternalObservation{}, errors.Wrap(resource.Ignore(uperrors.IsNotFound, err), "cannot get repository")
	}

	repoList := repositories.RepositoryListResponse{
		Repositories: []repositories.Repository{resp.Repository},
	}

	cr.Status.SetConditions(v1.Available())
	cr.Status.AtProvider.RepositoryObservation = repository.StatusFromResponse(repoList.Repositories[0])

	publishPolicy := repositories.PublishPolicy("draft")
	if cr.Spec.ForProvider.Publish {
		publishPolicy = "publish"
	}

	resourceUpToDate := cr.Spec.ForProvider.Public == resp.Public || ptr.Deref(resp.Publish, "") == publishPolicy

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: resourceUpToDate,
	}, nil
}

func (e *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*repov1alpha1.Repository)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotRepository)
	}
	visibility := createOrUpdatePublic(cr.Spec.ForProvider.Public)
	publishPolicy := createOrUpdatePublish(cr.Spec.ForProvider.Publish)
	err := e.repositories.CreateOrUpdateWithOptions(ctx, cr.Spec.ForProvider.OrganizationName, cr.Spec.ForProvider.Name, visibility, publishPolicy)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, "cannot create repository")
	}
	meta.SetExternalName(cr, cr.Spec.ForProvider.Name)
	return managed.ExternalCreation{}, nil
}

func (e *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*repov1alpha1.Repository)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotRepository)
	}
	visibility := createOrUpdatePublic(cr.Spec.ForProvider.Public)
	publishPolicy := createOrUpdatePublish(cr.Spec.ForProvider.Publish)

	err := e.repositories.CreateOrUpdateWithOptions(ctx, cr.Spec.ForProvider.OrganizationName, cr.Spec.ForProvider.Name, visibility, publishPolicy)
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, "cannot update repository")
	}

	return managed.ExternalUpdate{}, nil
}

func (e *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*repov1alpha1.Repository)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotRepository)
	}

	return managed.ExternalDelete{}, errors.Wrap(resource.Ignore(uperrors.IsNotFound, e.repositories.Delete(ctx, cr.Spec.ForProvider.OrganizationName, meta.GetExternalName(cr))), "cannot delete repository")
}

func createOrUpdatePublic(isPublic bool) repositories.CreateOrUpdateOption {
	private := repositories.WithPublic()
	if !isPublic {
		private = repositories.WithPrivate()
	}
	return private
}

func createOrUpdatePublish(publish bool) repositories.CreateOrUpdateOption {
	publishPolicy := repositories.WithDraft()
	if publish {
		publishPolicy = repositories.WithPublish()
	}
	return publishPolicy
}
