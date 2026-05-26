package sqlite

import (
	"context"
	"fmt"

	"github.com/lakshmanpatel/gitant/internal/store"
)

// ProtectionStore implements store.ProtectionStore for SQLite
type ProtectionStore struct {
	db *DB
}

// NewProtectionStore creates a new SQLite protection store
func NewProtectionStore(db *DB) *ProtectionStore {
	return &ProtectionStore{db: db}
}

// Get gets protection for a branch
func (s *ProtectionStore) Get(ctx context.Context, repoID, branch string) (*store.BranchProtection, error) {
	p := &store.BranchProtection{}
	err := s.db.QueryRowContext(ctx,
		`SELECT branch, require_pr, require_approval, no_force_push
		 FROM branch_protections WHERE repo_id = ? AND branch = ?`, repoID, branch,
	).Scan(&p.Branch, &p.RequirePR, &p.RequireApproval, &p.NoForcePush)
	if err != nil {
		return nil, fmt.Errorf("protection not found: %s", branch)
	}
	return p, nil
}

// List lists all protections for a repository
func (s *ProtectionStore) List(ctx context.Context, repoID string) ([]store.BranchProtection, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT branch, require_pr, require_approval, no_force_push
		 FROM branch_protections WHERE repo_id = ? ORDER BY branch`, repoID)
	if err != nil {
		return nil, fmt.Errorf("listing protections: %w", err)
	}
	defer rows.Close()

	protections := []store.BranchProtection{}
	for rows.Next() {
		var p store.BranchProtection
		if err := rows.Scan(&p.Branch, &p.RequirePR, &p.RequireApproval, &p.NoForcePush); err != nil {
			return nil, fmt.Errorf("scanning protection: %w", err)
		}
		protections = append(protections, p)
	}
	return protections, nil
}

// Set creates or updates protection for a branch
func (s *ProtectionStore) Set(ctx context.Context, repoID string, p store.BranchProtection) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT OR REPLACE INTO branch_protections (repo_id, branch, require_pr, require_approval, no_force_push)
		 VALUES (?, ?, ?, ?, ?)`,
		repoID, p.Branch, p.RequirePR, p.RequireApproval, p.NoForcePush)
	if err != nil {
		return fmt.Errorf("setting protection: %w", err)
	}
	return nil
}

// Remove removes protection for a branch
func (s *ProtectionStore) Remove(ctx context.Context, repoID, branch string) error {
	result, err := s.db.ExecContext(ctx,
		"DELETE FROM branch_protections WHERE repo_id = ? AND branch = ?", repoID, branch)
	if err != nil {
		return fmt.Errorf("removing protection: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("protection not found: %s", branch)
	}
	return nil
}

// Save is a no-op for SQLite
func (s *ProtectionStore) Save() error {
	return nil
}
