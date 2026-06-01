package adapters

import (
	"context"
	"time"

	"github.com/lakshmanpatel/gitant/internal/domain/models"
	"github.com/lakshmanpatel/gitant/internal/domain/repositories"
	"github.com/lakshmanpatel/gitant/internal/storage"
)

// Compile-time check.
var _ repositories.RepositoryRepository = (*RepositoryAdapter)(nil)

// RepositoryAdapter adapts the RepositoryRegistry to the domain RepositoryRepository interface.
type RepositoryAdapter struct {
	registry *storage.RepositoryRegistry
}

// NewRepositoryAdapter creates a new adapter.
func NewRepositoryAdapter(registry *storage.RepositoryRegistry) *RepositoryAdapter {
	return &RepositoryAdapter{registry: registry}
}

// Create implements repositories.RepositoryRepository.
func (a *RepositoryAdapter) Create(ctx context.Context, id, name, description string, private bool, owner string) (*models.Repository, error) {
	entry, err := a.registry.Create(id, name, description, private)
	if err != nil {
		return nil, err
	}
	return toDomainRepository(entry), nil
}

// Get implements repositories.RepositoryRepository.
func (a *RepositoryAdapter) Get(ctx context.Context, id string) (*models.Repository, error) {
	entry, err := a.registry.GetEntry(id)
	if err != nil {
		return nil, err
	}
	return toDomainRepository(entry), nil
}

// List implements repositories.RepositoryRepository.
func (a *RepositoryAdapter) List(ctx context.Context, filters models.RepositoryFilters) ([]*models.Repository, int, error) {
	entries := a.registry.List()
	var repos []*models.Repository
	for _, e := range entries {
		repos = append(repos, toDomainRepository(e))
	}
	return repos, len(repos), nil
}

// Delete implements repositories.RepositoryRepository.
func (a *RepositoryAdapter) Delete(ctx context.Context, id string) error {
	return a.registry.Delete(id)
}

// Star implements repositories.RepositoryRepository.
func (a *RepositoryAdapter) Star(ctx context.Context, repoID, did string) error {
	return a.registry.Star(repoID, did)
}

// Unstar implements repositories.RepositoryRepository.
func (a *RepositoryAdapter) Unstar(ctx context.Context, repoID, did string) error {
	return a.registry.Unstar(repoID, did)
}

// Fork implements repositories.RepositoryRepository.
func (a *RepositoryAdapter) Fork(ctx context.Context, sourceID, forkID, owner string) (*models.Repository, error) {
	entry, err := a.registry.Fork(sourceID, forkID, owner)
	if err != nil {
		return nil, err
	}
	return toDomainRepository(entry), nil
}

func toDomainRepository(entry *storage.RepoEntry) *models.Repository {
	created, _ := time.Parse(time.RFC3339, entry.CreatedAt)
	return &models.Repository{
		ID:          entry.ID,
		Name:        entry.Name,
		Description: entry.Description,
		Private:     entry.Private,
		CreatedAt:   created,
		// Owner not stored in RepoEntry; can be derived from path or DB
	}
}
