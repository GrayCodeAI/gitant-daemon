package sqlite

import (
	"context"
	"fmt"
	"time"

	"github.com/lakshmanpatel/gitant/internal/store"
)

// SessionStore implements store.SessionStore for SQLite
type SessionStore struct {
	db *DB
}

// NewSessionStore creates a new SQLite session store
func NewSessionStore(db *DB) *SessionStore {
	return &SessionStore{db: db}
}

// Create creates a new session
func (s *SessionStore) Create(ctx context.Context, session *store.Session) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO sessions (id, user_id, token, expires_at, created_at)
		 VALUES (?, ?, ?, ?, ?)`,
		session.ID, session.UserID, session.Token,
		session.ExpiresAt, session.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("creating session: %w", err)
	}
	return nil
}

// Get gets a session by token
func (s *SessionStore) Get(ctx context.Context, token string) (*store.Session, error) {
	session := &store.Session{}
	err := s.db.QueryRowContext(ctx,
		`SELECT id, user_id, token, expires_at, created_at
		 FROM sessions WHERE token = ? AND expires_at > ?`,
		token, time.Now(),
	).Scan(&session.ID, &session.UserID, &session.Token,
		&session.ExpiresAt, &session.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("getting session: %w", err)
	}
	return session, nil
}

// Delete deletes a session
func (s *SessionStore) Delete(ctx context.Context, token string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM sessions WHERE token = ?", token)
	if err != nil {
		return fmt.Errorf("deleting session: %w", err)
	}
	return nil
}

// DeleteExpired deletes expired sessions
func (s *SessionStore) DeleteExpired(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM sessions WHERE expires_at < ?", time.Now())
	if err != nil {
		return fmt.Errorf("deleting expired sessions: %w", err)
	}
	return nil
}
