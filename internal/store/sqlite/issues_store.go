package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/lakshmanpatel/gitant/internal/store"
)

// SQLIssueStore implements the store.IssueStore interface using SQLite
type SQLIssueStore struct {
	db *sql.DB
}

// Create creates a new issue
func (s *SQLIssueStore) Create(ctx context.Context, repoID, id, author, title, body string) (*store.Issue, error) {
	labelsJSON := "[]"

	query := `
		INSERT INTO issues (id, repo_id, title, body, status, author, labels, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	now := time.Now()
	_, err := s.db.ExecContext(ctx, query,
		id, repoID, title, body, "open", author, labelsJSON, now, now,
	)

	if err != nil {
		return nil, fmt.Errorf("creating issue: %w", err)
	}

	issue := &store.Issue{
		ID:        id,
		Title:     title,
		Body:      body,
		Status:    "open",
		Author:    author,
		Labels:    []string{},
		CreatedAt: now,
		UpdatedAt: now,
	}

	return issue, nil
}

// Get retrieves an issue by ID
func (s *SQLIssueStore) Get(ctx context.Context, repoID, issueID string) (*store.Issue, error) {
	query := `
		SELECT id, title, body, status, author, labels, created_at, updated_at
		FROM issues
		WHERE id = ? AND repo_id = ? AND tombstoned = 0
	`

	var labelsJSON string
	issue := &store.Issue{}

	err := s.db.QueryRowContext(ctx, query, issueID, repoID).Scan(
		&issue.ID,
		&issue.Title,
		&issue.Body,
		&issue.Status,
		&issue.Author,
		&labelsJSON,
		&issue.CreatedAt,
		&issue.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("issue not found: %s", issueID)
		}
		return nil, fmt.Errorf("getting issue: %w", err)
	}

	// Parse labels JSON
	if labelsJSON != "" {
		if err := json.Unmarshal([]byte(labelsJSON), &issue.Labels); err != nil {
			return nil, fmt.Errorf("parsing labels: %w", err)
		}
	}

	return issue, nil
}

// List lists issues with filters
func (s *SQLIssueStore) List(ctx context.Context, repoID string, filters store.IssueFilters) ([]*store.Issue, error) {
	query := `
		SELECT id, title, body, status, author, labels, created_at, updated_at
		FROM issues
		WHERE repo_id = ? AND tombstoned = 0
	`

	args := []interface{}{repoID}

	if filters.Status != "" {
		query += " AND status = ?"
		args = append(args, filters.Status)
	}

	query += " ORDER BY created_at DESC"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("listing issues: %w", err)
	}
	defer rows.Close()

	var issues []*store.Issue
	for rows.Next() {
		var labelsJSON string
		issue := &store.Issue{}
		err := rows.Scan(
			&issue.ID,
			&issue.Title,
			&issue.Body,
			&issue.Status,
			&issue.Author,
			&labelsJSON,
			&issue.CreatedAt,
			&issue.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scanning issue: %w", err)
		}

		if labelsJSON != "" {
			if err := json.Unmarshal([]byte(labelsJSON), &issue.Labels); err != nil {
				return nil, fmt.Errorf("parsing labels: %w", err)
			}
		}

		issues = append(issues, issue)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating issues: %w", err)
	}

	return issues, nil
}

// Update updates an issue using a function to modify the issue
func (s *SQLIssueStore) Update(ctx context.Context, repoID, issueID string, fn func(*store.Issue) error) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback()

	// Get current issue
	issue, err := s.Get(ctx, repoID, issueID)
	if err != nil {
		return err
	}

	// Apply the update function
	if err := fn(issue); err != nil {
		return fmt.Errorf("applying update: %w", err)
	}

	// Serialize labels to JSON
	labelsJSON, err := json.Marshal(issue.Labels)
	if err != nil {
		return fmt.Errorf("serializing labels: %w", err)
	}

	query := `
		UPDATE issues
		SET title = ?, body = ?, status = ?, labels = ?, assignee = ?, updated_at = ?
		WHERE id = ? AND repo_id = ?
	`

	_, err = tx.ExecContext(ctx, query,
		issue.Title, issue.Body, issue.Status, string(labelsJSON), issue.Assignee, time.Now(),
		issueID, repoID,
	)

	if err != nil {
		return fmt.Errorf("updating issue: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("committing transaction: %w", err)
	}

	return nil
}

// Delete soft-deletes an issue
func (s *SQLIssueStore) Delete(ctx context.Context, repoID, issueID string) error {
	query := `UPDATE issues SET tombstoned = 1, updated_at = ? WHERE id = ? AND repo_id = ?`

	_, err := s.db.ExecContext(ctx, query, time.Now(), issueID, repoID)
	if err != nil {
		return fmt.Errorf("deleting issue: %w", err)
	}

	return nil
}

// Save is a no-op for SQL stores (data is persisted on every operation)
func (s *SQLIssueStore) Save() error {
	return nil
}
