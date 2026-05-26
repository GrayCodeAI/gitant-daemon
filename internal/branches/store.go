package branches

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Branch represents a named branch
type Branch struct {
	Name        string    `json:"name"`
	RepoID      string    `json:"repo_id"`
	Head        string    `json:"head"` // Commit hash
	Author      string    `json:"author"`
	Description string    `json:"description"`
	Status      string    `json:"status"` // "active", "closed", "merged"
	Protected   bool      `json:"protected"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	ClosedAt    *time.Time `json:"closed_at,omitempty"`
}

// Store manages branches
type Store struct {
	mu       sync.RWMutex
	baseDir  string
	branches map[string][]*Branch
}

// NewStore creates a new branch store
func NewStore(baseDir string) *Store {
	return &Store{
		baseDir:  baseDir,
		branches: make(map[string][]*Branch),
	}
}

// Load loads branches from disk
func (s *Store) Load() error {
	path := filepath.Join(s.baseDir, "named-branches.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return json.Unmarshal(data, &s.branches)
}

// Save saves branches to disk
func (s *Store) Save() error {
	path := filepath.Join(s.baseDir, "named-branches.json")
	data, err := json.MarshalIndent(s.branches, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// Create creates a new branch
func (s *Store) Create(branch *Branch) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if branch already exists
	for _, b := range s.branches[branch.RepoID] {
		if b.Name == branch.Name && b.Status == "active" {
			return fmt.Errorf("branch already exists: %s", branch.Name)
		}
	}

	branch.CreatedAt = time.Now()
	branch.UpdatedAt = time.Now()
	branch.Status = "active"

	s.branches[branch.RepoID] = append(s.branches[branch.RepoID], branch)
	return s.Save()
}

// Get gets a branch by name
func (s *Store) Get(repoID, name string) (*Branch, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, b := range s.branches[repoID] {
		if b.Name == name {
			return b, nil
		}
	}
	return nil, fmt.Errorf("branch not found")
}

// List lists branches for a repository
func (s *Store) List(repoID, status string) []*Branch {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*Branch
	for _, b := range s.branches[repoID] {
		if status != "" && b.Status != status {
			continue
		}
		result = append(result, b)
	}
	return result
}

// Close closes a branch
func (s *Store) Close(repoID, name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, b := range s.branches[repoID] {
		if b.Name == name {
			b.Status = "closed"
			now := time.Now()
			b.ClosedAt = &now
			b.UpdatedAt = now
			return s.Save()
		}
	}
	return fmt.Errorf("branch not found")
}

// Update updates branch head
func (s *Store) Update(repoID, name, head string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, b := range s.branches[repoID] {
		if b.Name == name {
			b.Head = head
			b.UpdatedAt = time.Now()
			return s.Save()
		}
	}
	return fmt.Errorf("branch not found")
}

// Protect protects a branch
func (s *Store) Protect(repoID, name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, b := range s.branches[repoID] {
		if b.Name == name {
			b.Protected = true
			b.UpdatedAt = time.Now()
			return s.Save()
		}
	}
	return fmt.Errorf("branch not found")
}

// Unprotect unprotects a branch
func (s *Store) Unprotect(repoID, name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, b := range s.branches[repoID] {
		if b.Name == name {
			b.Protected = false
			b.UpdatedAt = time.Now()
			return s.Save()
		}
	}
	return fmt.Errorf("branch not found")
}
