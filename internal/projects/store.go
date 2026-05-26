package projects

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Project represents a project board
type Project struct {
	ID          string    `json:"id"`
	RepoID      string    `json:"repo_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Status      string    `json:"status"` // "active", "closed", "archived"
	Columns     []Column  `json:"columns"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Column represents a project column
type Column struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Order int    `json:"order"`
	Cards []Card `json:"cards"`
}

// Card represents a project card
type Card struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Assignee    string    `json:"assignee"`
	IssueID     string    `json:"issue_id,omitempty"`
	PRID        string    `json:"pr_id,omitempty"`
	Labels      []string  `json:"labels"`
	Order       int       `json:"order"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Store manages projects
type Store struct {
	mu       sync.RWMutex
	baseDir  string
	projects map[string][]*Project // repoID -> projects
}

// NewStore creates a new project store
func NewStore(baseDir string) *Store {
	return &Store{
		baseDir:  baseDir,
		projects: make(map[string][]*Project),
	}
}

// Load loads projects from disk
func (s *Store) Load() error {
	path := filepath.Join(s.baseDir, "projects.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return json.Unmarshal(data, &s.projects)
}

// Save saves projects to disk
func (s *Store) Save() error {
	path := filepath.Join(s.baseDir, "projects.json")
	data, err := json.MarshalIndent(s.projects, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// Create creates a new project
func (s *Store) Create(project *Project) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	project.ID = fmt.Sprintf("proj-%d", time.Now().UnixNano())
	project.CreatedAt = time.Now()
	project.UpdatedAt = time.Now()
	if project.Status == "" {
		project.Status = "active"
	}

	// Create default columns if none specified
	if len(project.Columns) == 0 {
		project.Columns = []Column{
			{ID: "col-1", Name: "To Do", Order: 0, Cards: []Card{}},
			{ID: "col-2", Name: "In Progress", Order: 1, Cards: []Card{}},
			{ID: "col-3", Name: "Done", Order: 2, Cards: []Card{}},
		}
	}

	s.projects[project.RepoID] = append(s.projects[project.RepoID], project)
	return s.Save()
}

// Get gets a project by ID
func (s *Store) Get(repoID, projectID string) (*Project, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, p := range s.projects[repoID] {
		if p.ID == projectID {
			return p, nil
		}
	}
	return nil, fmt.Errorf("project not found")
}

// List lists projects for a repository
func (s *Store) List(repoID string, status string) []*Project {
	s.mu.RLock()
	defer s.mu.RUnlock()

	projects := s.projects[repoID]
	if projects == nil {
		return []*Project{}
	}

	var result []*Project
	for _, p := range projects {
		if status != "" && p.Status != status {
			continue
		}
		result = append(result, p)
	}
	return result
}

// Update updates a project
func (s *Store) Update(repoID, projectID string, fn func(*Project) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, p := range s.projects[repoID] {
		if p.ID == projectID {
			if err := fn(p); err != nil {
				return err
			}
			p.UpdatedAt = time.Now()
			return s.Save()
		}
	}
	return fmt.Errorf("project not found")
}

// AddCard adds a card to a column
func (s *Store) AddCard(repoID, projectID, columnID string, card *Card) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, p := range s.projects[repoID] {
		if p.ID == projectID {
			for i, col := range p.Columns {
				if col.ID == columnID {
					card.ID = fmt.Sprintf("card-%d", time.Now().UnixNano())
					card.CreatedAt = time.Now()
					card.UpdatedAt = time.Now()
					card.Order = len(col.Cards)
					if card.Labels == nil {
						card.Labels = []string{}
					}
					p.Columns[i].Cards = append(p.Columns[i].Cards, *card)
					p.UpdatedAt = time.Now()
					return s.Save()
				}
			}
			return fmt.Errorf("column not found")
		}
	}
	return fmt.Errorf("project not found")
}

// MoveCard moves a card between columns
func (s *Store) MoveCard(repoID, projectID, cardID, targetColumnID string, targetOrder int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, p := range s.projects[repoID] {
		if p.ID == projectID {
			// Find and remove card from source column
			var card *Card
			for i, col := range p.Columns {
				for j, c := range col.Cards {
					if c.ID == cardID {
						card = &c
						p.Columns[i].Cards = append(col.Cards[:j], col.Cards[j+1:]...)
						break
					}
				}
				if card != nil {
					break
				}
			}

			if card == nil {
				return fmt.Errorf("card not found")
			}

			// Add card to target column
			for i, col := range p.Columns {
				if col.ID == targetColumnID {
					card.Order = targetOrder
					card.UpdatedAt = time.Now()
					p.Columns[i].Cards = append(col.Cards, *card)
					p.UpdatedAt = time.Now()
					return s.Save()
				}
			}

			return fmt.Errorf("target column not found")
		}
	}
	return fmt.Errorf("project not found")
}

// DeleteCard deletes a card
func (s *Store) DeleteCard(repoID, projectID, cardID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, p := range s.projects[repoID] {
		if p.ID == projectID {
			for i, col := range p.Columns {
				for j, card := range col.Cards {
					if card.ID == cardID {
						p.Columns[i].Cards = append(col.Cards[:j], col.Cards[j+1:]...)
						p.UpdatedAt = time.Now()
						return s.Save()
					}
				}
			}
			return fmt.Errorf("card not found")
		}
	}
	return fmt.Errorf("project not found")
}

// Delete deletes a project
func (s *Store) Delete(repoID, projectID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	projects := s.projects[repoID]
	for i, p := range projects {
		if p.ID == projectID {
			s.projects[repoID] = append(projects[:i], projects[i+1:]...)
			return s.Save()
		}
	}
	return fmt.Errorf("project not found")
}
