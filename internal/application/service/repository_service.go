package service

import (
	"context"

	"github.com/lakshmanpatel/gitant/internal/application/ports"
	"github.com/lakshmanpatel/gitant/internal/domain/models"
	"github.com/lakshmanpatel/gitant/internal/domain/repositories"
)

// Compile-time check.
var _ ports.RepositoryService = (*RepositoryServiceImpl)(nil)

// RepositoryServiceImpl implements ports.RepositoryService.
type RepositoryServiceImpl struct {
	repoRepo repositories.RepositoryRepository
}

// NewRepositoryService creates a new service.
func NewRepositoryService(repoRepo repositories.RepositoryRepository) *RepositoryServiceImpl {
	return &RepositoryServiceImpl{repoRepo: repoRepo}
}

// CreateRepository implements ports.RepositoryService.
func (s *RepositoryServiceImpl) CreateRepository(ctx context.Context, req ports.CreateRepoRequest) (*models.Repository, error) {
	return s.repoRepo.Create(ctx, req.Name, req.Name, req.Description, req.Private, req.Owner)
}

// GetRepository implements ports.RepositoryService.
func (s *RepositoryServiceImpl) GetRepository(ctx context.Context, id string) (*models.Repository, error) {
	return s.repoRepo.Get(ctx, id)
}

// ListRepositories implements ports.RepositoryService.
func (s *RepositoryServiceImpl) ListRepositories(ctx context.Context, filters models.RepositoryFilters) ([]*models.Repository, int, error) {
	return s.repoRepo.List(ctx, filters)
}

// DeleteRepository implements ports.RepositoryService.
func (s *RepositoryServiceImpl) DeleteRepository(ctx context.Context, id string) error {
	return s.repoRepo.Delete(ctx, id)
}

// StarRepository implements ports.RepositoryService.
func (s *RepositoryServiceImpl) StarRepository(ctx context.Context, repoID, did string) error {
	return s.repoRepo.Star(ctx, repoID, did)
}

// UnstarRepository implements ports.RepositoryService.
func (s *RepositoryServiceImpl) UnstarRepository(ctx context.Context, repoID, did string) error {
	return s.repoRepo.Unstar(ctx, repoID, did)
}

// ForkRepository implements ports.RepositoryService.
func (s *RepositoryServiceImpl) ForkRepository(ctx context.Context, req ports.ForkRepoRequest) (*models.Repository, error) {
	return s.repoRepo.Fork(ctx, req.SourceID, req.ForkName, req.Owner)
}
