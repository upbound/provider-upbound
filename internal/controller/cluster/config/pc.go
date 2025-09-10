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

package config

import (
	"context"

	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/types"
	k8scli "sigs.k8s.io/controller-runtime/pkg/client"

	apisv1alpha1cluster "github.com/upbound/provider-upbound/apis/cluster/v1alpha1"
	pcv1alpha1common "github.com/upbound/provider-upbound/apis/common/providerconfig/v1alpha1"
	"github.com/upbound/provider-upbound/internal/client"
)

// GetProviderConfigSpecFn returns a function that returns the spec of
// the referenced ProviderConfig by a legacy cluster-scoped MR.
func GetProviderConfigSpecFn(mg resource.LegacyManaged) client.GetProviderConfigSpecFn {
	return func(ctx context.Context, kube k8scli.Client) (*pcv1alpha1common.ProviderConfigSpec, error) {
		pc := &apisv1alpha1cluster.ProviderConfig{}
		if err := kube.Get(ctx, types.NamespacedName{Name: mg.GetProviderConfigReference().Name}, pc); err != nil {
			return nil, errors.Wrap(err, "failed to get the referenced ProviderConfig by a legacy managed resource")
		}
		return &pc.Spec.ProviderConfigSpec, nil
	}
}
