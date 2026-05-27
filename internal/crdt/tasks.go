package crdt

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/lakshmanpatel/gitant/internal/persistence"
)

// TaskStatus represents the status of a task
type TaskStatus string

const (
	TaskOpen      TaskStatus = "open"
	TaskClaimed   TaskStatus = "claimed"
	TaskCompleted TaskStatus = "completed"
	TaskFailed    TaskStatus = "failed"
)

// Task represents an agent task with CRDT operation log
type Task struct {
	ID          string        `json:"id"`
	RepoID      string        `json:"repo_id"`
	Title       string        `json:"title"`
	Description string        `json:"description"`
	Status      TaskStatus    `json:"status"`
	ClaimedBy   string        `json:"claimed_by"`
	CreatedBy   string        `json:"created_by"`
	CreatedAt   time.Time     `json:"created_at"`
	ClaimedAt   *time.Time    `json:"claimed_at,omitempty"`
	CompletedAt *time.Time    `json:"completed_at,omitempty"`
	Result      string        `json:"result,omitempty"`
	Tombstoned  bool          `json:"tombstoned,omitempty"`
	log         *OperationLog `json:"-"`
}

// taskSnapshot is the JSON-serializable representation of a Task
type taskSnapshot struct {
	ID          string       `json:"id"`
	RepoID      string       `json:"repo_id"`
	Title       string       `json:"title"`
	Description string       `json:"description"`
	Status      TaskStatus   `json:"status"`
	ClaimedBy   string       `json:"claimed_by"`
	CreatedBy   string       `json:"created_by"`
	CreatedAt   time.Time    `json:"created_at"`
	ClaimedAt   *time.Time   `json:"claimed_at,omitempty"`
	CompletedAt *time.Time   `json:"completed_at,omitempty"`
	Result      string       `json:"result,omitempty"`
	Tombstoned  bool         `json:"tombstoned,omitempty"`
	Log         []*Operation `json:"log,omitempty"`
}

// MarshalJSON serializes a Task including its operation log
func (t *Task) MarshalJSON() ([]byte, error) {
	return json.Marshal(taskSnapshot{
		ID:          t.ID,
		RepoID:      t.RepoID,
		Title:       t.Title,
		Description: t.Description,
		Status:      t.Status,
		ClaimedBy:   t.ClaimedBy,
		CreatedBy:   t.CreatedBy,
		CreatedAt:   t.CreatedAt,
		ClaimedAt:   t.ClaimedAt,
		CompletedAt: t.CompletedAt,
		Result:      t.Result,
		Tombstoned:  t.Tombstoned,
		Log:         t.log.Operations(),
	})
}

// UnmarshalJSON deserializes a Task and rebuilds its operation log
func (t *Task) UnmarshalJSON(data []byte) error {
	var snap taskSnapshot
	if err := json.Unmarshal(data, &snap); err != nil {
		return err
	}
	t.ID = snap.ID
	t.RepoID = snap.RepoID
	t.Title = snap.Title
	t.Description = snap.Description
	t.Status = snap.Status
	t.ClaimedBy = snap.ClaimedBy
	t.CreatedBy = snap.CreatedBy
	t.CreatedAt = snap.CreatedAt
	t.ClaimedAt = snap.ClaimedAt
	t.CompletedAt = snap.CompletedAt
	t.Result = snap.Result
	t.Tombstoned = snap.Tombstoned
	t.log = NewOperationLog()
	for _, op := range snap.Log {
		t.log.ImportOperation(op)
	}
	return nil
}

// Log returns the operation log
func (t *Task) Log() *OperationLog {
	return t.log
}

// SetResult sets the task result
func (t *Task) SetResult(author, result string) {
	t.Result = result
	t.log.Add(&Operation{
		ID:     generateID(),
		Type:   OpSetResult,
		Author: author,
		Data:   map[string]interface{}{"result": result},
	})
}

// Tombstone marks this task as deleted
func (t *Task) Tombstone(author string) {
	t.Tombstoned = true
	t.log.Add(&Operation{
		ID:     generateID(),
		Type:   OpTombstone,
		Author: author,
	})
}

// Merge merges another task's operations into this one
func (t *Task) Merge(other *Task) {
	t.log.clock.Merge(other.log.clock)

	existingIDs := make(map[string]bool)
	for _, op := range t.log.Operations() {
		existingIDs[op.ID] = true
	}
	for _, op := range other.log.Operations() {
		if !existingIDs[op.ID] {
			t.log.ImportOperation(op)
		}
	}

	allOps := make([]*Operation, len(t.log.Operations()))
	copy(allOps, t.log.Operations())
	sort.Slice(allOps, func(a, b int) bool {
		if allOps[a].Lamport != allOps[b].Lamport {
			return allOps[a].Lamport < allOps[b].Lamport
		}
		return allOps[a].Timestamp.Before(allOps[b].Timestamp)
	})

	// Reset and replay
	t.Status = TaskOpen
	t.ClaimedBy = ""
	t.ClaimedAt = nil
	t.CompletedAt = nil
	t.Result = ""
	t.Tombstoned = false
	t.applyOperations(allOps)
}

func (t *Task) applyOperations(ops []*Operation) {
	for _, op := range ops {
		switch op.Type {
		case OpSetTitle:
			if title, ok := op.Data["title"].(string); ok {
				t.Title = title
			}
		case OpSetBody:
			if desc, ok := op.Data["body"].(string); ok {
				t.Description = desc
			}
		case OpSetStatus:
			if status, ok := op.Data["status"].(string); ok {
				t.Status = TaskStatus(status)
			}
		case OpClaimTask:
			if claimedBy, ok := op.Data["claimed_by"].(string); ok {
				t.Status = TaskClaimed
				t.ClaimedBy = claimedBy
				ts := op.Timestamp
				t.ClaimedAt = &ts
			}
		case OpCompleteTask:
			t.Status = TaskCompleted
			ts := op.Timestamp
			t.CompletedAt = &ts
		case OpFailTask:
			t.Status = TaskFailed
			ts := op.Timestamp
			t.CompletedAt = &ts
		case OpSetResult:
			if result, ok := op.Data["result"].(string); ok {
				t.Result = result
			}
		case OpTombstone:
			t.Tombstoned = true
		}
	}
}

// TaskStore manages tasks per repository
type TaskStore struct {
	mu      sync.RWMutex
	dataDir string
	tasks   map[string]map[string]*Task // repoID -> taskID -> Task
}

// NewTaskStore creates a new task store
func NewTaskStore(dataDir string) *TaskStore {
	return &TaskStore{
		dataDir: dataDir,
		tasks:   make(map[string]map[string]*Task),
	}
}

// Load loads tasks from disk
func (s *TaskStore) Load() error {
	if s.dataDir == "" {
		return nil
	}
	path := filepath.Join(s.dataDir, "tasks.json")
	return persistence.LoadJSON(path, &s.tasks)
}

// Save persists tasks to disk
func (s *TaskStore) Save() error {
	if s.dataDir == "" {
		return nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.saveLocked()
}

// saveLocked persists while the caller already holds the write lock.
func (s *TaskStore) saveLocked() error {
	if s.dataDir == "" {
		return nil
	}
	return persistence.SaveJSON(filepath.Join(s.dataDir, "tasks.json"), s.tasks)
}

// List returns a copy of all non-tombstoned tasks for a repository, optionally filtered by status
func (s *TaskStore) List(repoID string, status TaskStatus) []Task {
	s.mu.RLock()
	defer s.mu.RUnlock()

	repo, ok := s.tasks[repoID]
	if !ok {
		return []Task{}
	}

	result := make([]Task, 0, len(repo))
	for _, t := range repo {
		if t.Tombstoned {
			continue
		}
		if status == "" || t.Status == status {
			result = append(result, *t)
		}
	}
	return result
}

// Get returns a specific task
func (s *TaskStore) Get(repoID, taskID string) (*Task, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	repo, ok := s.tasks[repoID]
	if !ok {
		return nil, fmt.Errorf("repo not found: %s", repoID)
	}
	task, ok := repo[taskID]
	if !ok || task.Tombstoned {
		return nil, fmt.Errorf("task not found: %s", taskID)
	}
	cp := *task
	return &cp, nil
}

// Create creates a new task
func (s *TaskStore) Create(repoID, id, createdBy, title, description string) *Task {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.tasks[repoID]; !ok {
		s.tasks[repoID] = make(map[string]*Task)
	}

	now := time.Now()
	task := &Task{
		ID:          id,
		RepoID:      repoID,
		Title:       title,
		Description: description,
		Status:      TaskOpen,
		CreatedBy:   createdBy,
		CreatedAt:   now,
		log:         NewOperationLog(),
	}

	task.log.Add(&Operation{
		ID:        id,
		Type:      OpCreate,
		Author:    createdBy,
		Timestamp: now,
		Data: map[string]interface{}{
			"title":       title,
			"description": description,
		},
	})

	s.tasks[repoID][id] = task
	cp := *task
	return &cp
}

// Claim claims a task
func (s *TaskStore) Claim(repoID, taskID, claimedBy string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	repo, ok := s.tasks[repoID]
	if !ok {
		return fmt.Errorf("repo not found: %s", repoID)
	}
	task, ok := repo[taskID]
	if !ok || task.Tombstoned {
		return fmt.Errorf("task not found: %s", taskID)
	}
	if task.Status != TaskOpen {
		return fmt.Errorf("task is not open: %s", taskID)
	}

	now := time.Now()
	task.Status = TaskClaimed
	task.ClaimedBy = claimedBy
	task.ClaimedAt = &now
	task.log.Add(&Operation{
		ID:        generateID(),
		Type:      OpClaimTask,
		Author:    claimedBy,
		Timestamp: now,
		Data:      map[string]interface{}{"claimed_by": claimedBy},
	})
	return s.saveLocked()
}

// Complete completes a task
func (s *TaskStore) Complete(repoID, taskID, result string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	repo, ok := s.tasks[repoID]
	if !ok {
		return fmt.Errorf("repo not found: %s", repoID)
	}
	task, ok := repo[taskID]
	if !ok || task.Tombstoned {
		return fmt.Errorf("task not found: %s", taskID)
	}
	if task.Status != TaskClaimed {
		return fmt.Errorf("task is not claimed: %s", taskID)
	}

	now := time.Now()
	task.Status = TaskCompleted
	task.CompletedAt = &now
	task.Result = result
	task.log.Add(&Operation{
		ID:        generateID(),
		Type:      OpCompleteTask,
		Author:    task.ClaimedBy,
		Timestamp: now,
	})
	if result != "" {
		task.SetResult(task.ClaimedBy, result)
	}
	return s.saveLocked()
}

// All returns deep copies of all tasks across all repositories
func (s *TaskStore) All() map[string][]Task {
	s.mu.RLock()
	defer s.mu.RUnlock()
	data := make(map[string][]Task, len(s.tasks))
	for repoID, repoTasks := range s.tasks {
		tasks := make([]Task, 0, len(repoTasks))
		for _, t := range repoTasks {
			tasks = append(tasks, *t)
		}
		data[repoID] = tasks
	}
	return data
}

// MergeRemote merges a remote task snapshot into the local store
func (s *TaskStore) MergeRemote(repoID string, remote *Task) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.tasks[repoID]; !ok {
		s.tasks[repoID] = make(map[string]*Task)
	}

	if local, ok := s.tasks[repoID][remote.ID]; ok {
		local.Merge(remote)
	} else {
		cp := *remote
		s.tasks[repoID][remote.ID] = &cp
	}
	return s.saveLocked()
}
