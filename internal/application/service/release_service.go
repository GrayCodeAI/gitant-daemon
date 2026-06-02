package service

import (
	"context"

	"github.com/lakshmanpatel/gitant/internal/application/ports"
	"github.com/lakshmanpatel/gitant/internal/domain/models"
	"github.com/lakshmanpatel/gitant/internal/domain/repositories"
)

var _ ports.ReleaseService = (*ReleaseServiceImpl)(nil)

type ReleaseServiceImpl struct {
	releaseRepo repositories.ReleaseRepository
}

func NewReleaseService(releaseRepo repositories.ReleaseRepository) *ReleaseServiceImpl {
	return &ReleaseServiceImpl{releaseRepo: releaseRepo}
}

func (s *ReleaseServiceImpl) CreateRelease(ctx context.Context, req ports.CreateReleaseRequest) (*models.Release, error) {
	return s.releaseRepo.Create(ctx, req.RepoID, req.TagName, req.Name, req.Body, req.Author, req.Draft, req.Prerelease)
}

func (s *ReleaseServiceImpl) GetRelease(ctx context.Context, repoID, releaseID string) (*models.Release, error) {
	return s.releaseRepo.Get(ctx, repoID, releaseID)
}

func (s *ReleaseServiceImpl) ListReleases(ctx context.Context, repoID string) ([]*models.Release, error) {
	return s.releaseRepo.List(ctx, repoID)
}
