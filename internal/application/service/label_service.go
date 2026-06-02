package service

import (
	"context"

	"github.com/lakshmanpatel/gitant/internal/application/ports"
	"github.com/lakshmanpatel/gitant/internal/domain/models"
	"github.com/lakshmanpatel/gitant/internal/domain/repositories"
)

var _ ports.LabelService = (*LabelServiceImpl)(nil)

type LabelServiceImpl struct {
	labelRepo repositories.LabelRepository
}

func NewLabelService(labelRepo repositories.LabelRepository) *LabelServiceImpl {
	return &LabelServiceImpl{labelRepo: labelRepo}
}

func (s *LabelServiceImpl) CreateLabel(ctx context.Context, req ports.CreateLabelRequest) (*models.Label, error) {
	return s.labelRepo.Create(ctx, req.RepoID, req.Name, req.Color)
}

func (s *LabelServiceImpl) ListLabels(ctx context.Context, repoID string) ([]*models.Label, error) {
	return s.labelRepo.List(ctx, repoID)
}

func (s *LabelServiceImpl) DeleteLabel(ctx context.Context, repoID, labelID string) error {
	return s.labelRepo.Delete(ctx, repoID, labelID)
}
