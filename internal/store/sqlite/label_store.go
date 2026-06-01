package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/lakshmanpatel/gitant/internal/store"
)

// SQLLabelStore implements the store.LabelStore interface using SQLite
type SQLLabelStore struct {
	db *sql.DB
}

// List lists all labels for a repository
func (s *SQLLabelStore) List(ctx context.Context, repoID string) ([]store.Label, error) {
	query := `
		SELECT name, color
		FROM labels
		WHERE repo_id = ?
		ORDER BY name
	`

	rows, err := s.db.QueryContext(ctx, query, repoID)
	if err != nil {
		return nil, fmt.Errorf("listing labels: %w", err)
	}
	defer rows.Close()

	var labels []store.Label
	for rows.Next() {
		var label store.Label
		err := rows.Scan(&label.Name, &label.Color)
		if err != nil {
			return nil, fmt.Errorf("scanning label: %w", err)
		}
		labels = append(labels, label)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating labels: %w", err)
	}

	return labels, nil
}

// Add adds a new label to a repository
func (s *SQLLabelStore) Add(ctx context.Context, repoID, name, color string) error {
	query := `
		INSERT INTO labels (repo_id, name, color, created_at)
		VALUES (?, ?, ?, ?)
	`

	_, err := s.db.ExecContext(ctx, query, repoID, name, color, time.Now())
	if err != nil {
		return fmt.Errorf("adding label: %w", err)
	}

	return nil
}

// Remove removes a label from a repository
func (s *SQLLabelStore) Remove(ctx context.Context, repoID, name string) error {
	query := `DELETE FROM labels WHERE repo_id = ? AND name = ?`

	_, err := s.db.ExecContext(ctx, query, repoID, name)
	if err != nil {
		return fmt.Errorf("removing label: %w", err)
	}

	return nil
}

// Save is a no-op for SQL stores (data is persisted on every operation)
func (s *SQLLabelStore) Save() error {
	return nil
}
