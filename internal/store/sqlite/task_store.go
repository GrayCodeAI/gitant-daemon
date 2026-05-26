package sqlite

import (
	"context"
	"fmt"
	"time"

	"github.com/lakshmanpatel/gitant/internal/store"
)

// TaskStore implements store.TaskStore for SQLite
type TaskStore struct {
	db *DB
}

// NewTaskStore creates a new SQLite task store
func NewTaskStore(db *DB) *TaskStore {
	return &TaskStore{db: db}
}

// Create creates a new task
func (s *TaskStore) Create(ctx context.Context, repoID, id, createdBy, title, description string) (*store.Task, error) {
	now := time.Now()
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO tasks (id, repo_id, title, description, status, created_by, created_at)
		 VALUES (?, ?, ?, ?, 'open', ?, ?)`,
		id, repoID, title, description, createdBy, now,
	)
	if err != nil {
		return nil, fmt.Errorf("creating task: %w", err)
	}

	return &store.Task{
		ID:          id,
		RepoID:      repoID,
		Title:       title,
		Description: description,
		Status:      "open",
		CreatedBy:   createdBy,
		CreatedAt:   now,
	}, nil
}

// List lists tasks for a repository
func (s *TaskStore) List(ctx context.Context, repoID string, status string) ([]*store.Task, error) {
	query := `SELECT id, repo_id, title, description, status, claimed_by, created_by,
			  created_at, claimed_at, completed_at, result
		      FROM tasks WHERE repo_id = ?`
	args := []interface{}{repoID}

	if status != "" {
		query += " AND status = ?"
		args = append(args, status)
	}
	query += " ORDER BY created_at DESC"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("listing tasks: %w", err)
	}
	defer rows.Close()

	var tasks []*store.Task
	for rows.Next() {
		t := &store.Task{}
		if err := rows.Scan(&t.ID, &t.RepoID, &t.Title, &t.Description,
			&t.Status, &t.ClaimedBy, &t.CreatedBy,
			&t.CreatedAt, &t.ClaimedAt, &t.CompletedAt, &t.Result); err != nil {
			return nil, fmt.Errorf("scanning task: %w", err)
		}
		tasks = append(tasks, t)
	}
	if tasks == nil {
		tasks = []*store.Task{}
	}
	return tasks, nil
}

// Claim claims a task
func (s *TaskStore) Claim(ctx context.Context, repoID, taskID, claimedBy string) error {
	now := time.Now()
	result, err := s.db.ExecContext(ctx,
		`UPDATE tasks SET status = 'claimed', claimed_by = ?, claimed_at = ?
		 WHERE id = ? AND repo_id = ? AND status = 'open'`,
		claimedBy, now, taskID, repoID)
	if err != nil {
		return fmt.Errorf("claiming task: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("task not found or not open: %s", taskID)
	}
	return nil
}

// Complete completes a task
func (s *TaskStore) Complete(ctx context.Context, repoID, taskID, result string) error {
	now := time.Now()
	res, err := s.db.ExecContext(ctx,
		`UPDATE tasks SET status = 'completed', completed_at = ?, result = ?
		 WHERE id = ? AND repo_id = ? AND status = 'claimed'`,
		now, result, taskID, repoID)
	if err != nil {
		return fmt.Errorf("completing task: %w", err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("task not found or not claimed: %s", taskID)
	}
	return nil
}

// Save is a no-op for SQLite
func (s *TaskStore) Save() error {
	return nil
}
