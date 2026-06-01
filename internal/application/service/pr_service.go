package service

import (
	"context"

	"github.com/lakshmanpatel/gitant/internal/application/ports"
	"github.com/lakshmanpatel/gitant/internal/domain/models"
	"github.com/lakshmanpatel/gitant/internal/domain/repositories"
)

// Compile-time check.
var _ ports.PRService = (*PRServiceImpl)(nil)

// PRServiceImpl implements ports.PRService.
type PRServiceImpl struct {
	prRepo repositories.PullRequestRepository
}

// NewPRService creates a new service.
func NewPRService(prRepo repositories.PullRequestRepository) *PRServiceImpl {
	return &PRServiceImpl{prRepo: prRepo}
}

// CreatePR implements ports.PRService.
func (s *PRServiceImpl) CreatePR(ctx context.Context, req ports.CreatePRRequest) (*models.PullRequest, error) {
	return s.prRepo.Create(ctx, req.RepoID, req.Title, req.Body, req.Author, req.SourceBranch, req.TargetBranch)
}

// GetPR implements ports.PRService.
func (s *PRServiceImpl) GetPR(ctx context.Context, repoID, prID string) (*models.PullRequest, error) {
	return s.prRepo.Get(ctx, repoID, prID)
}

// ListPRs implements ports.PRService.
func (s *PRServiceImpl) ListPRs(ctx context.Context, repoID string, filters models.PRFilters) ([]*models.PullRequest, int, error) {
	return s.prRepo.List(ctx, repoID, filters)
}

// MergePR implements ports.PRService.
func (s *PRServiceImpl) MergePR(ctx context.Context, repoID, prID string) error {
	return s.prRepo.Update(ctx, repoID, prID, func(pr *models.PullRequest) error {
		pr.Status = "merged"
		return nil
	})
}
