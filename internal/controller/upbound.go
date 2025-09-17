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

package controller

import (
	"github.com/crossplane/crossplane-runtime/v2/pkg/errors"
	ctrl "sigs.k8s.io/controller-runtime"

	xpv2controller "github.com/crossplane/crossplane-runtime/v2/pkg/controller"

	controllercluster "github.com/upbound/provider-upbound/internal/controller/cluster"
	controller "github.com/upbound/provider-upbound/internal/controller/namespaced"
)

// Setup creates all Upbound controllers with the supplied logger and adds them to
// the supplied manager.
func Setup(mgr ctrl.Manager, o xpv2controller.Options) error {
	if err := controllercluster.Setup(mgr, o); err != nil {
		return errors.Wrap(err, "failed to setup controllers related to cluster-scoped managed resources")
	}

	if err := controller.Setup(mgr, o); err != nil {
		return errors.Wrap(err, "failed to setup controllers related to namespaced managed resources")
	}

	return nil
}
