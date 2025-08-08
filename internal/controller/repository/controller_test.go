package repository

import (
	"context"
	"errors"
	"github.com/upbound/provider-upbound/internal/controller"
	"testing"

	managed "github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	cmp "github.com/google/go-cmp/cmp"
	cmpopts "github.com/google/go-cmp/cmp/cmpopts"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	"github.com/upbound/up-sdk-go/service/repositories"
	repos "github.com/upbound/up-sdk-go/service/repositories"

	"github.com/upbound/provider-upbound/apis/repository/v1alpha1"
)

func TestObserve(t *testing.T) {
	type args struct {
		mg resource.Managed
	}
	type want struct {
		o   managed.ExternalObservation
		err error
	}

	cases := map[string]struct {
		setupMocks func(m *controller.mockClient)
		args       args
		want       want
	}{
		"ExternalNameEmpty": {
			args: args{
				mg: &v1alpha1.Repository{},
			},
			want: want{
				o: managed.ExternalObservation{
					ResourceExists: false,
				},
				err: nil,
			},
		},
		"RepoNotFound": {
			setupMocks: func(m *controller.mockClient) {
				m.getFn = func(ctx context.Context, org, repo string) (*repos.RepositoryResponse, error) {
					return nil, errors.New("not found")
				}
			},
			args: args{
				mg: &v1alpha1.Repository{
					Spec: v1alpha1.RepositorySpec{
						ForProvider: v1alpha1.RepositoryParameters{
							OrganizationName: "org",
						},
					},
				},
			},
			want: want{
				o:   managed.ExternalObservation{},
				err: nil,
			},
		},
		"RepoUpToDate": {
			setupMocks: func(m *controller.mockClient) {
				m.getFn = func(ctx context.Context, org, repo string) (*repos.RepositoryResponse, error) {
					return &repos.RepositoryResponse{
						Repository: repos.Repository{
							Public:  true,
							Publish: ptr.To(repositories.PublishPolicy("publish")),
						},
					}, nil
				}
			},
			args: args{
				mg: &v1alpha1.Repository{
					Spec: v1alpha1.RepositorySpec{
						ForProvider: v1alpha1.RepositoryParameters{
							OrganizationName: "org",
							Public:           true,
							Publish:          true,
						},
					},
					ObjectMeta: v1.ObjectMeta{
						Annotations: map[string]string{"crossplane.io/external-name": "name"},
					},
				},
			},
			want: want{
				o: managed.ExternalObservation{
					ResourceExists:   true,
					ResourceUpToDate: true,
				},
				err: nil,
			},
		},
		"RepoOutOfSync": {
			setupMocks: func(m *controller.mockClient) {
				m.getFn = func(ctx context.Context, org, repo string) (*repos.RepositoryResponse, error) {
					return &repos.RepositoryResponse{
						Repository: repos.Repository{
							Public:  false,
							Publish: ptr.To(repositories.PublishPolicy("draft")),
						},
					}, nil
				}
			},
			args: args{
				mg: &v1alpha1.Repository{
					Spec: v1alpha1.RepositorySpec{
						ForProvider: v1alpha1.RepositoryParameters{
							OrganizationName: "org",
							Public:           true,
							Publish:          true,
						},
					},
					ObjectMeta: v1.ObjectMeta{
						Annotations: map[string]string{"crossplane.io/external-name": "name"},
					},
				},
			},
			want: want{
				o: managed.ExternalObservation{
					ResourceExists:   true,
					ResourceUpToDate: false,
				},
				err: nil,
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			mockClient := &controller.mockClient{}
			if tc.setupMocks != nil {
				tc.setupMocks(mockClient)
			}
			e := &external{repositories: mockClient}

			got, err := e.Observe(context.Background(), tc.args.mg)
			if diff := cmp.Diff(tc.want.err, err, cmpopts.EquateErrors()); diff != "" {
				t.Errorf("unexpected error (-want, +got):\n%s", diff)
			}
			if diff := cmp.Diff(tc.want.o, got); diff != "" {
				t.Errorf("unexpected observation (-want, +got):\n%s", diff)
			}
		})
	}
}
