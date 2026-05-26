package stacked

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Diff represents a stacked diff
type Diff struct {
	ID          string    `json:"id"`
	RepoID      string    `json:"repo_id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Author      string    `json:"author"`
	Branch      string    `json:"branch"`
	BaseCommit  string    `json:"base_commit"`
	HeadCommit  string    `json:"head_commit"`
	Status      string    `json:"status"` // "draft", "ready", "approved", "landed"
	ParentID    string    `json:"parent_id,omitempty"` // For stacking
	Children    []string  `json:"children"` // Child diff IDs
	Reviewers   []string  `json:"reviewers"`
	Comments    []Comment `json:"comments"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Comment represents a diff comment
type Comment struct {
	ID        string    `json:"id"`
	Author    string    `json:"author"`
	Body      string    `json:"body"`
	FilePath  string    `json:"file_path,omitempty"`
	Line      int       `json:"line,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// Store manages stacked diffs
type Store struct {
	mu    sync.RWMutex
	baseDir string
	diffs map[string][]*Diff
}

// NewStore creates a new stacked diff store
func NewStore(baseDir string) *Store {
	return &Store{
		baseDir: baseDir,
		diffs: make(map[string][]*Diff),
	}
}

// Load loads diffs from disk
func (s *Store) Load() error {
	path := filepath.Join(s.baseDir, "stacked-diffs.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return json.Unmarshal(data, &s.diffs)
}

// Save saves diffs to disk
func (s *Store) Save() error {
	path := filepath.Join(s.baseDir, "stacked-diffs.json")
	data, err := json.MarshalIndent(s.diffs, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// Create creates a new diff
func (s *Store) Create(diff *Diff) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	diff.ID = fmt.Sprintf("diff-%d", time.Now().UnixNano())
	diff.CreatedAt = time.Now()
	diff.UpdatedAt = time.Now()
	if diff.Children == nil {
		diff.Children = []string{}
	}
	if diff.Reviewers == nil {
		diff.Reviewers = []string{}
	}
	if diff.Comments == nil {
		diff.Comments = []Comment{}
	}

	// If parent specified, add to parent's children
	if diff.ParentID != "" {
		for _, d := range s.diffs[diff.RepoID] {
			if d.ID == diff.ParentID {
				d.Children = append(d.Children, diff.ID)
				break
			}
		}
	}

	s.diffs[diff.RepoID] = append(s.diffs[diff.RepoID], diff)
	return s.Save()
}

// Get gets a diff by ID
func (s *Store) Get(repoID, diffID string) (*Diff, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, d := range s.diffs[repoID] {
		if d.ID == diffID {
			return d, nil
		}
	}
	return nil, fmt.Errorf("diff not found")
}

// List lists diffs for a repository
func (s *Store) List(repoID, status string) []*Diff {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*Diff
	for _, d := range s.diffs[repoID] {
		if status != "" && d.Status != status {
			continue
		}
		result = append(result, d)
	}
	return result
}

// GetStack gets a stack of diffs
func (s *Store) GetStack(repoID, diffID string) ([]*Diff, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Find root of stack
	var root *Diff
	for _, d := range s.diffs[repoID] {
		if d.ID == diffID {
			root = d
			break
		}
	}

	if root == nil {
		return nil, fmt.Errorf("diff not found")
	}

	// Walk up to root
	for root.ParentID != "" {
		found := false
		for _, d := range s.diffs[repoID] {
			if d.ID == root.ParentID {
				root = d
				found = true
				break
			}
		}
		if !found {
			break
		}
	}

	// Collect stack
	var stack []*Diff
	var collect func(id string)
	collect = func(id string) {
		for _, d := range s.diffs[repoID] {
			if d.ID == id {
				stack = append(stack, d)
				for _, childID := range d.Children {
					collect(childID)
				}
				break
			}
		}
	}

	collect(root.ID)
	return stack, nil
}

// AddComment adds a comment to a diff
func (s *Store) AddComment(repoID, diffID string, comment *Comment) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, d := range s.diffs[repoID] {
		if d.ID == diffID {
			comment.ID = fmt.Sprintf("comment-%d", time.Now().UnixNano())
			comment.CreatedAt = time.Now()
			d.Comments = append(d.Comments, *comment)
			d.UpdatedAt = time.Now()
			return s.Save()
		}
	}
	return fmt.Errorf("diff not found")
}

// Land lands a diff
func (s *Store) Land(repoID, diffID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, d := range s.diffs[repoID] {
		if d.ID == diffID {
			d.Status = "landed"
			d.UpdatedAt = time.Now()
			return s.Save()
		}
	}
	return fmt.Errorf("diff not found")
}
