package crdt

import (
	"fmt"
	"path/filepath"
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

// Task represents an agent task
type Task struct {
	ID          string     `json:"id"`
	RepoID      string     `json:"repo_id"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Status      TaskStatus `json:"status"`
	ClaimedBy   string     `json:"claimed_by"`
	CreatedBy   string     `json:"created_by"`
	CreatedAt   time.Time  `json:"created_at"`
	ClaimedAt   *time.Time `json:"claimed_at,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	Result      string     `json:"result,omitempty"`
}

// TaskStore manages tasks per repository
type TaskStore struct {
	mu      sync.RWMutex
	dataDir string
	tasks   map[string][]Task // repoID -> tasks
}

// NewTaskStore creates a new task store
func NewTaskStore(dataDir string) *TaskStore {
	return &TaskStore{
		dataDir: dataDir,
		tasks:   make(map[string][]Task),
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
	return persistence.SaveJSON(filepath.Join(s.dataDir, "tasks.json"), s.tasks)
}

// List returns a copy of all tasks for a repository, optionally filtered by status
func (s *TaskStore) List(repoID string, status TaskStatus) []Task {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tasks := s.tasks[repoID]
	if status == "" {
		result := make([]Task, len(tasks))
		copy(result, tasks)
		return result
	}

	filtered := make([]Task, 0)
	for _, t := range tasks {
		if t.Status == status {
			filtered = append(filtered, t)
		}
	}
	return filtered
}

// Create creates a new task
func (s *TaskStore) Create(repoID, id, createdBy, title, description string) *Task {
	s.mu.Lock()
	defer s.mu.Unlock()

	task := Task{
		ID:          id,
		RepoID:      repoID,
		Title:       title,
		Description: description,
		Status:      TaskOpen,
		CreatedBy:   createdBy,
		CreatedAt:   time.Now(),
	}

	s.tasks[repoID] = append(s.tasks[repoID], task)
	return &s.tasks[repoID][len(s.tasks[repoID])-1]
}

// Claim claims a task
func (s *TaskStore) Claim(repoID, taskID, claimedBy string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, t := range s.tasks[repoID] {
		if t.ID == taskID {
			if t.Status != TaskOpen {
				return fmt.Errorf("task is not open: %s", taskID)
			}
			now := time.Now()
			s.tasks[repoID][i].Status = TaskClaimed
			s.tasks[repoID][i].ClaimedBy = claimedBy
			s.tasks[repoID][i].ClaimedAt = &now
			return s.saveLocked()
		}
	}

	return fmt.Errorf("task not found: %s", taskID)
}

// saveLocked persists while the caller already holds the write lock.
func (s *TaskStore) saveLocked() error {
	if s.dataDir == "" {
		return nil
	}
	path := filepath.Join(s.dataDir, "tasks.json")
	return persistence.SaveJSON(path, s.tasks)
}

// All returns deep copies of all tasks across all repositories
func (s *TaskStore) All() map[string][]Task {
	s.mu.RLock()
	defer s.mu.RUnlock()
	data := make(map[string][]Task, len(s.tasks))
	for repoID, tasks := range s.tasks {
		tasksCopy := make([]Task, len(tasks))
		copy(tasksCopy, tasks)
		data[repoID] = tasksCopy
	}
	return data
}

// Complete completes a task
func (s *TaskStore) Complete(repoID, taskID, result string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, t := range s.tasks[repoID] {
		if t.ID == taskID {
			if t.Status != TaskClaimed {
				return fmt.Errorf("task is not claimed: %s", taskID)
			}
			now := time.Now()
			s.tasks[repoID][i].Status = TaskCompleted
			s.tasks[repoID][i].CompletedAt = &now
			s.tasks[repoID][i].Result = result
			return s.saveLocked()
		}
	}

	return fmt.Errorf("task not found: %s", taskID)
}
