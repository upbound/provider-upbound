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

	pcv1alpha1common "github.com/upbound/provider-upbound/apis/common/providerconfig/v1alpha1"
	"github.com/upbound/provider-upbound/apis/namespaced/v1alpha1"
	"github.com/upbound/provider-upbound/internal/client"
)

const (
	errTrackPCUsage = "cannot track provider config usage"
)

// GetProviderConfigSpecFn returns a function that returns the spec of
// the referenced ProviderConfig by a namespaced modern MR.
func GetProviderConfigSpecFn(mg resource.ModernManaged) client.GetProviderConfigSpecFn {
	return func(ctx context.Context, kube k8scli.Client) (*pcv1alpha1common.ProviderConfigSpec, error) {
		if err := resource.NewProviderConfigUsageTracker(kube, &v1alpha1.ProviderConfigUsage{}).Track(ctx, mg); err != nil {
			return nil, errors.Wrap(err, errTrackPCUsage)
		}

		ref := mg.GetProviderConfigReference()
		if ref == nil {
			return nil, errors.New("empty provider config reference")
		}

		obj, err := kube.Scheme().New(v1alpha1.SchemeGroupVersion.WithKind(ref.Kind))
		if err != nil {
			return nil, errors.Wrapf(err, "failed to instantiate provider config of kind %q referenced by managed resource %s/%s", ref.Kind, mg.GetNamespace(), mg.GetName())
		}

		pcObj, ok := obj.(resource.ProviderConfig)
		if !ok {
			return nil, errors.Errorf("referenced kind %q by managed resource %s/%s from spec.providerConfigRef is not a valid provider config type of the provider", ref.Kind, mg.GetNamespace(), mg.GetName())
		}

		if err := kube.Get(ctx, types.NamespacedName{Name: ref.Name, Namespace: mg.GetNamespace()}, pcObj); err != nil {
			return nil, errors.Wrapf(err, "failed to get referenced provider config by managed resource %s/%s", mg.GetNamespace(), mg.GetName())
		}

		switch pc := obj.(type) {
		case *v1alpha1.ProviderConfig:
			return &pc.Spec.ProviderConfigSpec, nil

		case *v1alpha1.ClusterProviderConfig:
			return &pc.Spec.ProviderConfigSpec, nil

		default:
			return nil, errors.Errorf("failed to handle the referenced provider config kind %q by managed resource %s/%s", ref.Kind, mg.GetNamespace(), mg.GetName())
		}
	}
}
