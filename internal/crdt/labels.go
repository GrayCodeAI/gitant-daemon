package crdt

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"sync"

	"github.com/lakshmanpatel/gitant/internal/persistence"
)

// Label represents a repository label with CRDT operation log
type Label struct {
	Name       string        `json:"name"`
	Color      string        `json:"color"`
	Tombstoned bool          `json:"tombstoned,omitempty"`
	log        *OperationLog `json:"-"`
}

// labelSnapshot is the JSON-serializable representation of a Label
type labelSnapshot struct {
	Name       string       `json:"name"`
	Color      string       `json:"color"`
	Tombstoned bool         `json:"tombstoned,omitempty"`
	Log        []*Operation `json:"log,omitempty"`
}

// MarshalJSON serializes a Label including its operation log
func (l *Label) MarshalJSON() ([]byte, error) {
	return json.Marshal(labelSnapshot{
		Name:       l.Name,
		Color:      l.Color,
		Tombstoned: l.Tombstoned,
		Log:        l.log.Operations(),
	})
}

// UnmarshalJSON deserializes a Label and rebuilds its operation log
func (l *Label) UnmarshalJSON(data []byte) error {
	var snap labelSnapshot
	if err := json.Unmarshal(data, &snap); err != nil {
		return err
	}
	l.Name = snap.Name
	l.Color = snap.Color
	l.Tombstoned = snap.Tombstoned
	l.log = NewOperationLog()
	for _, op := range snap.Log {
		l.log.ImportOperation(op)
	}
	return nil
}

// Log returns the operation log
func (l *Label) Log() *OperationLog {
	return l.log
}

// SetColor changes the label color
func (l *Label) SetColor(author, color string) {
	l.Color = color
	l.log.Add(&Operation{
		ID:     generateID(),
		Type:   OpSetColor,
		Author: author,
		Data:   map[string]interface{}{"color": color},
	})
}

// Tombstone marks this label as deleted
func (l *Label) Tombstone(author string) {
	l.Tombstoned = true
	l.log.Add(&Operation{
		ID:     generateID(),
		Type:   OpTombstone,
		Author: author,
	})
}

// Merge merges another label's operations into this one
func (l *Label) Merge(other *Label) {
	l.log.clock.Merge(other.log.clock)

	existingIDs := make(map[string]bool)
	for _, op := range l.log.Operations() {
		existingIDs[op.ID] = true
	}
	for _, op := range other.log.Operations() {
		if !existingIDs[op.ID] {
			l.log.ImportOperation(op)
		}
	}

	allOps := make([]*Operation, len(l.log.Operations()))
	copy(allOps, l.log.Operations())
	sort.Slice(allOps, func(a, b int) bool {
		if allOps[a].Lamport != allOps[b].Lamport {
			return allOps[a].Lamport < allOps[b].Lamport
		}
		return allOps[a].Timestamp.Before(allOps[b].Timestamp)
	})

	// Reset and replay
	l.Tombstoned = false
	l.applyOperations(allOps)
}

func (l *Label) applyOperations(ops []*Operation) {
	for _, op := range ops {
		switch op.Type {
		case OpSetColor:
			if color, ok := op.Data["color"].(string); ok {
				l.Color = color
			}
		case OpTombstone:
			l.Tombstoned = true
		}
	}
}

// LabelStore manages labels per repository using a map for O(1) lookup
type LabelStore struct {
	mu      sync.RWMutex
	dataDir string
	labels  map[string]map[string]*Label // repoID -> labelName -> Label
}

// NewLabelStore creates a new label store
func NewLabelStore(dataDir string) *LabelStore {
	return &LabelStore{
		dataDir: dataDir,
		labels:  make(map[string]map[string]*Label),
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
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.saveLocked()
}

// saveLocked persists while the caller already holds the write lock.
func (s *LabelStore) saveLocked() error {
	if s.dataDir == "" {
		return nil
	}
	return persistence.SaveJSON(filepath.Join(s.dataDir, "labels.json"), s.labels)
}

// List returns all non-tombstoned labels for a repository
func (s *LabelStore) List(repoID string) []Label {
	s.mu.RLock()
	defer s.mu.RUnlock()

	repo, ok := s.labels[repoID]
	if !ok {
		return []Label{}
	}

	result := make([]Label, 0, len(repo))
	for _, l := range repo {
		if !l.Tombstoned {
			result = append(result, *l)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})
	return result
}

// Get returns a specific label
func (s *LabelStore) Get(repoID, name string) (*Label, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	repo, ok := s.labels[repoID]
	if !ok {
		return nil, fmt.Errorf("repo not found: %s", repoID)
	}
	label, ok := repo[name]
	if !ok || label.Tombstoned {
		return nil, fmt.Errorf("label not found: %s", name)
	}
	cp := *label
	return &cp, nil
}

// Add adds a label to a repository
func (s *LabelStore) Add(repoID, name, color, author string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.labels[repoID]; !ok {
		s.labels[repoID] = make(map[string]*Label)
	}

	if existing, ok := s.labels[repoID][name]; ok && !existing.Tombstoned {
		return fmt.Errorf("label already exists: %s", name)
	}

	if color == "" {
		color = "#6b7280"
	}

	l := &Label{
		Name:  name,
		Color: color,
		log:   NewOperationLog(),
	}
	l.log.Add(&Operation{
		ID:     generateID(),
		Type:   OpCreate,
		Author: author,
		Data:   map[string]interface{}{"name": name, "color": color},
	})

	s.labels[repoID][name] = l
	return s.saveLocked()
}

// Remove tombstones a label
func (s *LabelStore) Remove(repoID, name, author string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	repo, ok := s.labels[repoID]
	if !ok {
		return fmt.Errorf("repo not found: %s", repoID)
	}
	label, ok := repo[name]
	if !ok || label.Tombstoned {
		return fmt.Errorf("label not found: %s", name)
	}

	label.Tombstone(author)
	return s.saveLocked()
}

// MergeRemote merges a remote label snapshot into the local store
func (s *LabelStore) MergeRemote(repoID string, remote *Label) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.labels[repoID]; !ok {
		s.labels[repoID] = make(map[string]*Label)
	}

	if local, ok := s.labels[repoID][remote.Name]; ok {
		local.Merge(remote)
	} else {
		cp := *remote
		s.labels[repoID][remote.Name] = &cp
	}
	return s.saveLocked()
}
