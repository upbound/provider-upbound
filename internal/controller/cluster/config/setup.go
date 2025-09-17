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

package config

import (
	ctrl "sigs.k8s.io/controller-runtime"

	xpcontroller "github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/event"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/providerconfig"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"

	v1alpha1cluster "github.com/upbound/provider-upbound/apis/cluster/v1alpha1"
)

// SetupGated calls setup when the legacy
// ProviderConfig GVR becomes available in the API.
func SetupGated(mgr ctrl.Manager, o xpcontroller.Options) error {
	o.Gate.Register(func() {
		if err := setup(mgr, o); err != nil {
			panic(err)
		}
	}, v1alpha1cluster.ProviderConfigGroupVersionKind)
	return nil
}

// setup adds a controller that reconciles legacy ProviderConfigs by
// accounting for their current usage.
func setup(mgr ctrl.Manager, o xpcontroller.Options) error {
	name := providerconfig.ControllerName(v1alpha1cluster.ProviderConfigGroupKind)

	of := resource.ProviderConfigKinds{
		Config:    v1alpha1cluster.ProviderConfigGroupVersionKind,
		Usage:     v1alpha1cluster.ProviderConfigUsageGroupVersionKind,
		UsageList: v1alpha1cluster.ProviderConfigUsageListGroupVersionKind,
	}

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o.ForControllerRuntime()).
		For(&v1alpha1cluster.ProviderConfig{}).
		Watches(&v1alpha1cluster.ProviderConfigUsage{}, &resource.EnqueueRequestForProviderConfig{}).
		Complete(providerconfig.NewReconciler(mgr, of,
			providerconfig.WithLogger(o.Logger.WithValues("controller", name)),
			providerconfig.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name)))))
}
