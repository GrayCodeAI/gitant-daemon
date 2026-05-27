package store

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// MemorySessionStore is an in-memory implementation of SessionStore with TTL-based expiry.
type MemorySessionStore struct {
	mu       sync.RWMutex
	sessions map[string]*Session // token -> session
}

// NewMemorySessionStore creates a new MemorySessionStore.
func NewMemorySessionStore() *MemorySessionStore {
	return &MemorySessionStore{
		sessions: make(map[string]*Session),
	}
}

func (s *MemorySessionStore) Create(_ context.Context, session *Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[session.Token] = session
	return nil
}

func (s *MemorySessionStore) Get(_ context.Context, token string) (*Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	session, ok := s.sessions[token]
	if !ok {
		return nil, fmt.Errorf("session not found")
	}
	return session, nil
}

func (s *MemorySessionStore) Delete(_ context.Context, token string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, token)
	return nil
}

func (s *MemorySessionStore) DeleteExpired(_ context.Context) error {
	now := time.Now()
	s.mu.Lock()
	defer s.mu.Unlock()
	for token, session := range s.sessions {
		if session.ExpiresAt.Before(now) {
			delete(s.sessions, token)
		}
	}
	return nil
}
