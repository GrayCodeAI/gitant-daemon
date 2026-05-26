package operations

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Operation represents a recorded operation
type Operation struct {
	ID        string                 `json:"id"`
	RepoID    string                 `json:"repo_id"`
	Type      string                 `json:"type"` // "commit", "push", "merge", "revert", "create_branch", etc.
	User      string                 `json:"user"`
	Params    map[string]interface{} `json:"params"`
	Result    map[string]interface{} `json:"result"`
	Undoable  bool                   `json:"undoable"`
	Undone    bool                   `json:"undone"`
	Timestamp time.Time              `json:"timestamp"`
}

// Store manages operations
type Store struct {
	mu         sync.RWMutex
	baseDir    string
	operations map[string][]*Operation
}

// NewStore creates a new operations store
func NewStore(baseDir string) *Store {
	return &Store{
		baseDir:    baseDir,
		operations: make(map[string][]*Operation),
	}
}

// Load loads operations from disk
func (s *Store) Load() error {
	path := filepath.Join(s.baseDir, "operations.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return json.Unmarshal(data, &s.operations)
}

// Save saves operations to disk
func (s *Store) Save() error {
	path := filepath.Join(s.baseDir, "operations.json")
	data, err := json.MarshalIndent(s.operations, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// Record records an operation
func (s *Store) Record(op *Operation) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	op.ID = fmt.Sprintf("op-%d", time.Now().UnixNano())
	op.Timestamp = time.Now()
	op.Undoable = true

	s.operations[op.RepoID] = append(s.operations[op.RepoID], op)
	return s.Save()
}

// List lists operations for a repository
func (s *Store) List(repoID string, limit int) []*Operation {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ops := s.operations[repoID]
	if limit > 0 && len(ops) > limit {
		ops = ops[len(ops)-limit:]
	}
	return ops
}

// Get gets an operation by ID
func (s *Store) Get(repoID, opID string) (*Operation, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, op := range s.operations[repoID] {
		if op.ID == opID {
			return op, nil
		}
	}
	return nil, fmt.Errorf("operation not found")
}

// Undo undoes the last operation
func (s *Store) Undo(repoID string) (*Operation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	ops := s.operations[repoID]
	if len(ops) == 0 {
		return nil, fmt.Errorf("no operations to undo")
	}

	// Find last undoable operation
	for i := len(ops) - 1; i >= 0; i-- {
		if ops[i].Undoable && !ops[i].Undone {
			ops[i].Undone = true
			return ops[i], s.Save()
		}
	}

	return nil, fmt.Errorf("no undoable operations")
}

// Redo redoes the last undone operation
func (s *Store) Redo(repoID string) (*Operation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	ops := s.operations[repoID]
	if len(ops) == 0 {
		return nil, fmt.Errorf("no operations to redo")
	}

	// Find last undone operation
	for i := len(ops) - 1; i >= 0; i-- {
		if ops[i].Undone {
			ops[i].Undone = false
			return ops[i], s.Save()
		}
	}

	return nil, fmt.Errorf("no undone operations")
}

// GetHistory gets the operation history
func (s *Store) GetHistory(repoID string) []*Operation {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ops := s.operations[repoID]
	if ops == nil {
		return []*Operation{}
	}
	return ops
}
