package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/lakshmanpatel/gitant/internal/store"
)

// SQLReleaseStore implements the store.ReleaseStore interface using SQLite
type SQLReleaseStore struct {
	db *sql.DB
}

// Create creates a new release
func (s *SQLReleaseStore) Create(ctx context.Context, repoID, tag, title, body, author string) (*store.Release, error) {
	query := `
		INSERT INTO releases (id, repo_id, tag, title, body, author, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`

	id := fmt.Sprintf("%s-%s", repoID, tag)
	now := time.Now()

	_, err := s.db.ExecContext(ctx, query,
		id, repoID, tag, title, body, author, now,
	)

	if err != nil {
		return nil, fmt.Errorf("creating release: %w", err)
	}

	release := &store.Release{
		ID:        id,
		RepoID:    repoID,
		Tag:       tag,
		Title:     title,
		Body:      body,
		Author:    author,
		CreatedAt: now,
	}

	return release, nil
}

// Get retrieves a release by ID
func (s *SQLReleaseStore) Get(ctx context.Context, repoID, releaseID string) (*store.Release, error) {
	query := `
		SELECT id, repo_id, tag, title, body, author, created_at
		FROM releases
		WHERE id = ? AND repo_id = ?
	`

	release := &store.Release{}
	err := s.db.QueryRowContext(ctx, query, releaseID, repoID).Scan(
		&release.ID,
		&release.RepoID,
		&release.Tag,
		&release.Title,
		&release.Body,
		&release.Author,
		&release.CreatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("release not found: %s", releaseID)
		}
		return nil, fmt.Errorf("getting release: %w", err)
	}

	return release, nil
}

// List lists releases for a repository
func (s *SQLReleaseStore) List(ctx context.Context, repoID string) ([]*store.Release, error) {
	query := `
		SELECT id, repo_id, tag, title, body, author, created_at
		FROM releases
		WHERE repo_id = ?
		ORDER BY created_at DESC
	`

	rows, err := s.db.QueryContext(ctx, query, repoID)
	if err != nil {
		return nil, fmt.Errorf("listing releases: %w", err)
	}
	defer rows.Close()

	var releases []*store.Release
	for rows.Next() {
		release := &store.Release{}
		err := rows.Scan(
			&release.ID,
			&release.RepoID,
			&release.Tag,
			&release.Title,
			&release.Body,
			&release.Author,
			&release.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scanning release: %w", err)
		}
		releases = append(releases, release)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating releases: %w", err)
	}

	return releases, nil
}

// Delete removes a release
func (s *SQLReleaseStore) Delete(ctx context.Context, repoID, releaseID string) error {
	query := `DELETE FROM releases WHERE id = ? AND repo_id = ?`

	_, err := s.db.ExecContext(ctx, query, releaseID, repoID)
	if err != nil {
		return fmt.Errorf("deleting release: %w", err)
	}

	return nil
}

// Save is a no-op for SQL stores (data is persisted on every operation)
func (s *SQLReleaseStore) Save() error {
	return nil
}
