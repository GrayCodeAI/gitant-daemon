package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/lakshmanpatel/gitant/internal/store"
)

// PullRequestStore implements store.PullRequestStore for SQLite
type PullRequestStore struct {
	db *DB
}

// NewPullRequestStore creates a new SQLite PR store
func NewPullRequestStore(db *DB) *PullRequestStore {
	return &PullRequestStore{db: db}
}

// Create creates a new pull request
func (s *PullRequestStore) Create(ctx context.Context, repoID, id, author, title, body, sourceBranch, targetBranch string) (*store.PullRequest, error) {
	now := time.Now()
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO pull_requests (id, repo_id, title, body, status, author, source_branch, target_branch, created_at, updated_at)
		 VALUES (?, ?, ?, ?, 'open', ?, ?, ?, ?, ?)`,
		id, repoID, title, body, author, sourceBranch, targetBranch, now, now,
	)
	if err != nil {
		return nil, fmt.Errorf("creating PR: %w", err)
	}

	return &store.PullRequest{
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
	}, nil
}

// Get gets a PR by ID
func (s *PullRequestStore) Get(ctx context.Context, repoID, prID string) (*store.PullRequest, error) {
	pr := &store.PullRequest{}
	err := s.db.QueryRowContext(ctx,
		`SELECT id, title, body, status, author, source_branch, target_branch, assignee, created_at, updated_at
		 FROM pull_requests WHERE id = ? AND repo_id = ?`, prID, repoID,
	).Scan(&pr.ID, &pr.Title, &pr.Body, &pr.Status, &pr.Author,
		&pr.SourceBranch, &pr.TargetBranch, &pr.Assignee,
		&pr.CreatedAt, &pr.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("PR not found: %s", prID)
		}
		return nil, fmt.Errorf("getting PR: %w", err)
	}

	labels, _ := s.getLabels(ctx, prID)
	pr.Labels = labels
	reviewers, _ := s.getReviewers(ctx, prID)
	pr.Reviewers = reviewers

	return pr, nil
}

// List lists PRs in a repository
func (s *PullRequestStore) List(ctx context.Context, repoID string, filters store.PRFilters) ([]*store.PullRequest, error) {
	query := `SELECT id, title, body, status, author, source_branch, target_branch, assignee, created_at, updated_at
		      FROM pull_requests WHERE repo_id = ?`
	args := []interface{}{repoID}

	if filters.Status != "" {
		query += " AND status = ?"
		args = append(args, filters.Status)
	}
	query += " ORDER BY created_at DESC"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("listing PRs: %w", err)
	}
	defer rows.Close()

	var prs []*store.PullRequest
	for rows.Next() {
		pr := &store.PullRequest{}
		if err := rows.Scan(&pr.ID, &pr.Title, &pr.Body, &pr.Status, &pr.Author,
			&pr.SourceBranch, &pr.TargetBranch, &pr.Assignee,
			&pr.CreatedAt, &pr.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scanning PR: %w", err)
		}
		labels, _ := s.getLabels(ctx, pr.ID)
		pr.Labels = labels
		reviewers, _ := s.getReviewers(ctx, pr.ID)
		pr.Reviewers = reviewers
		prs = append(prs, pr)
	}

	if prs == nil {
		prs = []*store.PullRequest{}
	}

	return prs, nil
}

// Update updates a PR
func (s *PullRequestStore) Update(ctx context.Context, repoID, prID string, fn func(*store.PullRequest) error) error {
	pr, err := s.Get(ctx, repoID, prID)
	if err != nil {
		return err
	}

	if err := fn(pr); err != nil {
		return err
	}

	pr.UpdatedAt = time.Now()

	_, err = s.db.ExecContext(ctx,
		`UPDATE pull_requests SET title = ?, body = ?, status = ?, assignee = ?, updated_at = ?
		 WHERE id = ? AND repo_id = ?`,
		pr.Title, pr.Body, pr.Status, pr.Assignee,
		pr.UpdatedAt, prID, repoID,
	)
	if err != nil {
		return fmt.Errorf("updating PR: %w", err)
	}

	s.setLabels(ctx, prID, pr.Labels)
	s.setReviewers(ctx, prID, pr.Reviewers)

	return nil
}

// Delete deletes a PR
func (s *PullRequestStore) Delete(ctx context.Context, repoID, prID string) error {
	result, err := s.db.ExecContext(ctx,
		"DELETE FROM pull_requests WHERE id = ? AND repo_id = ?", prID, repoID)
	if err != nil {
		return fmt.Errorf("deleting PR: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("PR not found: %s", prID)
	}
	return nil
}

// Save is a no-op for SQLite
func (s *PullRequestStore) Save() error {
	return nil
}

func (s *PullRequestStore) getLabels(ctx context.Context, prID string) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT label FROM pr_labels WHERE pr_id = ?", prID)
	if err != nil {
		return []string{}, nil
	}
	defer rows.Close()
	var labels []string
	for rows.Next() {
		var l string
		rows.Scan(&l)
		labels = append(labels, l)
	}
	if labels == nil {
		return []string{}, nil
	}
	return labels, nil
}

func (s *PullRequestStore) setLabels(ctx context.Context, prID string, labels []string) {
	s.db.ExecContext(ctx, "DELETE FROM pr_labels WHERE pr_id = ?", prID)
	for _, l := range labels {
		s.db.ExecContext(ctx, "INSERT INTO pr_labels (pr_id, label) VALUES (?, ?)", prID, l)
	}
}

func (s *PullRequestStore) getReviewers(ctx context.Context, prID string) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT reviewer FROM pr_reviewers WHERE pr_id = ?", prID)
	if err != nil {
		return []string{}, nil
	}
	defer rows.Close()
	var reviewers []string
	for rows.Next() {
		var r string
		rows.Scan(&r)
		reviewers = append(reviewers, r)
	}
	if reviewers == nil {
		return []string{}, nil
	}
	return reviewers, nil
}

func (s *PullRequestStore) setReviewers(ctx context.Context, prID string, reviewers []string) {
	s.db.ExecContext(ctx, "DELETE FROM pr_reviewers WHERE pr_id = ?", prID)
	for _, r := range reviewers {
		s.db.ExecContext(ctx, "INSERT INTO pr_reviewers (pr_id, reviewer) VALUES (?, ?)", prID, r)
	}
}
