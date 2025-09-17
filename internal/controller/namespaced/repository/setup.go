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
	ctrl "sigs.k8s.io/controller-runtime"

	xpcontroller "github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/event"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"

	repov1alpha1 "github.com/upbound/provider-upbound/apis/namespaced/repository/v1alpha1"
	"github.com/upbound/provider-upbound/internal/features"
)

// SetupGated calls setup when the namespaced
// Repository GVR becomes available in the API.
func SetupGated(mgr ctrl.Manager, o xpcontroller.Options) error {
	o.Gate.Register(func() {
		if err := setup(mgr, o); err != nil {
			panic(err)
		}
	}, repov1alpha1.RepositoryGroupVersionKind)
	return nil
}

// setup adds a controller that reconciles Repository managed resources.
func setup(mgr ctrl.Manager, o xpcontroller.Options) error {
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
