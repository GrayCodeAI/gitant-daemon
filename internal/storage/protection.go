package storage

import (
	"fmt"
	"path/filepath"
	"sync"

	"github.com/lakshmanpatel/gitant/internal/persistence"
)

// BranchProtection defines rules for a protected branch
type BranchProtection struct {
	Branch          string `json:"branch"`
	RequirePR       bool   `json:"require_pr"`
	RequireApproval bool   `json:"require_approval"`
	NoForcePush     bool   `json:"no_force_push"`
}

// ProtectionStore manages branch protection rules per repository
type ProtectionStore struct {
	mu         sync.RWMutex
	dataDir    string
	protections map[string][]BranchProtection // repoID -> protections
}

// NewProtectionStore creates a new protection store
func NewProtectionStore(dataDir string) *ProtectionStore {
	return &ProtectionStore{
		dataDir:     dataDir,
		protections: make(map[string][]BranchProtection),
	}
}

// Load loads protections from disk
func (s *ProtectionStore) Load() error {
	if s.dataDir == "" {
		return nil
	}
	path := filepath.Join(s.dataDir, "protections.json")
	return persistence.LoadJSON(path, &s.protections)
}

// Save persists protections to disk
func (s *ProtectionStore) Save() error {
	if s.dataDir == "" {
		return nil
	}
	return persistence.SaveJSON(filepath.Join(s.dataDir, "protections.json"), s.protections)
}

// Get returns protection rules for a specific branch in a repo
func (s *ProtectionStore) Get(repoID, branch string) *BranchProtection {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, p := range s.protections[repoID] {
		if p.Branch == branch {
			return &p
		}
	}
	return nil
}

// List returns all protection rules for a repository
func (s *ProtectionStore) List(repoID string) []BranchProtection {
	s.mu.RLock()
	defer s.mu.RUnlock()

	protections := s.protections[repoID]
	if protections == nil {
		return []BranchProtection{}
	}
	return protections
}

// Set creates or updates protection rules for a branch
func (s *ProtectionStore) Set(repoID string, protection BranchProtection) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.protections[repoID] == nil {
		s.protections[repoID] = make([]BranchProtection, 0)
	}

	// Update existing or add new
	for i, p := range s.protections[repoID] {
		if p.Branch == protection.Branch {
			s.protections[repoID][i] = protection
			return s.Save()
		}
	}

	s.protections[repoID] = append(s.protections[repoID], protection)
	return s.Save()
}

// Remove removes protection rules for a branch
func (s *ProtectionStore) Remove(repoID, branch string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	protections := s.protections[repoID]
	for i, p := range protections {
		if p.Branch == branch {
			s.protections[repoID] = append(protections[:i], protections[i+1:]...)
			return s.Save()
		}
	}

	return fmt.Errorf("no protection rules found for branch: %s", branch)
}
