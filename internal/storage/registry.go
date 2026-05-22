package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/lakshmanpatel/gitant/internal/persistence"
)

// RepoEntry represents a repository in the registry
type RepoEntry struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Private     bool     `json:"private"`
	Path        string   `json:"path"`
	CreatedAt   string   `json:"created_at"`
	Stars       int      `json:"stars"`
	StarredBy   []string `json:"starred_by"`
}

// RepositoryRegistry manages multiple repositories
type RepositoryRegistry struct {
	mu       sync.RWMutex
	baseDir  string
	dataDir  string
	repos    map[string]*RepoEntry
}

// NewRepositoryRegistry creates a new repository registry
func NewRepositoryRegistry(baseDir, dataDir string) (*RepositoryRegistry, error) {
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("creating base directory: %w", err)
	}

	r := &RepositoryRegistry{
		baseDir: baseDir,
		dataDir: dataDir,
		repos:   make(map[string]*RepoEntry),
	}

	// Try loading persisted metadata first
	if dataDir != "" {
		registryPath := filepath.Join(dataDir, "registry.json")
		if err := persistence.LoadJSON(registryPath, &r.repos); err == nil && len(r.repos) > 0 {
			return r, nil
		}
	}

	// Fallback: scan for existing repos
	entries, err := os.ReadDir(baseDir)
	if err == nil {
		for _, entry := range entries {
			if entry.IsDir() {
				repoPath := filepath.Join(baseDir, entry.Name())
				if _, err := OpenRepository(repoPath); err == nil {
					r.repos[entry.Name()] = &RepoEntry{
						ID:   entry.Name(),
						Name: entry.Name(),
						Path: repoPath,
					}
				}
			}
		}
		// Save scanned data so metadata persists
		r.Save()
	}

	return r, nil
}

// Save persists the registry to disk
func (r *RepositoryRegistry) Save() error {
	if r.dataDir == "" {
		return nil
	}
	return persistence.SaveJSON(filepath.Join(r.dataDir, "registry.json"), r.repos)
}

// Create creates a new repository
func (r *RepositoryRegistry) Create(id, name, description string, private bool) (*RepoEntry, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.repos[id]; ok {
		return nil, fmt.Errorf("repository already exists: %s", id)
	}

	repoPath := filepath.Join(r.baseDir, id)
	if _, err := InitRepository(repoPath); err != nil {
		return nil, fmt.Errorf("initializing repository: %w", err)
	}

	entry := &RepoEntry{
		ID:          id,
		Name:        name,
		Description: description,
		Private:     private,
		Path:        repoPath,
		CreatedAt:   time.Now().Format(time.RFC3339),
	}
	r.repos[id] = entry

	return entry, r.Save()
}

// Open opens an existing repository
func (r *RepositoryRegistry) Open(id string) (*Repository, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	entry, ok := r.repos[id]
	if !ok {
		return nil, fmt.Errorf("repository not found: %s", id)
	}

	return OpenRepository(entry.Path)
}

// GetEntry returns the registry entry for a repository
func (r *RepositoryRegistry) GetEntry(id string) (*RepoEntry, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	entry, ok := r.repos[id]
	if !ok {
		return nil, fmt.Errorf("repository not found: %s", id)
	}

	return entry, nil
}

// List returns all repositories
func (r *RepositoryRegistry) List() []*RepoEntry {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]*RepoEntry, 0, len(r.repos))
	for _, entry := range r.repos {
		result = append(result, entry)
	}
	return result
}

// Star adds a star to a repository
func (r *RepositoryRegistry) Star(id, did string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	entry, ok := r.repos[id]
	if !ok {
		return fmt.Errorf("repository not found: %s", id)
	}

	if entry.StarredBy == nil {
		entry.StarredBy = make([]string, 0)
	}

	for _, d := range entry.StarredBy {
		if d == did {
			return nil // already starred
		}
	}

	entry.StarredBy = append(entry.StarredBy, did)
	entry.Stars = len(entry.StarredBy)
	return r.Save()
}

// Unstar removes a star from a repository
func (r *RepositoryRegistry) Unstar(id, did string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	entry, ok := r.repos[id]
	if !ok {
		return fmt.Errorf("repository not found: %s", id)
	}

	for i, d := range entry.StarredBy {
		if d == did {
			entry.StarredBy = append(entry.StarredBy[:i], entry.StarredBy[i+1:]...)
			entry.Stars = len(entry.StarredBy)
			return r.Save()
		}
	}

	return nil // not starred
}

// Delete removes a repository
func (r *RepositoryRegistry) Delete(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	entry, ok := r.repos[id]
	if !ok {
		return fmt.Errorf("repository not found: %s", id)
	}

	if err := os.RemoveAll(entry.Path); err != nil {
		return fmt.Errorf("removing repository: %w", err)
	}

	delete(r.repos, id)
	return r.Save()
}
