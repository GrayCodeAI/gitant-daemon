package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/lakshmanpatel/gitant/internal/store"
)

// IssueStore implements store.IssueStore for SQLite
type IssueStore struct {
	db *DB
}

// NewIssueStore creates a new SQLite issue store
func NewIssueStore(db *DB) *IssueStore {
	return &IssueStore{db: db}
}

// Create creates a new issue
func (s *IssueStore) Create(ctx context.Context, repoID, id, author, title, body string) (*store.Issue, error) {
	now := time.Now()
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO issues (id, repo_id, title, body, status, author, created_at, updated_at)
		 VALUES (?, ?, ?, ?, 'open', ?, ?, ?)`,
		id, repoID, title, body, author, now, now,
	)
	if err != nil {
		return nil, fmt.Errorf("creating issue: %w", err)
	}

	// Record the create operation
	opData, _ := json.Marshal(map[string]interface{}{"title": title, "body": body})
	_, err = s.db.ExecContext(ctx,
		`INSERT INTO issue_operations (id, issue_id, op_type, author, timestamp, lamport, data)
		 VALUES (?, ?, 'create', ?, ?, 0, ?)`,
		fmt.Sprintf("op-%d", now.UnixNano()), id, author, now, string(opData),
	)
	if err != nil {
		return nil, fmt.Errorf("recording create operation: %w", err)
	}

	return &store.Issue{
		ID:        id,
		Title:     title,
		Body:      body,
		Status:    "open",
		Author:    author,
		Labels:    []string{},
		Assignee:  "",
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

// Get gets an issue by ID
func (s *IssueStore) Get(ctx context.Context, repoID, issueID string) (*store.Issue, error) {
	issue := &store.Issue{}
	err := s.db.QueryRowContext(ctx,
		`SELECT id, title, body, status, author, assignee, created_at, updated_at
		 FROM issues WHERE id = ? AND repo_id = ?`, issueID, repoID,
	).Scan(&issue.ID, &issue.Title, &issue.Body, &issue.Status,
		&issue.Author, &issue.Assignee, &issue.CreatedAt, &issue.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("issue not found: %s", issueID)
		}
		return nil, fmt.Errorf("getting issue: %w", err)
	}

	// Load labels
	labels, err := s.getLabels(ctx, issueID)
	if err != nil {
		return nil, err
	}
	issue.Labels = labels

	return issue, nil
}

// List lists issues in a repository
func (s *IssueStore) List(ctx context.Context, repoID string, filters store.IssueFilters) ([]*store.Issue, error) {
	query := `SELECT id, title, body, status, author, assignee, created_at, updated_at
		      FROM issues WHERE repo_id = ?`
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
		issue := &store.Issue{}
		if err := rows.Scan(&issue.ID, &issue.Title, &issue.Body, &issue.Status,
			&issue.Author, &issue.Assignee, &issue.CreatedAt, &issue.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scanning issue: %w", err)
		}

		labels, err := s.getLabels(ctx, issue.ID)
		if err != nil {
			return nil, err
		}
		issue.Labels = labels

		// Filter by labels if specified
		if len(filters.Labels) > 0 {
			if !hasAllLabels(issue.Labels, filters.Labels) {
				continue
			}
		}

		issues = append(issues, issue)
	}

	if issues == nil {
		issues = []*store.Issue{}
	}

	return issues, nil
}

// Update updates an issue
func (s *IssueStore) Update(ctx context.Context, repoID, issueID string, fn func(*store.Issue) error) error {
	issue, err := s.Get(ctx, repoID, issueID)
	if err != nil {
		return err
	}

	if err := fn(issue); err != nil {
		return err
	}

	issue.UpdatedAt = time.Now()

	_, err = s.db.ExecContext(ctx,
		`UPDATE issues SET title = ?, body = ?, status = ?, assignee = ?, updated_at = ?
		 WHERE id = ? AND repo_id = ?`,
		issue.Title, issue.Body, issue.Status, issue.Assignee,
		issue.UpdatedAt, issueID, repoID,
	)
	if err != nil {
		return fmt.Errorf("updating issue: %w", err)
	}

	// Update labels
	if err := s.setLabels(ctx, issueID, issue.Labels); err != nil {
		return err
	}

	return nil
}

// Delete deletes an issue
func (s *IssueStore) Delete(ctx context.Context, repoID, issueID string) error {
	result, err := s.db.ExecContext(ctx,
		"DELETE FROM issues WHERE id = ? AND repo_id = ?", issueID, repoID)
	if err != nil {
		return fmt.Errorf("deleting issue: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("issue not found: %s", issueID)
	}

	return nil
}

// Save is a no-op for SQLite (data is persisted immediately)
func (s *IssueStore) Save() error {
	return nil
}

// getLabels gets labels for an issue
func (s *IssueStore) getLabels(ctx context.Context, issueID string) ([]string, error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT label FROM issue_labels WHERE issue_id = ?", issueID)
	if err != nil {
		return nil, fmt.Errorf("getting labels: %w", err)
	}
	defer rows.Close()

	labels := []string{}
	for rows.Next() {
		var label string
		if err := rows.Scan(&label); err != nil {
			return nil, fmt.Errorf("scanning label: %w", err)
		}
		labels = append(labels, label)
	}
	return labels, nil
}

// setLabels sets labels for an issue
func (s *IssueStore) setLabels(ctx context.Context, issueID string, labels []string) error {
	_, err := s.db.ExecContext(ctx,
		"DELETE FROM issue_labels WHERE issue_id = ?", issueID)
	if err != nil {
		return fmt.Errorf("deleting labels: %w", err)
	}

	for _, label := range labels {
		_, err := s.db.ExecContext(ctx,
			"INSERT INTO issue_labels (issue_id, label) VALUES (?, ?)",
			issueID, label)
		if err != nil {
			return fmt.Errorf("inserting label: %w", err)
		}
	}

	return nil
}

// hasAllLabels checks if issue has all required labels
func hasAllLabels(issueLabels, requiredLabels []string) bool {
	labelSet := make(map[string]bool)
	for _, l := range issueLabels {
		labelSet[l] = true
	}
	for _, l := range requiredLabels {
		if !labelSet[l] {
			return false
		}
	}
	return true
}
