package epics

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Epic represents an epic (collection of issues)
type Epic struct {
	ID          string    `json:"id"`
	RepoID      string    `json:"repo_id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Status      string    `json:"status"` // "open", "closed", "in_progress"
	Author      string    `json:"author"`
	Assignee    string    `json:"assignee"`
	Labels      []string  `json:"labels"`
	Issues      []string  `json:"issues"` // Issue IDs
	StartDate   *time.Time `json:"start_date,omitempty"`
	EndDate     *time.Time `json:"end_date,omitempty"`
	Progress    float64   `json:"progress"` // 0.0 to 1.0
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Store manages epics
type Store struct {
	mu     sync.RWMutex
	baseDir string
	epics  map[string][]*Epic
}

// NewStore creates a new epic store
func NewStore(baseDir string) *Store {
	return &Store{
		baseDir: baseDir,
		epics:  make(map[string][]*Epic),
	}
}

// Load loads epics from disk
func (s *Store) Load() error {
	path := filepath.Join(s.baseDir, "epics.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return json.Unmarshal(data, &s.epics)
}

// Save saves epics to disk
func (s *Store) Save() error {
	path := filepath.Join(s.baseDir, "epics.json")
	data, err := json.MarshalIndent(s.epics, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// Create creates a new epic
func (s *Store) Create(epic *Epic) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	epic.ID = fmt.Sprintf("epic-%d", time.Now().UnixNano())
	epic.CreatedAt = time.Now()
	epic.UpdatedAt = time.Now()
	if epic.Issues == nil {
		epic.Issues = []string{}
	}
	if epic.Labels == nil {
		epic.Labels = []string{}
	}

	s.epics[epic.RepoID] = append(s.epics[epic.RepoID], epic)
	return s.Save()
}

// Get gets an epic by ID
func (s *Store) Get(repoID, epicID string) (*Epic, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, e := range s.epics[repoID] {
		if e.ID == epicID {
			return e, nil
		}
	}
	return nil, fmt.Errorf("epic not found")
}

// List lists epics for a repository
func (s *Store) List(repoID, status string) []*Epic {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*Epic
	for _, e := range s.epics[repoID] {
		if status != "" && e.Status != status {
			continue
		}
		result = append(result, e)
	}
	return result
}

// AddIssue adds an issue to an epic
func (s *Store) AddIssue(repoID, epicID, issueID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, e := range s.epics[repoID] {
		if e.ID == epicID {
			// Check if already exists
			for _, id := range e.Issues {
				if id == issueID {
					return fmt.Errorf("issue already in epic")
				}
			}
			e.Issues = append(e.Issues, issueID)
			e.UpdatedAt = time.Now()
			return s.Save()
		}
	}
	return fmt.Errorf("epic not found")
}

// RemoveIssue removes an issue from an epic
func (s *Store) RemoveIssue(repoID, epicID, issueID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, e := range s.epics[repoID] {
		if e.ID == epicID {
			for i, id := range e.Issues {
				if id == issueID {
					e.Issues = append(e.Issues[:i], e.Issues[i+1:]...)
					e.UpdatedAt = time.Now()
					return s.Save()
				}
			}
			return fmt.Errorf("issue not in epic")
		}
	}
	return fmt.Errorf("epic not found")
}

// UpdateProgress updates epic progress
func (s *Store) UpdateProgress(repoID, epicID string, progress float64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, e := range s.epics[repoID] {
		if e.ID == epicID {
			e.Progress = progress
			e.UpdatedAt = time.Now()
			return s.Save()
		}
	}
	return fmt.Errorf("epic not found")
}

// Close closes an epic
func (s *Store) Close(repoID, epicID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, e := range s.epics[repoID] {
		if e.ID == epicID {
			e.Status = "closed"
			e.Progress = 1.0
			e.UpdatedAt = time.Now()
			return s.Save()
		}
	}
	return fmt.Errorf("epic not found")
}
