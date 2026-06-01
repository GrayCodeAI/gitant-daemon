package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/lakshmanpatel/gitant/internal/store"
)

// SQLPullRequestStore implements the store.PullRequestStore interface using SQLite
type SQLPullRequestStore struct {
	db *sql.DB
}

// Create creates a new pull request
func (s *SQLPullRequestStore) Create(ctx context.Context, repoID, id, author, title, body, sourceBranch, targetBranch string) (*store.PullRequest, error) {
	labelsJSON := "[]"
	reviewersJSON := "[]"

	query := `
		INSERT INTO pull_requests (id, repo_id, title, body, status, author, source_branch, target_branch, labels, reviewers, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	now := time.Now()
	_, err := s.db.ExecContext(ctx, query,
		id, repoID, title, body, "open", author, sourceBranch, targetBranch, labelsJSON, reviewersJSON, now, now,
	)

	if err != nil {
		return nil, fmt.Errorf("creating pull request: %w", err)
	}

	pr := &store.PullRequest{
		ID:           id,
		Title:        title,
		Body:         body,
		Status:       "open",
		Author:       author,
		SourceBranch: sourceBranch,
		TargetBranch: targetBranch,
		Labels:       []string{},
		Reviewers:    []string{},
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	return pr, nil
}

// Get retrieves a pull request by ID
func (s *SQLPullRequestStore) Get(ctx context.Context, repoID, prID string) (*store.PullRequest, error) {
	query := `
		SELECT id, title, body, status, author, source_branch, target_branch, labels, reviewers, created_at, updated_at
		FROM pull_requests
		WHERE id = ? AND repo_id = ? AND tombstoned = 0
	`

	var labelsJSON, reviewersJSON string
	pr := &store.PullRequest{}

	err := s.db.QueryRowContext(ctx, query, prID, repoID).Scan(
		&pr.ID,
		&pr.Title,
		&pr.Body,
		&pr.Status,
		&pr.Author,
		&pr.SourceBranch,
		&pr.TargetBranch,
		&labelsJSON,
		&reviewersJSON,
		&pr.CreatedAt,
		&pr.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("pull request not found: %s", prID)
		}
		return nil, fmt.Errorf("getting pull request: %w", err)
	}

	// Parse labels and reviewers JSON
	if labelsJSON != "" {
		if err := json.Unmarshal([]byte(labelsJSON), &pr.Labels); err != nil {
			return nil, fmt.Errorf("parsing labels: %w", err)
		}
	}
	if reviewersJSON != "" {
		if err := json.Unmarshal([]byte(reviewersJSON), &pr.Reviewers); err != nil {
			return nil, fmt.Errorf("parsing reviewers: %w", err)
		}
	}

	return pr, nil
}

// List lists pull requests with filters
func (s *SQLPullRequestStore) List(ctx context.Context, repoID string, filters store.PRFilters) ([]*store.PullRequest, error) {
	query := `
		SELECT id, title, body, status, author, source_branch, target_branch, labels, reviewers, created_at, updated_at
		FROM pull_requests
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
		return nil, fmt.Errorf("listing pull requests: %w", err)
	}
	defer rows.Close()

	var pullRequests []*store.PullRequest
	for rows.Next() {
		var labelsJSON, reviewersJSON string
		pr := &store.PullRequest{}
		err := rows.Scan(
			&pr.ID,
			&pr.Title,
			&pr.Body,
			&pr.Status,
			&pr.Author,
			&pr.SourceBranch,
			&pr.TargetBranch,
			&labelsJSON,
			&reviewersJSON,
			&pr.CreatedAt,
			&pr.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scanning pull request: %w", err)
		}

		if labelsJSON != "" {
			if err := json.Unmarshal([]byte(labelsJSON), &pr.Labels); err != nil {
				return nil, fmt.Errorf("parsing labels: %w", err)
			}
		}
		if reviewersJSON != "" {
			if err := json.Unmarshal([]byte(reviewersJSON), &pr.Reviewers); err != nil {
				return nil, fmt.Errorf("parsing reviewers: %w", err)
			}
		}

		pullRequests = append(pullRequests, pr)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating pull requests: %w", err)
	}

	return pullRequests, nil
}

// Update updates a pull request using a function to modify the PR
func (s *SQLPullRequestStore) Update(ctx context.Context, repoID, prID string, fn func(*store.PullRequest) error) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback()

	// Get current PR
	pr, err := s.Get(ctx, repoID, prID)
	if err != nil {
		return err
	}

	// Apply the update function
	if err := fn(pr); err != nil {
		return fmt.Errorf("applying update: %w", err)
	}

	// Serialize labels and reviewers to JSON
	labelsJSON, err := json.Marshal(pr.Labels)
	if err != nil {
		return fmt.Errorf("serializing labels: %w", err)
	}
	reviewersJSON, err := json.Marshal(pr.Reviewers)
	if err != nil {
		return fmt.Errorf("serializing reviewers: %w", err)
	}

	query := `
		UPDATE pull_requests
		SET title = ?, body = ?, status = ?, labels = ?, reviewers = ?, assignee = ?, updated_at = ?
		WHERE id = ? AND repo_id = ?
	`

	_, err = tx.ExecContext(ctx, query,
		pr.Title, pr.Body, pr.Status, string(labelsJSON), string(reviewersJSON), pr.Assignee, time.Now(),
		prID, repoID,
	)

	if err != nil {
		return fmt.Errorf("updating pull request: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("committing transaction: %w", err)
	}

	return nil
}

// Delete soft-deletes a pull request
func (s *SQLPullRequestStore) Delete(ctx context.Context, repoID, prID string) error {
	query := `UPDATE pull_requests SET tombstoned = 1, updated_at = ? WHERE id = ? AND repo_id = ?`

	_, err := s.db.ExecContext(ctx, query, time.Now(), prID, repoID)
	if err != nil {
		return fmt.Errorf("deleting pull request: %w", err)
	}

	return nil
}

// Save is a no-op for SQL stores (data is persisted on every operation)
func (s *SQLPullRequestStore) Save() error {
	return nil
}
