package sqlite

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/lakshmanpatel/gitant/internal/store"
)

// SQLProtectionStore implements the store.ProtectionStore interface using SQLite
type SQLProtectionStore struct {
	db *sql.DB
}

// Get retrieves a branch protection rule
func (s *SQLProtectionStore) Get(ctx context.Context, repoID, branch string) (*store.BranchProtection, error) {
	query := `
		SELECT branch, require_pr, require_approval, no_force_push
		FROM branch_protections
		WHERE repo_id = ? AND branch = ?
	`

	protection := &store.BranchProtection{}
	var requirePr, requireApproval, noForcePush int

	err := s.db.QueryRowContext(ctx, query, repoID, branch).Scan(
		&protection.Branch,
		&requirePr,
		&requireApproval,
		&noForcePush,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("protection rule not found for branch: %s", branch)
		}
		return nil, fmt.Errorf("getting protection: %w", err)
	}

	protection.RequirePR = requirePr == 1
	protection.RequireApproval = requireApproval == 1
	protection.NoForcePush = noForcePush == 1

	return protection, nil
}

// List lists all protection rules for a repository
func (s *SQLProtectionStore) List(ctx context.Context, repoID string) ([]store.BranchProtection, error) {
	query := `
		SELECT branch, require_pr, require_approval, no_force_push
		FROM branch_protections
		WHERE repo_id = ?
	`

	rows, err := s.db.QueryContext(ctx, query, repoID)
	if err != nil {
		return nil, fmt.Errorf("listing protections: %w", err)
	}
	defer rows.Close()

	var protections []store.BranchProtection
	for rows.Next() {
		var protection store.BranchProtection
		var requirePr, requireApproval, noForcePush int

		err := rows.Scan(
			&protection.Branch,
			&requirePr,
			&requireApproval,
			&noForcePush,
		)
		if err != nil {
			return nil, fmt.Errorf("scanning protection: %w", err)
		}

		protection.RequirePR = requirePr == 1
		protection.RequireApproval = requireApproval == 1
		protection.NoForcePush = noForcePush == 1

		protections = append(protections, protection)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating protections: %w", err)
	}

	return protections, nil
}

// Set creates or updates a branch protection rule
func (s *SQLProtectionStore) Set(ctx context.Context, repoID string, protection store.BranchProtection) error {
	query := `
		INSERT INTO branch_protections (repo_id, branch, require_pr, require_approval, no_force_push)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(repo_id, branch) DO UPDATE SET
			require_pr = excluded.require_pr,
			require_approval = excluded.require_approval,
			no_force_push = excluded.no_force_push
	`

	requirePr := 0
	if protection.RequirePR {
		requirePr = 1
	}

	requireApproval := 0
	if protection.RequireApproval {
		requireApproval = 1
	}

	noForcePush := 0
	if protection.NoForcePush {
		noForcePush = 1
	}

	_, err := s.db.ExecContext(ctx, query,
		repoID, protection.Branch, requirePr, requireApproval, noForcePush,
	)

	if err != nil {
		return fmt.Errorf("setting protection: %w", err)
	}

	return nil
}

// Remove removes a branch protection rule
func (s *SQLProtectionStore) Remove(ctx context.Context, repoID, branch string) error {
	query := `DELETE FROM branch_protections WHERE repo_id = ? AND branch = ?`

	_, err := s.db.ExecContext(ctx, query, repoID, branch)
	if err != nil {
		return fmt.Errorf("removing protection: %w", err)
	}

	return nil
}

// Save is a no-op for SQL stores (data is persisted on every operation)
func (s *SQLProtectionStore) Save() error {
	return nil
}
