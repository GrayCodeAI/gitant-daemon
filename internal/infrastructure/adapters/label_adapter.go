package adapters

import (
	"context"

	"github.com/lakshmanpatel/gitant/internal/crdt"
	"github.com/lakshmanpatel/gitant/internal/domain/models"
	"github.com/lakshmanpatel/gitant/internal/domain/repositories"
)

type LabelAdapter struct {
	store *crdt.LabelStore
}

func NewLabelAdapter(store *crdt.LabelStore) repositories.LabelRepository {
	return &LabelAdapter{store: store}
}

func (a *LabelAdapter) Create(ctx context.Context, repoID, name, color string) (*models.Label, error) {
	err := a.store.Add(repoID, name, color, "system")
	if err != nil {
		return nil, err
	}
	return a.Get(ctx, repoID, name)
}

func (a *LabelAdapter) Get(ctx context.Context, repoID, labelID string) (*models.Label, error) {
	label, err := a.store.Get(repoID, labelID)
	if err != nil {
		return nil, err
	}
	return toDomainLabel(label), nil
}

func (a *LabelAdapter) List(ctx context.Context, repoID string) ([]*models.Label, error) {
	labels := a.store.List(repoID)
	result := make([]*models.Label, len(labels))
	for i, l := range labels {
		label := l // capture loop variable
		result[i] = toDomainLabel(&label)
	}
	return result, nil
}

func (a *LabelAdapter) Delete(ctx context.Context, repoID, labelID string) error {
	return a.store.Remove(repoID, labelID, "system")
}

func toDomainLabel(l *crdt.Label) *models.Label {
	return &models.Label{
		ID:     l.Name, // Labels are keyed by name
		Name:   l.Name,
		Color:  l.Color,
	}
}
