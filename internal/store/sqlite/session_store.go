package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/lakshmanpatel/gitant/internal/store"
)

// SQLSessionStore implements SessionStore using SQLite
type SQLSessionStore struct {
	db *sql.DB
}

// Create creates a new session
func (s *SQLSessionStore) Create(ctx context.Context, session *store.Session) error {
	query := `
		INSERT INTO sessions (id, user_id, token, expires_at, created_at)
		VALUES (?, ?, ?, ?, ?)
	`

	_, err := s.db.ExecContext(ctx, query,
		session.ID,
		session.UserID,
		session.Token,
		session.ExpiresAt,
		session.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("creating session: %w", err)
	}

	return nil
}

// Get retrieves a session by token
func (s *SQLSessionStore) Get(ctx context.Context, token string) (*store.Session, error) {
	query := `
		SELECT id, user_id, token, expires_at, created_at
		FROM sessions
		WHERE token = ?
	`

	session := &store.Session{}
	err := s.db.QueryRowContext(ctx, query, token).Scan(
		&session.ID,
		&session.UserID,
		&session.Token,
		&session.ExpiresAt,
		&session.CreatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("session not found")
		}
		return nil, fmt.Errorf("getting session: %w", err)
	}

	return session, nil
}

// Delete removes a session by token
func (s *SQLSessionStore) Delete(ctx context.Context, token string) error {
	query := `DELETE FROM sessions WHERE token = ?`

	result, err := s.db.ExecContext(ctx, query, token)
	if err != nil {
		return fmt.Errorf("deleting session: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("session not found")
	}

	return nil
}

// DeleteExpired removes all expired sessions
func (s *SQLSessionStore) DeleteExpired(ctx context.Context) error {
	query := `DELETE FROM sessions WHERE expires_at < ?`

	_, err := s.db.ExecContext(ctx, query, time.Now())
	if err != nil {
		return fmt.Errorf("deleting expired sessions: %w", err)
	}

	return nil
}
