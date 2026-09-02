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

	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	xptest "github.com/crossplane/crossplane-runtime/v2/pkg/test"
	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"
	"github.com/google/go-cmp/cmp"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/upbound/up-sdk-go"
	upfake "github.com/upbound/up-sdk-go/fake"
	uperrors "github.com/upbound/up-sdk-go/errors"
	"github.com/upbound/up-sdk-go/service/common"
	"github.com/upbound/up-sdk-go/service/tokens"

	iamv1alpha1 "github.com/upbound/provider-upbound/apis/namespaced/iam/v1alpha1"
)

const (
	testTokenUUID       = "4654b8b5-c01d-4fbe-8800-22c347c21383"
	testTokenName       = "test-token"
	testSecretName      = "test-connection-secret"
	testTokenNamespace  = "test-namespace"
)

// tokenCR returns a namespaced Token CR with the external-name annotation set
// to testTokenUUID and, optionally, a WriteConnectionSecretToReference.
// The CR namespace is testTokenNamespace; Observe uses cr.GetNamespace() for
// the secret lookup because LocalSecretReference carries no Namespace field.
func tokenCR(withSecret bool) *iamv1alpha1.Token {
	cr := &iamv1alpha1.Token{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testTokenNamespace,
			Annotations: map[string]string{
				"crossplane.io/external-name": testTokenUUID,
			},
		},
		Spec: iamv1alpha1.TokenSpec{
			ForProvider: iamv1alpha1.TokenParameters{
				Name: testTokenName,
			},
		},
	}
	if withSecret {
		cr.Spec.ManagedResourceSpec.WriteConnectionSecretToReference = &xpv2.LocalSecretReference{
			Name: testSecretName,
		}
	}
	return cr
}

// tokensMockDoFn returns a MockDo function for the up-sdk fake client.
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

func TestObserve(t *testing.T) {
	errBoom := errors.New("boom")

	type args struct {
		mg       resource.Managed
		tokensDo func(*http.Request, interface{}) error
		kubeGet  xptest.MockGetFn
	}
	type want struct {
		o   managed.ExternalObservation
		err error
	}

	cases := map[string]struct {
		reason string
		args
		want
	}{
		"NoExternalName_ResourceDoesNotExist": {
			reason: "a CR without an external-name annotation has never been created upstream",
			args: args{
				mg: &iamv1alpha1.Token{},
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
				o:   managed.ExternalObservation{},
				err: errors.New(errConnectionSecretUnavailable),
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

			e := &external{
				kube: &xptest.MockClient{MockGet: tc.args.kubeGet},
				tokens: tokens.NewClient(up.NewConfig(func(c *up.Config) {
					c.Client = &upfake.MockClient{
						MockNewRequest: upfake.NewMockNewRequestFn(nil, nil),
						MockDo:         tc.args.tokensDo,
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
		})
	}
}
