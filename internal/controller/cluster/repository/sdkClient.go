package repository

import (
	"context"

	repos "github.com/upbound/up-sdk-go/service/repositories"
)

type sdkClient struct {
	upstream *repos.Client
}

func (s *sdkClient) Get(ctx context.Context, org, repo string) (*repos.RepositoryResponse, error) {
	return s.upstream.Get(ctx, org, repo)
}

func (s *sdkClient) CreateOrUpdateWithOptions(ctx context.Context, org, repo string, opts ...repos.CreateOrUpdateOption) error {
	return s.upstream.CreateOrUpdateWithOptions(ctx, org, repo, opts...)
}

func (s *sdkClient) Delete(ctx context.Context, account, name string) error { // nolint:interfacer
	return s.upstream.Delete(ctx, account, name)
}
