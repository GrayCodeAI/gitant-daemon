package adapters

import (
	"context"

	"github.com/lakshmanpatel/gitant/internal/crdt"
	"github.com/lakshmanpatel/gitant/internal/domain/models"
	"github.com/lakshmanpatel/gitant/internal/domain/repositories"
)

type ReleaseAdapter struct {
	store *crdt.ReleaseStore
}

func NewReleaseAdapter(store *crdt.ReleaseStore) repositories.ReleaseRepository {
	return &ReleaseAdapter{store: store}
}

func (a *ReleaseAdapter) Create(ctx context.Context, repoID, tagName, name, body, author string, draft, prerelease bool) (*models.Release, error) {
	release, err := a.store.Create(repoID, tagName, name, body, author)
	if err != nil {
		return nil, err
	}
	return toDomainRelease(release), nil
}

func (a *ReleaseAdapter) Get(ctx context.Context, repoID, releaseID string) (*models.Release, error) {
	release, err := a.store.Get(repoID, releaseID)
	if err != nil {
		return nil, err
	}
	return toDomainRelease(release), nil
}

func (a *ReleaseAdapter) List(ctx context.Context, repoID string) ([]*models.Release, error) {
	releases := a.store.List(repoID)
	result := make([]*models.Release, len(releases))
	for i, r := range releases {
		result[i] = toDomainRelease(r)
	}
	return result, nil
}

func (a *ReleaseAdapter) Delete(ctx context.Context, repoID, releaseID string) error {
	return a.store.Delete(repoID, releaseID)
}

func toDomainRelease(r *crdt.Release) *models.Release {
	return &models.Release{
		ID:         r.ID,
		RepoID:     r.RepoID,
		TagName:    r.Tag,
		Name:       r.Title,
		Body:       r.Body,
		Author:     r.Author,
		Draft:      false, // CRDT doesn't track this
		Prerelease: false, // CRDT doesn't track this
		CreatedAt:  r.CreatedAt,
	}
}
