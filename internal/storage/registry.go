package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// RepoEntry represents a repository in the registry
type RepoEntry struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Private     bool   `json:"private"`
	Path        string `json:"path"`
	CreatedAt   string `json:"created_at"`
}

// RepositoryRegistry manages multiple repositories
type RepositoryRegistry struct {
	mu       sync.RWMutex
	baseDir  string
	repos    map[string]*RepoEntry
}

// NewRepositoryRegistry creates a new repository registry
func NewRepositoryRegistry(baseDir string) (*RepositoryRegistry, error) {
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("creating base directory: %w", err)
	}

	r := &RepositoryRegistry{
		baseDir: baseDir,
		repos:   make(map[string]*RepoEntry),
	}

	// Scan for existing repos
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
	}

	return r, nil
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
		CreatedAt:   "2026-01-01T00:00:00Z",
	}
	r.repos[id] = entry

	return entry, nil
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
	return nil
}
