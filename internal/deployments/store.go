package deployments

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Deployment represents a deployment
type Deployment struct {
	ID          string            `json:"id"`
	RepoID      string            `json:"repo_id"`
	Environment string            `json:"environment"`
	Ref         string            `json:"ref"` // Branch, tag, or SHA
	Status      string            `json:"status"` // "pending", "success", "failure", "cancelled"
	Deployer    string            `json:"deployer"`
	URL         string            `json:"url"`
	Metadata    map[string]string `json:"metadata"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
	DeployedAt  *time.Time        `json:"deployed_at,omitempty"`
}

// Environment represents a deployment environment
type Environment struct {
	Name        string            `json:"name"`
	RepoID      string            `json:"repo_id"`
	URL         string            `json:"url"`
	Status      string            `json:"status"` // "available", "stopped"
	Variables   map[string]string `json:"variables"`
	Protection  bool              `json:"protection"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

// Store manages deployments
type Store struct {
	mu           sync.RWMutex
	baseDir      string
	deployments  map[string][]*Deployment
	environments map[string][]*Environment
}

// NewStore creates a new deployment store
func NewStore(baseDir string) *Store {
	return &Store{
		baseDir:      baseDir,
		deployments:  make(map[string][]*Deployment),
		environments: make(map[string][]*Environment),
	}
}

// Load loads deployments from disk
func (s *Store) Load() error {
	path := filepath.Join(s.baseDir, "deployments.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return json.Unmarshal(data, &s.deployments)
}

// Save saves deployments to disk
func (s *Store) Save() error {
	path := filepath.Join(s.baseDir, "deployments.json")
	data, err := json.MarshalIndent(s.deployments, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// CreateDeployment creates a new deployment
func (s *Store) CreateDeployment(deploy *Deployment) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	deploy.ID = fmt.Sprintf("deploy-%d", time.Now().UnixNano())
	deploy.CreatedAt = time.Now()
	deploy.UpdatedAt = time.Now()
	deploy.Status = "pending"
	if deploy.Metadata == nil {
		deploy.Metadata = make(map[string]string)
	}

	s.deployments[deploy.RepoID] = append(s.deployments[deploy.RepoID], deploy)
	return s.Save()
}

// GetDeployment gets a deployment by ID
func (s *Store) GetDeployment(repoID, deployID string) (*Deployment, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, d := range s.deployments[repoID] {
		if d.ID == deployID {
			return d, nil
		}
	}
	return nil, fmt.Errorf("deployment not found")
}

// ListDeployments lists deployments for a repository
func (s *Store) ListDeployments(repoID, environment string) []*Deployment {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*Deployment
	for _, d := range s.deployments[repoID] {
		if environment != "" && d.Environment != environment {
			continue
		}
		result = append(result, d)
	}
	return result
}

// UpdateDeploymentStatus updates deployment status
func (s *Store) UpdateDeploymentStatus(repoID, deployID, status string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, d := range s.deployments[repoID] {
		if d.ID == deployID {
			d.Status = status
			d.UpdatedAt = time.Now()
			if status == "success" {
				now := time.Now()
				d.DeployedAt = &now
			}
			return s.Save()
		}
	}
	return fmt.Errorf("deployment not found")
}

// CreateEnvironment creates a new environment
func (s *Store) CreateEnvironment(env *Environment) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	env.CreatedAt = time.Now()
	env.UpdatedAt = time.Now()
	env.Status = "available"
	if env.Variables == nil {
		env.Variables = make(map[string]string)
	}

	s.environments[env.RepoID] = append(s.environments[env.RepoID], env)
	return s.Save()
}

// GetEnvironment gets an environment by name
func (s *Store) GetEnvironment(repoID, name string) (*Environment, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, e := range s.environments[repoID] {
		if e.Name == name {
			return e, nil
		}
	}
	return nil, fmt.Errorf("environment not found")
}

// ListEnvironments lists environments for a repository
func (s *Store) ListEnvironments(repoID string) []*Environment {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.environments[repoID]
}
