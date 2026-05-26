package technotes

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Technote represents a technical note
type Technote struct {
	ID        string    `json:"id"`
	RepoID    string    `json:"repo_id"`
	Title     string    `json:"title"`
	Body      string    `json:"body"`
	Author    string    `json:"author"`
	Tags      []string  `json:"tags"`
	Ref       string    `json:"ref"` // Git ref (commit, branch, tag)
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Store manages technotes
type Store struct {
	mu        sync.RWMutex
	baseDir   string
	technotes map[string][]*Technote
}

// NewStore creates a new technotes store
func NewStore(baseDir string) *Store {
	return &Store{
		baseDir:   baseDir,
		technotes: make(map[string][]*Technote),
	}
}

// Load loads technotes from disk
func (s *Store) Load() error {
	path := filepath.Join(s.baseDir, "technotes.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return json.Unmarshal(data, &s.technotes)
}

// Save saves technotes to disk
func (s *Store) Save() error {
	path := filepath.Join(s.baseDir, "technotes.json")
	data, err := json.MarshalIndent(s.technotes, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// Create creates a new technote
func (s *Store) Create(note *Technote) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	note.ID = fmt.Sprintf("note-%d", time.Now().UnixNano())
	note.CreatedAt = time.Now()
	note.UpdatedAt = time.Now()
	if note.Tags == nil {
		note.Tags = []string{}
	}

	s.technotes[note.RepoID] = append(s.technotes[note.RepoID], note)
	return s.Save()
}

// Get gets a technote by ID
func (s *Store) Get(repoID, noteID string) (*Technote, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, n := range s.technotes[repoID] {
		if n.ID == noteID {
			return n, nil
		}
	}
	return nil, fmt.Errorf("technote not found")
}

// List lists technotes for a repository
func (s *Store) List(repoID, tag string) []*Technote {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*Technote
	for _, n := range s.technotes[repoID] {
		if tag != "" {
			found := false
			for _, t := range n.Tags {
				if t == tag {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}
		result = append(result, n)
	}
	return result
}

// Update updates a technote
func (s *Store) Update(repoID, noteID string, fn func(*Technote) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, n := range s.technotes[repoID] {
		if n.ID == noteID {
			if err := fn(n); err != nil {
				return err
			}
			n.UpdatedAt = time.Now()
			return s.Save()
		}
	}
	return fmt.Errorf("technote not found")
}

// Delete deletes a technote
func (s *Store) Delete(repoID, noteID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	notes := s.technotes[repoID]
	for i, n := range notes {
		if n.ID == noteID {
			s.technotes[repoID] = append(notes[:i], notes[i+1:]...)
			return s.Save()
		}
	}
	return fmt.Errorf("technote not found")
}
