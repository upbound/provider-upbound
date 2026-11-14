/*
Copyright 2026 Upbound Inc.

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

package token

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/crossplane/crossplane-runtime/v2/pkg/event"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	xptest "github.com/crossplane/crossplane-runtime/v2/pkg/test"
	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/upbound/up-sdk-go"
	uperrors "github.com/upbound/up-sdk-go/errors"
	upfake "github.com/upbound/up-sdk-go/fake"
	"github.com/upbound/up-sdk-go/service/common"
	"github.com/upbound/up-sdk-go/service/tokens"

	iamv1alpha1cluster "github.com/upbound/provider-upbound/apis/cluster/iam/v1alpha1"
)

const (
	testTokenUUID       = "4654b8b5-c01d-4fbe-8800-22c347c21383"
	testTokenName       = "test-token"
	testSecretName      = "test-connection-secret"
	testSecretNamespace = "test-namespace"
)

// tokenCR returns a cluster-scoped Token CR with the external-name annotation
// set to testTokenUUID and, optionally, a WriteConnectionSecretToReference.
func tokenCR(withSecret bool) *iamv1alpha1cluster.Token {
	cr := &iamv1alpha1cluster.Token{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				"crossplane.io/external-name": testTokenUUID,
			},
		},
		Spec: iamv1alpha1cluster.TokenSpec{
			ForProvider: iamv1alpha1cluster.TokenParameters{
				Name: testTokenName,
			},
		},
	}
	if withSecret {
		cr.Spec.ClusterManagedResourceSpec.WriteConnectionSecretToReference = &xpv2.SecretReference{
			Name:      testSecretName,
			Namespace: testSecretNamespace,
		}
	}
	return cr
}

// tokensMockDo returns a MockDo function for the up-sdk fake client.
// When populate is true, it sets AttributeSet["name"] so ResourceUpToDate
// reflects the real name comparison in Observe.
func tokensMockDoFn(err error, populate bool) func(*http.Request, interface{}) error {
	return func(_ *http.Request, obj interface{}) error {
		if err != nil {
			return err
		}
		if populate {
			if r, ok := obj.(**tokens.TokenResponse); ok {
				*r = &tokens.TokenResponse{
					DataSet: common.DataSet{
						AttributeSet: common.AttributeSet{"name": testTokenName},
					},
				}
			}
		}
		return nil
	}
}

// recreateCR returns tokenCR(true) with spec.connectionSecretRecovery
// Recreate and the default management policies, modified by fns.
func recreateCR(fns ...func(*iamv1alpha1cluster.Token)) *iamv1alpha1cluster.Token {
	cr := tokenCR(true)
	cr.Spec.ConnectionSecretRecovery = iamv1alpha1cluster.ConnectionSecretRecoveryRecreate
	cr.SetManagementPolicies(xpv2.ManagementPolicies{xpv2.ManagementActionAll})
	for _, fn := range fns {
		fn(cr)
	}
	return cr
}

// tokensAPIFn answers GET like tokensMockDoFn(nil, true) and DELETE with
// deleteErr.
func tokensAPIFn(deleteErr error) func(*http.Request, interface{}) error {
	return func(req *http.Request, obj interface{}) error {
		if req.Method == http.MethodDelete {
			return deleteErr
		}
		return tokensMockDoFn(nil, true)(req, obj)
	}
}

// recorder records events.
type recorder struct{ events []event.Event }

func (r *recorder) Event(_ runtime.Object, e event.Event)      { r.events = append(r.events, e) }
func (r *recorder) WithAnnotations(_ ...string) event.Recorder { return r }

func TestObserve(t *testing.T) {
	errBoom := errors.New("boom")
	errNotFound := kerrors.NewNotFound(schema.GroupResource{Resource: "secrets"}, testSecretName)
	get := "GET /" + testTokenUUID
	del := "DELETE /" + testTokenUUID
	read := "uncached GET " + testSecretName
	recreating := []event.Event{event.Warning(reasonRecreatingToken, errors.Errorf(msgFmtRecreatingToken, testTokenUUID))}

	type args struct {
		mg       resource.Managed
		tokensDo func(*http.Request, interface{}) error
		kubeGet  xptest.MockGetFn
		apiGet   xptest.MockGetFn // uncached reads
	}
	type want struct {
		o      managed.ExternalObservation
		err    error
		calls  []string // upstream calls and uncached reads; checked if set
		events []event.Event
	}

	cases := map[string]struct {
		reason string
		args
		want
	}{
		"NoExternalName_ResourceDoesNotExist": {
			reason: "a CR without an external-name annotation has never been created upstream",
			args: args{
				mg: &iamv1alpha1cluster.Token{},
			},
			want: want{
				o: managed.ExternalObservation{ResourceExists: false},
			},
		},
		"TokenNotFound_ResourceDoesNotExist": {
			reason: "a 404 from the Upbound API means the upstream token was deleted and must be recreated",
			args: args{
				mg:       tokenCR(false),
				tokensDo: tokensMockDoFn(&uperrors.Error{Status: http.StatusNotFound}, false),
			},
			want: want{
				o: managed.ExternalObservation{ResourceExists: false},
			},
		},
		"TokenAPIError_Propagated": {
			reason: "an unexpected API error must be surfaced so the reconciler can back off",
			args: args{
				mg:       tokenCR(false),
				tokensDo: tokensMockDoFn(errBoom, false),
			},
			want: want{
				o:   managed.ExternalObservation{},
				err: errors.Wrap(errBoom, "failed to get token"),
			},
		},
		"TokenFound_NoConnectionSecretRef_Healthy": {
			reason: "a token found upstream with no connection secret reference is fully healthy",
			args: args{
				mg:       tokenCR(false),
				tokensDo: tokensMockDoFn(nil, true),
			},
			want: want{
				o: managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: true},
			},
		},
		"TokenFound_ConnectionSecretMissing_ErrorSurfaced": {
			reason: "the JWT is create-only; a missing secret means the value is unrecoverable and must be flagged",
			args: args{
				mg:       tokenCR(true),
				tokensDo: tokensMockDoFn(nil, true),
				kubeGet:  xptest.NewMockGetFn(kerrors.NewNotFound(schema.GroupResource{Resource: "secrets"}, testSecretName)),
			},
			want: want{
				o:     managed.ExternalObservation{},
				err:   errors.New(errConnectionSecretUnavailable),
				calls: []string{get},
			},
		},
		"TokenFound_ConnectionSecretEmpty_ErrorSurfaced": {
			reason: "a secret with DATA=0 is as unusable as a missing one and must be treated identically",
			args: args{
				mg:       tokenCR(true),
				tokensDo: tokensMockDoFn(nil, true),
				// MockGetFn returns nil error but does not populate Data — len(nil) == 0.
				kubeGet: xptest.NewMockGetFn(nil),
			},
			want: want{
				o:   managed.ExternalObservation{},
				err: errors.New(errConnectionSecretUnavailable),
			},
		},
		"TokenFound_ConnectionSecretExists_Healthy": {
			reason: "a token found upstream with a populated connection secret is fully healthy",
			args: args{
				mg:       tokenCR(true),
				tokensDo: tokensMockDoFn(nil, true),
				kubeGet: xptest.NewMockGetFn(nil, func(obj client.Object) error {
					obj.(*corev1.Secret).Data = map[string][]byte{"token": []byte("jwt-value")}
					return nil
				}),
			},
			want: want{
				o: managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: true},
			},
		},
		"TokenFound_ConnectionSecretGetError_Propagated": {
			reason: "an unexpected kube API error when reading the connection secret must be propagated",
			args: args{
				mg:       tokenCR(true),
				tokensDo: tokensMockDoFn(nil, true),
				kubeGet:  xptest.NewMockGetFn(errBoom),
			},
			want: want{
				o:   managed.ExternalObservation{},
				err: errors.Wrap(errBoom, errGetConnectionSecret),
			},
		},
		"TokenFound_DeletingMR_SkipsSecretCheck": {
			reason: "a Token being deleted must not be blocked by a missing connection secret; the check is skipped so Delete() can run and remove the finalizer",
			args: args{
				mg: func() *iamv1alpha1cluster.Token {
					cr := tokenCR(true)
					now := metav1.Now()
					cr.DeletionTimestamp = &now
					return cr
				}(),
				tokensDo: tokensMockDoFn(nil, true),
				kubeGet:  xptest.NewMockGetFn(kerrors.NewNotFound(schema.GroupResource{Resource: "secrets"}, testSecretName)),
			},
			want: want{
				o: managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: true},
			},
		},
		"Recreate_CachedMissUncachedHit_Healthy": {
			reason: "a lagging cache must never trigger recovery: the uncached read finds the secret",
			args: args{
				mg:       recreateCR(),
				tokensDo: tokensAPIFn(nil),
				kubeGet:  xptest.NewMockGetFn(errNotFound),
				apiGet: xptest.NewMockGetFn(nil, func(obj client.Object) error {
					obj.(*corev1.Secret).Data = map[string][]byte{"token": []byte("jwt-value")}
					return nil
				}),
			},
			want: want{o: managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: true}, calls: []string{get, read}},
		},
		"Recreate_ConfirmedMissing_DeletesOldToken": {
			reason: "a confirmed missing secret deletes the old token and reports it gone, so the reconciler creates a new one",
			args: args{
				mg:       recreateCR(func(cr *iamv1alpha1cluster.Token) { meta.SetExternalCreateSucceeded(cr, time.Now().Add(-time.Hour)) }),
				tokensDo: tokensAPIFn(nil),
				kubeGet:  xptest.NewMockGetFn(errNotFound),
				apiGet:   xptest.NewMockGetFn(errNotFound),
			},
			want: want{o: managed.ExternalObservation{ResourceExists: false}, calls: []string{get, read, del}, events: recreating},
		},
		"Recreate_EmptyTokenKey_DeleteNotFound_Recovers": {
			reason: "a secret without a token is as unusable as a missing one, and an already deleted upstream token counts as gone",
			args: args{
				mg:       recreateCR(),
				tokensDo: tokensAPIFn(&uperrors.Error{Status: http.StatusNotFound}),
				apiGet:   xptest.NewMockGetFn(nil),
			},
			want: want{o: managed.ExternalObservation{ResourceExists: false}, calls: []string{get, read, del}, events: recreating},
		},
		"Recreate_UpstreamDeleteError_Returned": {
			reason: "if the old token can't be deleted nothing changes and the error is returned",
			args: args{
				mg:       recreateCR(),
				tokensDo: tokensAPIFn(errBoom),
				kubeGet:  xptest.NewMockGetFn(errNotFound),
				apiGet:   xptest.NewMockGetFn(errNotFound),
			},
			want: want{err: errors.Wrap(errBoom, errDeleteToken), calls: []string{get, read, del}},
		},
		"Recreate_UncachedReadError_Propagated": {
			reason: "an unconfirmed miss must not trigger recovery",
			args: args{
				mg:       recreateCR(),
				tokensDo: tokensAPIFn(nil),
				kubeGet:  xptest.NewMockGetFn(errNotFound),
				apiGet:   xptest.NewMockGetFn(errBoom),
			},
			want: want{err: errors.Wrap(errBoom, errGetConnectionSecret), calls: []string{get, read}},
		},
		"Recreate_DeletionPolicyOrphan_NoRecovery": {
			reason: "with deletionPolicy Orphan the provider must not delete the upstream token",
			args: args{
				mg:       recreateCR(func(cr *iamv1alpha1cluster.Token) { cr.SetDeletionPolicy(xpv2.DeletionOrphan) }),
				tokensDo: tokensAPIFn(nil),
				kubeGet:  xptest.NewMockGetFn(errNotFound),
				apiGet:   xptest.NewMockGetFn(errNotFound),
			},
			want: want{err: errors.New(errRecreateNotAllowed), calls: []string{get, read}},
		},
		"Recreate_ManagementPoliciesWithoutDelete_NoRecovery": {
			reason: "management policies without Delete forbid deleting the upstream token",
			args: args{
				mg: recreateCR(func(cr *iamv1alpha1cluster.Token) {
					cr.SetManagementPolicies(xpv2.ManagementPolicies{xpv2.ManagementActionObserve, xpv2.ManagementActionCreate, xpv2.ManagementActionUpdate, xpv2.ManagementActionLateInitialize})
				}),
				tokensDo: tokensAPIFn(nil),
				kubeGet:  xptest.NewMockGetFn(errNotFound),
				apiGet:   xptest.NewMockGetFn(errNotFound),
			},
			want: want{err: errors.New(errRecreateNotAllowed), calls: []string{get, read}},
		},
		"Recreate_CreatedRecently_NoRecovery": {
			reason: "a token created within the cooldown is not recreated again, so a secret that can't be published doesn't cause a recreate loop",
			args: args{
				mg:       recreateCR(func(cr *iamv1alpha1cluster.Token) { meta.SetExternalCreateSucceeded(cr, time.Now().Add(-time.Minute)) }),
				tokensDo: tokensAPIFn(nil),
				kubeGet:  xptest.NewMockGetFn(errNotFound),
				apiGet:   xptest.NewMockGetFn(errNotFound),
			},
			want: want{err: errors.New(errRecreatedRecently), calls: []string{get, read}},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			// Default mocks for cases that do not exercise the relevant path.
			if tc.args.tokensDo == nil {
				tc.args.tokensDo = tokensMockDoFn(nil, false)
			}
			if tc.args.kubeGet == nil {
				tc.args.kubeGet = xptest.NewMockGetFn(nil)
			}

			var calls []string
			rec := &recorder{}
			e := &external{
				kube: &xptest.MockClient{MockGet: tc.args.kubeGet},
				apiReader: &xptest.MockClient{MockGet: func(ctx context.Context, key client.ObjectKey, obj client.Object) error {
					calls = append(calls, "uncached GET "+key.Name)
					return tc.args.apiGet(ctx, key, obj)
				}},
				recorder:                  rec,
				managementPoliciesEnabled: true,
				tokens: tokens.NewClient(up.NewConfig(func(c *up.Config) {
					c.Client = &upfake.MockClient{
						MockNewRequest: func(ctx context.Context, method, _, urlPath string, _ interface{}) (*http.Request, error) {
							return http.NewRequestWithContext(ctx, method, "/"+urlPath, nil)
						},
						MockDo: func(req *http.Request, obj interface{}) error {
							calls = append(calls, req.Method+" "+req.URL.Path)
							return tc.args.tokensDo(req, obj)
						},
					}
				})),
			}

			got, err := e.Observe(context.Background(), tc.args.mg)

			if diff := cmp.Diff(tc.want.err, err, xptest.EquateErrors()); diff != "" {
				t.Errorf("\n%s\nObserve(...): -want error, +got error:\n%s", tc.reason, diff)
			}
			if diff := cmp.Diff(tc.want.o, got); diff != "" {
				t.Errorf("\n%s\nObserve(...): -want observation, +got observation:\n%s", tc.reason, diff)
			}
			if diff := cmp.Diff(tc.want.calls, calls); tc.want.calls != nil && diff != "" {
				t.Errorf("\n%s\nObserve(...): -want upstream calls, +got upstream calls:\n%s", tc.reason, diff)
			}
			if diff := cmp.Diff(tc.want.events, rec.events, cmpopts.EquateEmpty()); diff != "" {
				t.Errorf("\n%s\nObserve(...): -want events, +got events:\n%s", tc.reason, diff)
			}
		})
	}
}
