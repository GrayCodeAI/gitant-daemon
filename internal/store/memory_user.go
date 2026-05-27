package store

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// MemoryUserStore is an in-memory UserStore with optional JSON file persistence.
type MemoryUserStore struct {
	mu       sync.RWMutex
	users    map[string]*User    // id -> user
	byName   map[string]string   // username -> id
	byEmail  map[string]string   // email -> id
	savePath string             // if non-empty, persist to this file
}

// NewMemoryUserStore creates a new MemoryUserStore.
// If savePath is non-empty, data is persisted to that JSON file.
func NewMemoryUserStore(savePath string) *MemoryUserStore {
	s := &MemoryUserStore{
		users:    make(map[string]*User),
		byName:   make(map[string]string),
		byEmail:  make(map[string]string),
		savePath: savePath,
	}
	if savePath != "" {
		s.load()
	}
	return s
}

func (s *MemoryUserStore) Create(_ context.Context, user *User) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.users[user.ID]; exists {
		return fmt.Errorf("user already exists: %s", user.ID)
	}
	if _, exists := s.byName[user.Username]; exists {
		return fmt.Errorf("username already taken: %s", user.Username)
	}
	if _, exists := s.byEmail[user.Email]; exists {
		return fmt.Errorf("email already taken: %s", user.Email)
	}

	s.users[user.ID] = user
	s.byName[user.Username] = user.ID
	s.byEmail[user.Email] = user.ID
	return s.saveLocked()
}

func (s *MemoryUserStore) Get(_ context.Context, id string) (*User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	user, ok := s.users[id]
	if !ok {
		return nil, fmt.Errorf("user not found: %s", id)
	}
	return user, nil
}

func (s *MemoryUserStore) GetByUsername(_ context.Context, username string) (*User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	id, ok := s.byName[username]
	if !ok {
		return nil, fmt.Errorf("user not found: %s", username)
	}
	return s.users[id], nil
}

func (s *MemoryUserStore) GetByEmail(_ context.Context, email string) (*User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	id, ok := s.byEmail[email]
	if !ok {
		return nil, fmt.Errorf("user not found: %s", email)
	}
	return s.users[id], nil
}

func (s *MemoryUserStore) Update(_ context.Context, user *User) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	existing, ok := s.users[user.ID]
	if !ok {
		return fmt.Errorf("user not found: %s", user.ID)
	}
	// Update reverse indexes if username/email changed
	if existing.Username != user.Username {
		delete(s.byName, existing.Username)
		s.byName[user.Username] = user.ID
	}
	if existing.Email != user.Email {
		delete(s.byEmail, existing.Email)
		s.byEmail[user.Email] = user.ID
	}
	s.users[user.ID] = user
	return s.saveLocked()
}

func (s *MemoryUserStore) Delete(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	user, ok := s.users[id]
	if !ok {
		return fmt.Errorf("user not found: %s", id)
	}
	delete(s.byName, user.Username)
	delete(s.byEmail, user.Email)
	delete(s.users, id)
	return s.saveLocked()
}

func (s *MemoryUserStore) List(_ context.Context) ([]*User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	users := make([]*User, 0, len(s.users))
	for _, u := range s.users {
		users = append(users, u)
	}
	return users, nil
}

func (s *MemoryUserStore) saveLocked() error {
	if s.savePath == "" {
		return nil
	}
	dir := filepath.Dir(s.savePath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating user store dir: %w", err)
	}
	data, err := json.MarshalIndent(s.users, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.savePath, data, 0o644)
}

func (s *MemoryUserStore) load() {
	data, err := os.ReadFile(s.savePath)
	if err != nil {
		return // file doesn't exist yet
	}
	var users map[string]*User
	if err := json.Unmarshal(data, &users); err != nil {
		return
	}
	s.users = users
	for _, u := range users {
		s.byName[u.Username] = u.ID
		s.byEmail[u.Email] = u.ID
	}
}
