package sqlite

import (
	"context"
	"fmt"

	"github.com/lakshmanpatel/gitant/internal/store"
)

// LabelStore implements store.LabelStore for SQLite
type LabelStore struct {
	db *DB
}

// NewLabelStore creates a new SQLite label store
func NewLabelStore(db *DB) *LabelStore {
	return &LabelStore{db: db}
}

// List lists all labels for a repository
func (s *LabelStore) List(ctx context.Context, repoID string) ([]store.Label, error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT name, color FROM labels WHERE repo_id = ? ORDER BY name", repoID)
	if err != nil {
		return nil, fmt.Errorf("listing labels: %w", err)
	}
	defer rows.Close()

	labels := []store.Label{}
	for rows.Next() {
		var l store.Label
		if err := rows.Scan(&l.Name, &l.Color); err != nil {
			return nil, fmt.Errorf("scanning label: %w", err)
		}
		labels = append(labels, l)
	}
	return labels, nil
}

// Add adds a label to a repository
func (s *LabelStore) Add(ctx context.Context, repoID, name, color string) error {
	if color == "" {
		color = "#6b7280"
	}
	_, err := s.db.ExecContext(ctx,
		"INSERT OR IGNORE INTO labels (repo_id, name, color) VALUES (?, ?, ?)",
		repoID, name, color)
	if err != nil {
		return fmt.Errorf("adding label: %w", err)
	}
	return nil
}

// Remove removes a label from a repository
func (s *LabelStore) Remove(ctx context.Context, repoID, name string) error {
	result, err := s.db.ExecContext(ctx,
		"DELETE FROM labels WHERE repo_id = ? AND name = ?", repoID, name)
	if err != nil {
		return fmt.Errorf("removing label: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("label not found: %s", name)
	}
	return nil
}

// Save is a no-op for SQLite
func (s *LabelStore) Save() error {
	return nil
}
