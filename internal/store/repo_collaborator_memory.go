package store

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// MemoryRepoCollaboratorStore is an in-memory implementation of RepoCollaboratorStore.
type MemoryRepoCollaboratorStore struct {
	mu            sync.RWMutex
	collaborators map[string]map[string]*RepoCollaborator
}

// NewMemoryRepoCollaboratorStore creates an in-memory repo collaborator store.
func NewMemoryRepoCollaboratorStore() *MemoryRepoCollaboratorStore {
	return &MemoryRepoCollaboratorStore{
		collaborators: make(map[string]map[string]*RepoCollaborator),
	}
}

// Add adds or updates a repo collaborator membership.
func (s *MemoryRepoCollaboratorStore) Add(_ context.Context, collaborator *RepoCollaborator) error {
	if collaborator == nil {
		return fmt.Errorf("repo collaborator is required")
	}
	if collaborator.RepoID == "" {
		return fmt.Errorf("repo id is required")
	}
	if collaborator.UserID == "" {
		return fmt.Errorf("user id is required")
	}
	if collaborator.Role == "" {
		collaborator.Role = RepoRoleCollaborator
	}
	if collaborator.CreatedAt.IsZero() {
		collaborator.CreatedAt = time.Now()
	}
	collaborator.UpdatedAt = time.Now()

	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.collaborators[collaborator.RepoID]; !ok {
		s.collaborators[collaborator.RepoID] = make(map[string]*RepoCollaborator)
	}
	copy := *collaborator
	s.collaborators[collaborator.RepoID][collaborator.UserID] = &copy
	return nil
}

// Get returns a repo collaborator membership.
func (s *MemoryRepoCollaboratorStore) Get(_ context.Context, repoID, userID string) (*RepoCollaborator, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	byRepo, ok := s.collaborators[repoID]
	if !ok {
		return nil, fmt.Errorf("repo collaborator not found")
	}
	collaborator, ok := byRepo[userID]
	if !ok {
		return nil, fmt.Errorf("repo collaborator not found")
	}
	copy := *collaborator
	return &copy, nil
}

// ListByRepo lists collaborators for a repository.
func (s *MemoryRepoCollaboratorStore) ListByRepo(_ context.Context, repoID string) ([]*RepoCollaborator, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	byRepo := s.collaborators[repoID]
	result := make([]*RepoCollaborator, 0, len(byRepo))
	for _, collaborator := range byRepo {
		copy := *collaborator
		result = append(result, &copy)
	}
	return result, nil
}

// Remove removes a repo collaborator membership.
func (s *MemoryRepoCollaboratorStore) Remove(_ context.Context, repoID, userID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	byRepo, ok := s.collaborators[repoID]
	if !ok {
		return fmt.Errorf("repo collaborator not found")
	}
	if _, ok := byRepo[userID]; !ok {
		return fmt.Errorf("repo collaborator not found")
	}
	delete(byRepo, userID)
	if len(byRepo) == 0 {
		delete(s.collaborators, repoID)
	}
	return nil
}

// IsWriter reports whether the user owns or collaborates on the repo.
func (s *MemoryRepoCollaboratorStore) IsWriter(ctx context.Context, repoID, userID string) (bool, error) {
	collaborator, err := s.Get(ctx, repoID, userID)
	if err != nil {
		return false, nil
	}
	return collaborator.CanWrite(), nil
}
