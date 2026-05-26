package sqlite

import (
	"context"
	"fmt"
	"time"

	"github.com/lakshmanpatel/gitant/internal/store"
)

// ReleaseStore implements store.ReleaseStore for SQLite
type ReleaseStore struct {
	db *DB
}

// NewReleaseStore creates a new SQLite release store
func NewReleaseStore(db *DB) *ReleaseStore {
	return &ReleaseStore{db: db}
}

// Create creates a new release
func (s *ReleaseStore) Create(ctx context.Context, repoID, tag, title, body, author string) (*store.Release, error) {
	now := time.Now()
	id := fmt.Sprintf("rel-%d", now.UnixNano())

	_, err := s.db.ExecContext(ctx,
		`INSERT INTO releases (id, repo_id, tag, title, body, author, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		id, repoID, tag, title, body, author, now,
	)
	if err != nil {
		return nil, fmt.Errorf("creating release: %w", err)
	}

	return &store.Release{
		ID:        id,
		RepoID:    repoID,
		Tag:       tag,
		Title:     title,
		Body:      body,
		Author:    author,
		CreatedAt: now,
	}, nil
}

// Get gets a release by ID
func (s *ReleaseStore) Get(ctx context.Context, repoID, releaseID string) (*store.Release, error) {
	r := &store.Release{}
	err := s.db.QueryRowContext(ctx,
		`SELECT id, repo_id, tag, title, body, author, created_at
		 FROM releases WHERE id = ? AND repo_id = ?`, releaseID, repoID,
	).Scan(&r.ID, &r.RepoID, &r.Tag, &r.Title, &r.Body, &r.Author, &r.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("getting release: %w", err)
	}
	return r, nil
}

// List lists releases for a repository
func (s *ReleaseStore) List(ctx context.Context, repoID string) ([]*store.Release, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, repo_id, tag, title, body, author, created_at
		 FROM releases WHERE repo_id = ? ORDER BY created_at DESC`, repoID)
	if err != nil {
		return nil, fmt.Errorf("listing releases: %w", err)
	}
	defer rows.Close()

	var releases []*store.Release
	for rows.Next() {
		r := &store.Release{}
		if err := rows.Scan(&r.ID, &r.RepoID, &r.Tag, &r.Title, &r.Body, &r.Author, &r.CreatedAt); err != nil {
			return nil, fmt.Errorf("scanning release: %w", err)
		}
		releases = append(releases, r)
	}
	if releases == nil {
		releases = []*store.Release{}
	}
	return releases, nil
}

// Delete deletes a release
func (s *ReleaseStore) Delete(ctx context.Context, repoID, releaseID string) error {
	result, err := s.db.ExecContext(ctx,
		"DELETE FROM releases WHERE id = ? AND repo_id = ?", releaseID, repoID)
	if err != nil {
		return fmt.Errorf("deleting release: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("release not found: %s", releaseID)
	}
	return nil
}

// Save is a no-op for SQLite
func (s *ReleaseStore) Save() error {
	return nil
}
