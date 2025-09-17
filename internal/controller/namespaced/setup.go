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

package controller

import (
	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/upbound/provider-upbound/internal/controller/namespaced/config"
	"github.com/upbound/provider-upbound/internal/controller/namespaced/repository"
	"github.com/upbound/provider-upbound/internal/controller/namespaced/repositorypermission"
	"github.com/upbound/provider-upbound/internal/controller/namespaced/robot"
	"github.com/upbound/provider-upbound/internal/controller/namespaced/robotteammembership"
	"github.com/upbound/provider-upbound/internal/controller/namespaced/team"
	"github.com/upbound/provider-upbound/internal/controller/namespaced/token"
)

// Setup creates all Upbound controllers related to namespaced MRs
// with the supplied logger and adds them to the supplied manager.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	for _, setup := range []func(ctrl.Manager, controller.Options) error{
		config.SetupClusterScopedGated,
		config.SetupNamespacedGated,
		repository.SetupGated,
		repositorypermission.SetupGated,
		robot.SetupGated,
		robotteammembership.SetupGated,
		team.SetupGated,
		token.SetupGated,
	} {
		if err := setup(mgr, o); err != nil {
			return err
		}
	}
	return nil
}
