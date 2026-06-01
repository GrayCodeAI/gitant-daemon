package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/lakshmanpatel/gitant/internal/store"
)

// SQLTaskStore implements the store.TaskStore interface using SQLite
type SQLTaskStore struct {
	db *sql.DB
}

// Create creates a new task
func (s *SQLTaskStore) Create(ctx context.Context, repoID, id, createdBy, title, description string) (*store.Task, error) {
	query := `
		INSERT INTO tasks (id, repo_id, title, description, status, created_by, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`

	now := time.Now()
	_, err := s.db.ExecContext(ctx, query,
		id, repoID, title, description, "open", createdBy, now,
	)

	if err != nil {
		return nil, fmt.Errorf("creating task: %w", err)
	}

	task := &store.Task{
		ID:          id,
		RepoID:      repoID,
		Title:       title,
		Description: description,
		Status:      "open",
		CreatedBy:   createdBy,
		CreatedAt:   now,
	}

	return task, nil
}

// List lists tasks for a repository, optionally filtered by status
func (s *SQLTaskStore) List(ctx context.Context, repoID, status string) ([]*store.Task, error) {
	query := `
		SELECT id, repo_id, title, description, status, claimed_by, created_by, created_at, claimed_at, completed_at, result
		FROM tasks
		WHERE repo_id = ?
	`

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
		task := &store.Task{}
		err := rows.Scan(
			&task.ID,
			&task.RepoID,
			&task.Title,
			&task.Description,
			&task.Status,
			&task.ClaimedBy,
			&task.CreatedBy,
			&task.CreatedAt,
			&task.ClaimedAt,
			&task.CompletedAt,
			&task.Result,
		)
		if err != nil {
			return nil, fmt.Errorf("scanning task: %w", err)
		}
		tasks = append(tasks, task)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating tasks: %w", err)
	}

	return tasks, nil
}

// Claim claims a task for an agent
func (s *SQLTaskStore) Claim(ctx context.Context, repoID, taskID, claimedBy string) error {
	query := `
		UPDATE tasks
		SET status = 'claimed', claimed_by = ?, claimed_at = ?
		WHERE id = ? AND repo_id = ? AND status = 'open'
	`

	_, err := s.db.ExecContext(ctx, query, claimedBy, time.Now(), taskID, repoID)
	if err != nil {
		return fmt.Errorf("claiming task: %w", err)
	}

	return nil
}

// Complete completes a task
func (s *SQLTaskStore) Complete(ctx context.Context, repoID, taskID, result string) error {
	query := `
		UPDATE tasks
		SET status = 'completed', result = ?, completed_at = ?
		WHERE id = ? AND repo_id = ?
	`

	_, err := s.db.ExecContext(ctx, query, result, time.Now(), taskID, repoID)
	if err != nil {
		return fmt.Errorf("completing task: %w", err)
	}

	return nil
}

// Save is a no-op for SQL stores (data is persisted on every operation)
func (s *SQLTaskStore) Save() error {
	return nil
}
