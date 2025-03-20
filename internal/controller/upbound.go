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
	"github.com/crossplane/crossplane-runtime/pkg/controller"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/upbound/provider-upbound/internal/controller/config"
	"github.com/upbound/provider-upbound/internal/controller/repository"
	"github.com/upbound/provider-upbound/internal/controller/repositorypermission"
	"github.com/upbound/provider-upbound/internal/controller/robot"
	"github.com/upbound/provider-upbound/internal/controller/robotteammembership"
	"github.com/upbound/provider-upbound/internal/controller/team"
	"github.com/upbound/provider-upbound/internal/controller/token"
)

// Setup creates all Upbound controllers with the supplied logger and adds them to
// the supplied manager.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	for _, setup := range []func(ctrl.Manager, controller.Options) error{
		config.Setup,
		repository.Setup,
		repositorypermission.Setup,
		robot.Setup,
		robotteammembership.Setup,
		team.Setup,
		token.Setup,
	} {
		if err := setup(mgr, o); err != nil {
			return err
		}
	}
	return nil
}
