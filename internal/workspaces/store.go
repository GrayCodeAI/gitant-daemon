package workspaces

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Workspace represents a development workspace
type Workspace struct {
	ID          string    `json:"id"`
	RepoID      string    `json:"repo_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Branch      string    `json:"branch"`
	Status      string    `json:"status"` // "active", "stopped", "building"
	Image       string    `json:"image"`  // Docker image
	Resources   Resources `json:"resources"`
	Environment map[string]string `json:"environment"`
	Ports       []int     `json:"ports"`
	Owner       string    `json:"owner"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Resources represents workspace resources
type Resources struct {
	CPU    string `json:"cpu"`    // "1", "2", "4"
	Memory string `json:"memory"` // "1Gi", "2Gi", "4Gi"
	Disk   string `json:"disk"`   // "10Gi", "20Gi"
}

// Store manages workspaces
type Store struct {
	mu         sync.RWMutex
	baseDir    string
	workspaces map[string][]*Workspace
}

// NewStore creates a new workspace store
func NewStore(baseDir string) *Store {
	return &Store{
		baseDir:    baseDir,
		workspaces: make(map[string][]*Workspace),
	}
}

// Load loads workspaces from disk
func (s *Store) Load() error {
	path := filepath.Join(s.baseDir, "workspaces.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return json.Unmarshal(data, &s.workspaces)
}

// Save saves workspaces to disk
func (s *Store) Save() error {
	path := filepath.Join(s.baseDir, "workspaces.json")
	data, err := json.MarshalIndent(s.workspaces, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// Create creates a new workspace
func (s *Store) Create(ws *Workspace) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	ws.ID = fmt.Sprintf("ws-%d", time.Now().UnixNano())
	ws.CreatedAt = time.Now()
	ws.UpdatedAt = time.Now()
	ws.Status = "active"
	if ws.Environment == nil {
		ws.Environment = make(map[string]string)
	}

	s.workspaces[ws.RepoID] = append(s.workspaces[ws.RepoID], ws)
	return s.Save()
}

// Get gets a workspace by ID
func (s *Store) Get(repoID, wsID string) (*Workspace, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, ws := range s.workspaces[repoID] {
		if ws.ID == wsID {
			return ws, nil
		}
	}
	return nil, fmt.Errorf("workspace not found")
}

// List lists workspaces for a repository
func (s *Store) List(repoID string) []*Workspace {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.workspaces[repoID]
}

// Stop stops a workspace
func (s *Store) Stop(repoID, wsID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, ws := range s.workspaces[repoID] {
		if ws.ID == wsID {
			ws.Status = "stopped"
			ws.UpdatedAt = time.Now()
			return s.Save()
		}
	}
	return fmt.Errorf("workspace not found")
}

// Start starts a workspace
func (s *Store) Start(repoID, wsID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, ws := range s.workspaces[repoID] {
		if ws.ID == wsID {
			ws.Status = "active"
			ws.UpdatedAt = time.Now()
			return s.Save()
		}
	}
	return fmt.Errorf("workspace not found")
}

// Delete deletes a workspace
func (s *Store) Delete(repoID, wsID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	workspaces := s.workspaces[repoID]
	for i, ws := range workspaces {
		if ws.ID == wsID {
			s.workspaces[repoID] = append(workspaces[:i], workspaces[i+1:]...)
			return s.Save()
		}
	}
	return fmt.Errorf("workspace not found")
}
