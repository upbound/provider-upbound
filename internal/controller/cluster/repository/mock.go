package repository

import (
	"context"

	repos "github.com/upbound/up-sdk-go/service/repositories"
)

type mockClient struct {
	getFn            func(ctx context.Context, org, repo string) (*repos.RepositoryResponse, error)
	createOrUpdateFn func(ctx context.Context, org, repo string, opts ...repos.CreateOrUpdateOption) error
	deleteFn         func(ctx context.Context, account, name string) error
}

func (m *mockClient) Get(ctx context.Context, org, repo string) (*repos.RepositoryResponse, error) {
	return m.getFn(ctx, org, repo)
}

func (m *mockClient) CreateOrUpdateWithOptions(ctx context.Context, org, repo string, opts ...repos.CreateOrUpdateOption) error {
	return m.createOrUpdateFn(ctx, org, repo, opts...)
}

func (m *mockClient) Delete(ctx context.Context, account, name string) error { // nolint:interfacer
	return m.deleteFn(ctx, account, name)
}
