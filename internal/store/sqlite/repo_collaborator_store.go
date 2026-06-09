package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/lakshmanpatel/gitant/internal/store"
)

// SQLRepoCollaboratorStore implements RepoCollaboratorStore using SQLite.
type SQLRepoCollaboratorStore struct {
	db *sql.DB
}

// Add adds or updates a repo collaborator membership.
func (s *SQLRepoCollaboratorStore) Add(ctx context.Context, collaborator *store.RepoCollaborator) error {
	if collaborator == nil {
		return fmt.Errorf("repo collaborator is required")
	}
	if collaborator.RepoID == "" {
		return fmt.Errorf("repo id is required")
	}
	if collaborator.UserID == "" {
		return fmt.Errorf("user id is required")
	}
	if collaborator.Role == "" {
		collaborator.Role = store.RepoRoleCollaborator
	}
	if collaborator.CreatedAt.IsZero() {
		collaborator.CreatedAt = time.Now()
	}
	collaborator.UpdatedAt = time.Now()

	query := `
		INSERT INTO repo_collaborators (repo_id, user_id, role, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(repo_id, user_id) DO UPDATE SET
			role = excluded.role,
			updated_at = excluded.updated_at
	`
	_, err := s.db.ExecContext(ctx, query,
		collaborator.RepoID,
		collaborator.UserID,
		collaborator.Role,
		collaborator.CreatedAt,
		collaborator.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("adding repo collaborator: %w", err)
	}
	return nil
}

// Get returns a repo collaborator membership.
func (s *SQLRepoCollaboratorStore) Get(ctx context.Context, repoID, userID string) (*store.RepoCollaborator, error) {
	query := `
		SELECT repo_id, user_id, role, created_at, updated_at
		FROM repo_collaborators
		WHERE repo_id = ? AND user_id = ?
	`
	collaborator := &store.RepoCollaborator{}
	err := s.db.QueryRowContext(ctx, query, repoID, userID).Scan(
		&collaborator.RepoID,
		&collaborator.UserID,
		&collaborator.Role,
		&collaborator.CreatedAt,
		&collaborator.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("repo collaborator not found")
		}
		return nil, fmt.Errorf("getting repo collaborator: %w", err)
	}
	return collaborator, nil
}

// ListByRepo lists collaborators for a repository.
func (s *SQLRepoCollaboratorStore) ListByRepo(ctx context.Context, repoID string) ([]*store.RepoCollaborator, error) {
	query := `
		SELECT repo_id, user_id, role, created_at, updated_at
		FROM repo_collaborators
		WHERE repo_id = ?
		ORDER BY created_at ASC
	`
	rows, err := s.db.QueryContext(ctx, query, repoID)
	if err != nil {
		return nil, fmt.Errorf("listing repo collaborators: %w", err)
	}
	defer rows.Close()

	var collaborators []*store.RepoCollaborator
	for rows.Next() {
		collaborator := &store.RepoCollaborator{}
		if err := rows.Scan(
			&collaborator.RepoID,
			&collaborator.UserID,
			&collaborator.Role,
			&collaborator.CreatedAt,
			&collaborator.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scanning repo collaborator: %w", err)
		}
		collaborators = append(collaborators, collaborator)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating repo collaborators: %w", err)
	}
	return collaborators, nil
}

// Remove removes a repo collaborator membership.
func (s *SQLRepoCollaboratorStore) Remove(ctx context.Context, repoID, userID string) error {
	query := `DELETE FROM repo_collaborators WHERE repo_id = ? AND user_id = ?`
	result, err := s.db.ExecContext(ctx, query, repoID, userID)
	if err != nil {
		return fmt.Errorf("removing repo collaborator: %w", err)
	}
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("repo collaborator not found")
	}
	return nil
}

// IsWriter reports whether the user owns or collaborates on the repo.
func (s *SQLRepoCollaboratorStore) IsWriter(ctx context.Context, repoID, userID string) (bool, error) {
	collaborator, err := s.Get(ctx, repoID, userID)
	if err != nil {
		return false, nil
	}
	return collaborator.CanWrite(), nil
}
