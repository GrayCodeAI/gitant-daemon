package crdt

import (
	"fmt"
	"path/filepath"
	"sync"

	"github.com/lakshmanpatel/gitant/internal/persistence"
)

// Label represents a repository label
type Label struct {
	Name  string `json:"name"`
	Color string `json:"color"`
}

// LabelStore manages labels per repository
type LabelStore struct {
	mu       sync.RWMutex
	dataDir  string
	labels   map[string][]Label // repoID -> labels
}

// NewLabelStore creates a new label store
func NewLabelStore(dataDir string) *LabelStore {
	return &LabelStore{
		dataDir: dataDir,
		labels:  make(map[string][]Label),
	}
}

// Load loads labels from disk
func (s *LabelStore) Load() error {
	if s.dataDir == "" {
		return nil
	}
	path := filepath.Join(s.dataDir, "labels.json")
	return persistence.LoadJSON(path, &s.labels)
}

// Save persists labels to disk
func (s *LabelStore) Save() error {
	if s.dataDir == "" {
		return nil
	}
	return persistence.SaveJSON(filepath.Join(s.dataDir, "labels.json"), s.labels)
}

// List returns all labels for a repository
func (s *LabelStore) List(repoID string) []Label {
	s.mu.RLock()
	defer s.mu.RUnlock()

	labels := s.labels[repoID]
	if labels == nil {
		return []Label{}
	}
	return labels
}

// Add adds a label to a repository
func (s *LabelStore) Add(repoID, name, color string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.labels[repoID] == nil {
		s.labels[repoID] = make([]Label, 0)
	}

	// Check for duplicate
	for _, l := range s.labels[repoID] {
		if l.Name == name {
			return fmt.Errorf("label already exists: %s", name)
		}
	}

	if color == "" {
		color = "#6b7280" // default gray
	}

	s.labels[repoID] = append(s.labels[repoID], Label{Name: name, Color: color})
	return s.Save()
}

// Remove removes a label from a repository
func (s *LabelStore) Remove(repoID, name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	labels := s.labels[repoID]
	for i, l := range labels {
		if l.Name == name {
			s.labels[repoID] = append(labels[:i], labels[i+1:]...)
			return s.Save()
		}
	}

	return fmt.Errorf("label not found: %s", name)
}
