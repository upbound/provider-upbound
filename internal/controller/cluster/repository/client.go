package repository

import (
	"context"

	repos "github.com/upbound/up-sdk-go/service/repositories"
)

type RepoClient interface {
	Get(ctx context.Context, org string, repo string) (*repos.RepositoryResponse, error)

	CreateOrUpdateWithOptions(ctx context.Context, org string, repo string, opts ...repos.CreateOrUpdateOption) error

	Delete(ctx context.Context, account, name string) error
}
