package sqlite

import (
	"context"
	"fmt"
	"time"

	"github.com/lakshmanpatel/gitant/internal/store"
)

// ReviewCommentStore implements store.ReviewCommentStore for SQLite
type ReviewCommentStore struct {
	db *DB
}

// NewReviewCommentStore creates a new SQLite review comment store
func NewReviewCommentStore(db *DB) *ReviewCommentStore {
	return &ReviewCommentStore{db: db}
}

// Create creates a new review comment
func (s *ReviewCommentStore) Create(ctx context.Context, comment *store.ReviewComment) error {
	now := time.Now()
	comment.CreatedAt = now
	comment.UpdatedAt = now
	if comment.Status == "" {
		comment.Status = "open"
	}

	_, err := s.db.ExecContext(ctx,
		`INSERT INTO review_comments (id, pr_id, file_path, line_number, author_id, body, parent_id, status, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		comment.ID, comment.PRID, comment.FilePath, comment.LineNumber,
		comment.AuthorID, comment.Body, comment.ParentID,
		comment.Status, comment.CreatedAt, comment.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("creating review comment: %w", err)
	}
	return nil
}

// Get gets a review comment by ID
func (s *ReviewCommentStore) Get(ctx context.Context, id string) (*store.ReviewComment, error) {
	c := &store.ReviewComment{}
	err := s.db.QueryRowContext(ctx,
		`SELECT id, pr_id, file_path, line_number, author_id, body, parent_id, status, created_at, updated_at
		 FROM review_comments WHERE id = ?`, id,
	).Scan(&c.ID, &c.PRID, &c.FilePath, &c.LineNumber,
		&c.AuthorID, &c.Body, &c.ParentID,
		&c.Status, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("getting review comment: %w", err)
	}
	return c, nil
}

// ListByPR lists all review comments for a PR
func (s *ReviewCommentStore) ListByPR(ctx context.Context, prID string) ([]*store.ReviewComment, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, pr_id, file_path, line_number, author_id, body, parent_id, status, created_at, updated_at
		 FROM review_comments WHERE pr_id = ? ORDER BY created_at`, prID)
	if err != nil {
		return nil, fmt.Errorf("listing review comments: %w", err)
	}
	defer rows.Close()

	var comments []*store.ReviewComment
	for rows.Next() {
		c := &store.ReviewComment{}
		if err := rows.Scan(&c.ID, &c.PRID, &c.FilePath, &c.LineNumber,
			&c.AuthorID, &c.Body, &c.ParentID,
			&c.Status, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scanning review comment: %w", err)
		}
		comments = append(comments, c)
	}
	if comments == nil {
		comments = []*store.ReviewComment{}
	}
	return comments, nil
}

// Update updates a review comment
func (s *ReviewCommentStore) Update(ctx context.Context, comment *store.ReviewComment) error {
	comment.UpdatedAt = time.Now()
	_, err := s.db.ExecContext(ctx,
		`UPDATE review_comments SET body = ?, updated_at = ? WHERE id = ?`,
		comment.Body, comment.UpdatedAt, comment.ID)
	if err != nil {
		return fmt.Errorf("updating review comment: %w", err)
	}
	return nil
}

// Resolve resolves a review comment
func (s *ReviewCommentStore) Resolve(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE review_comments SET status = 'resolved', updated_at = ? WHERE id = ?`,
		time.Now(), id)
	if err != nil {
		return fmt.Errorf("resolving review comment: %w", err)
	}
	return nil
}

// Delete deletes a review comment
func (s *ReviewCommentStore) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM review_comments WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("deleting review comment: %w", err)
	}
	return nil
}
