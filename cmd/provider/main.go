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

package main

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/alecthomas/kingpin.v2"
	authv1 "k8s.io/api/authorization/v1"
	corev1 "k8s.io/api/core/v1"
	extv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/errors"
	"github.com/crossplane/crossplane-runtime/v2/pkg/feature"
	"github.com/crossplane/crossplane-runtime/v2/pkg/gate"
	"github.com/crossplane/crossplane-runtime/v2/pkg/logging"
	"github.com/crossplane/crossplane-runtime/v2/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/customresourcesgate"

	apiscluster "github.com/upbound/provider-upbound/apis/cluster"
	apis "github.com/upbound/provider-upbound/apis/namespaced"
	"github.com/upbound/provider-upbound/internal/bootcheck"
	upbound "github.com/upbound/provider-upbound/internal/controller"
	"github.com/upbound/provider-upbound/internal/features"
)

func init() {
	err := bootcheck.CheckEnv()
	if err != nil {
		log.Fatalf("bootcheck failed. provider will not be started: %v", err)
	}
}

func main() {
	var (
		app            = kingpin.New(filepath.Base(os.Args[0]), "Upbound support for Crossplane.").DefaultEnvars()
		debug          = app.Flag("debug", "Run with debug logging.").Short('d').Bool()
		leaderElection = app.Flag("leader-election", "Use leader election for the controller manager.").Short('l').Default("false").OverrideDefaultFromEnvar("LEADER_ELECTION").Bool()

		syncInterval     = app.Flag("sync", "How often all resources will be double-checked for drift from the desired state.").Short('s').Default("1h").Duration()
		pollInterval     = app.Flag("poll", "How often individual resources will be checked for drift from the desired state").Default("1m").Duration()
		maxReconcileRate = app.Flag("max-reconcile-rate", "The global maximum rate per second at which resources may checked for drift from the desired state.").Default("10").Int()

		enableManagementPolicies = app.Flag("enable-management-policies", "Enable support for Management Policies. Enabled by default; set to false to opt out.").Default("true").Envar("ENABLE_MANAGEMENT_POLICIES").Bool()
		enableSecretCache        = app.Flag("enable-secret-cache", "Enable caching of Secret objects. When true, Secrets are served from the informer cache instead of direct API calls. This reduces API server load but increases memory usage.").Default("true").Envar("ENABLE_SECRET_CACHE").Bool()
	)
	kingpin.MustParse(app.Parse(os.Args[1:]))

	zl := zap.New(zap.UseDevMode(*debug))
	logger := logging.NewLogrLogger(zl.WithName("provider-upbound"))
	ctrl.SetLogger(zl)

	cfg, err := ctrl.GetConfig()
	kingpin.FatalIfError(err, "Cannot get API server rest config")

	var clientOpts client.Options
	if !*enableSecretCache {
		clientOpts = client.Options{
			Cache: &client.CacheOptions{
				DisableFor: []client.Object{&corev1.Secret{}},
			},
		}
	}

	scheme := runtime.NewScheme()
	kingpin.FatalIfError(clientgoscheme.AddToScheme(scheme), "Cannot add client-go scheme")
	kingpin.FatalIfError(apiscluster.AddToScheme(scheme), "Cannot add cluster-scoped Upbound MR APIs to scheme")
	kingpin.FatalIfError(apis.AddToScheme(scheme), "Cannot add namespace-scoped Upbound MR APIs to scheme")
	kingpin.FatalIfError(extv1.AddToScheme(scheme), "Cannot add Core API Extensions to scheme")

	mgr, err := ctrl.NewManager(ratelimiter.LimitRESTConfig(cfg, *maxReconcileRate), ctrl.Options{
		Scheme:           scheme,
		LeaderElection:   *leaderElection,
		LeaderElectionID: "crossplane-leader-election-provider-upbound",
		Cache: cache.Options{
			SyncPeriod: syncInterval,
			ByObject: map[client.Object]cache.ByObject{
				&extv1.CustomResourceDefinition{}: {
					Transform: customresourcesgate.TransformStripCRDSchema,
				},
			},
		},
		Client:                     clientOpts,
		LeaderElectionResourceLock: resourcelock.LeasesResourceLock,
		LeaseDuration:              func() *time.Duration { d := 60 * time.Second; return &d }(),
		RenewDeadline:              func() *time.Duration { d := 50 * time.Second; return &d }(),
	})
	kingpin.FatalIfError(err, "Cannot create controller manager")

	o := controller.Options{
		Logger:                  logger,
		MaxConcurrentReconciles: *maxReconcileRate,
		PollInterval:            *pollInterval,
		GlobalRateLimiter:       ratelimiter.NewGlobal(*maxReconcileRate),
		Features:                &feature.Flags{},
		Gate:                    new(gate.Gate[schema.GroupVersionKind]),
	}

	if *enableManagementPolicies {
		o.Features.Enable(features.EnableManagementPolicies)
		logger.Info("Feature enabled", "flag", features.EnableManagementPolicies)
	}

	precheckCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	canSafeStart, err := canWatchCRD(precheckCtx, mgr)
	kingpin.FatalIfError(err, "SafeStart precheck failed")
	if canSafeStart {
		kingpin.FatalIfError(upbound.SetupGated(mgr, o), "Cannot setup Upbound controllers")
		kingpin.FatalIfError(customresourcesgate.Setup(mgr, o), "Cannot setup CustomResourcesGate controller")
	} else {
		logger.Info("Provider has missing RBAC permissions for watching CRDs, controller SafeStart capability will be disabled")
		kingpin.FatalIfError(upbound.Setup(mgr, o), "Cannot setup Upbound controllers")
	}
	kingpin.FatalIfError(mgr.Start(ctrl.SetupSignalHandler()), "Cannot start controller manager")
}

func canWatchCRD(ctx context.Context, mgr manager.Manager) (bool, error) {
	if err := authv1.AddToScheme(mgr.GetScheme()); err != nil {
		return false, err
	}
	verbs := []string{"get", "list", "watch"}
	for _, verb := range verbs {
		sar := &authv1.SelfSubjectAccessReview{
			Spec: authv1.SelfSubjectAccessReviewSpec{
				ResourceAttributes: &authv1.ResourceAttributes{
					Group:    "apiextensions.k8s.io",
					Resource: "customresourcedefinitions",
					Verb:     verb,
				},
			},
		}
		if err := mgr.GetClient().Create(ctx, sar); err != nil {
			return false, errors.Wrapf(err, "unable to perform RBAC check for verb %s on CustomResourceDefinitions", verb)
		}
		if !sar.Status.Allowed {
			return false, nil
		}
	}
	return true, nil
}
