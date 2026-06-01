package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/lakshmanpatel/gitant/internal/store"
)

// SQLReviewCommentStore implements the store.ReviewCommentStore interface using SQLite
type SQLReviewCommentStore struct {
	db *sql.DB
}

// Create creates a new review comment
func (s *SQLReviewCommentStore) Create(ctx context.Context, comment *store.ReviewComment) error {
	query := `
		INSERT INTO review_comments (id, pr_id, repo_id, file_path, line_number, author_id, body, parent_id, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := s.db.ExecContext(ctx, query,
		comment.ID,
		comment.PRID,
		"", // repo_id is derived from PR
		comment.FilePath,
		comment.LineNumber,
		comment.AuthorID,
		comment.Body,
		comment.ParentID,
		comment.Status,
		comment.CreatedAt,
		comment.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("creating review comment: %w", err)
	}

	return nil
}

// Get retrieves a review comment by ID
func (s *SQLReviewCommentStore) Get(ctx context.Context, id string) (*store.ReviewComment, error) {
	query := `
		SELECT id, pr_id, file_path, line_number, author_id, body, parent_id, status, created_at, updated_at
		FROM review_comments
		WHERE id = ?
	`

	comment := &store.ReviewComment{}
	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&comment.ID,
		&comment.PRID,
		&comment.FilePath,
		&comment.LineNumber,
		&comment.AuthorID,
		&comment.Body,
		&comment.ParentID,
		&comment.Status,
		&comment.CreatedAt,
		&comment.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("review comment not found: %s", id)
		}
		return nil, fmt.Errorf("getting review comment: %w", err)
	}

	return comment, nil
}

// ListByPR lists all review comments for a pull request
func (s *SQLReviewCommentStore) ListByPR(ctx context.Context, prID string) ([]*store.ReviewComment, error) {
	query := `
		SELECT id, pr_id, file_path, line_number, author_id, body, parent_id, status, created_at, updated_at
		FROM review_comments
		WHERE pr_id = ?
		ORDER BY created_at
	`

	rows, err := s.db.QueryContext(ctx, query, prID)
	if err != nil {
		return nil, fmt.Errorf("listing review comments: %w", err)
	}
	defer rows.Close()

	var comments []*store.ReviewComment
	for rows.Next() {
		comment := &store.ReviewComment{}
		err := rows.Scan(
			&comment.ID,
			&comment.PRID,
			&comment.FilePath,
			&comment.LineNumber,
			&comment.AuthorID,
			&comment.Body,
			&comment.ParentID,
			&comment.Status,
			&comment.CreatedAt,
			&comment.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scanning review comment: %w", err)
		}
		comments = append(comments, comment)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating review comments: %w", err)
	}

	return comments, nil
}

// Update updates a review comment
func (s *SQLReviewCommentStore) Update(ctx context.Context, comment *store.ReviewComment) error {
	query := `
		UPDATE review_comments
		SET body = ?, status = ?, updated_at = ?
		WHERE id = ?
	`

	_, err := s.db.ExecContext(ctx, query,
		comment.Body,
		comment.Status,
		time.Now(),
		comment.ID,
	)

	if err != nil {
		return fmt.Errorf("updating review comment: %w", err)
	}

	return nil
}

// Resolve marks a review comment as resolved
func (s *SQLReviewCommentStore) Resolve(ctx context.Context, id string) error {
	query := `
		UPDATE review_comments
		SET status = 'resolved', updated_at = ?
		WHERE id = ?
	`

	_, err := s.db.ExecContext(ctx, query, time.Now(), id)
	if err != nil {
		return fmt.Errorf("resolving review comment: %w", err)
	}

	return nil
}

// Delete removes a review comment
func (s *SQLReviewCommentStore) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM review_comments WHERE id = ?`

	_, err := s.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("deleting review comment: %w", err)
	}

	return nil
}
