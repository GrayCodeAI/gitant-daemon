package replicas

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Replica represents a repository replica on another node
type Replica struct {
	ID        string    `json:"id"`
	RepoID    string    `json:"repo_id"`
	NodeDID   string    `json:"node_did"`
	NodeURL   string    `json:"node_url"`
	Status    string    `json:"status"` // "active", "syncing", "stale", "error"
	LastSync  time.Time `json:"last_sync"`
	CreatedAt time.Time `json:"created_at"`
}

// Store manages replicas
type Store struct {
	mu       sync.RWMutex
	baseDir  string
	replicas map[string][]*Replica // repoID -> replicas
}

// NewStore creates a new replica store
func NewStore(baseDir string) *Store {
	return &Store{
		baseDir:  baseDir,
		replicas: make(map[string][]*Replica),
	}
}

// Load loads replicas from disk
func (s *Store) Load() error {
	path := filepath.Join(s.baseDir, "replicas.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return json.Unmarshal(data, &s.replicas)
}

// Save saves replicas to disk
func (s *Store) Save() error {
	path := filepath.Join(s.baseDir, "replicas.json")
	data, err := json.MarshalIndent(s.replicas, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// Register registers a replica
func (s *Store) Register(repoID, nodeDID, nodeURL string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check for existing replica
	for _, r := range s.replicas[repoID] {
		if r.NodeDID == nodeDID {
			return fmt.Errorf("replica already exists")
		}
	}

	replica := &Replica{
		ID:        fmt.Sprintf("replica-%d", time.Now().UnixNano()),
		RepoID:    repoID,
		NodeDID:   nodeDID,
		NodeURL:   nodeURL,
		Status:    "active",
		LastSync:  time.Now(),
		CreatedAt: time.Now(),
	}

	s.replicas[repoID] = append(s.replicas[repoID], replica)
	return s.Save()
}

// Unregister unregisters a replica
func (s *Store) Unregister(repoID, nodeDID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	replicas := s.replicas[repoID]
	for i, r := range replicas {
		if r.NodeDID == nodeDID {
			s.replicas[repoID] = append(replicas[:i], replicas[i+1:]...)
			return s.Save()
		}
	}
	return fmt.Errorf("replica not found")
}

// List lists replicas for a repository
func (s *Store) List(repoID string) []*Replica {
	s.mu.RLock()
	defer s.mu.RUnlock()

	replicas := s.replicas[repoID]
	if replicas == nil {
		return []*Replica{}
	}
	return replicas
}

// UpdateSync updates the last sync time for a replica
func (s *Store) UpdateSync(repoID, nodeDID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, r := range s.replicas[repoID] {
		if r.NodeDID == nodeDID {
			r.LastSync = time.Now()
			r.Status = "active"
			return s.Save()
		}
	}
	return fmt.Errorf("replica not found")
}

// MarkStale marks a replica as stale
func (s *Store) MarkStale(repoID, nodeDID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, r := range s.replicas[repoID] {
		if r.NodeDID == nodeDID {
			r.Status = "stale"
			return s.Save()
		}
	}
	return fmt.Errorf("replica not found")
}

// MarkError marks a replica as having an error
func (s *Store) MarkError(repoID, nodeDID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, r := range s.replicas[repoID] {
		if r.NodeDID == nodeDID {
			r.Status = "error"
			return s.Save()
		}
	}
	return fmt.Errorf("replica not found")
}

// GetHealthyReplicas returns healthy replicas for a repository
func (s *Store) GetHealthyReplicas(repoID string) []*Replica {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*Replica
	for _, r := range s.replicas[repoID] {
		if r.Status == "active" || r.Status == "syncing" {
			result = append(result, r)
		}
	}
	return result
}
