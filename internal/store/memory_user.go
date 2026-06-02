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
	mu            sync.RWMutex
	users         map[string]*User   // id -> user
	byName        map[string]string  // username -> id
	byEmail       map[string]string  // email -> id
	sshKeys       map[string]*SSHKey // key_id -> ssh_key
	byFingerprint map[string]string  // fingerprint -> key_id
	savePath      string             // if non-empty, persist to this file
}

// NewMemoryUserStore creates a new MemoryUserStore.
// If savePath is non-empty, data is persisted to that JSON file.
func NewMemoryUserStore(savePath string) *MemoryUserStore {
	s := &MemoryUserStore{
		users:         make(map[string]*User),
		byName:        make(map[string]string),
		byEmail:       make(map[string]string),
		sshKeys:       make(map[string]*SSHKey),
		byFingerprint: make(map[string]string),
		savePath:      savePath,
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

func (s *MemoryUserStore) AddSSHKey(_ context.Context, key *SSHKey) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.sshKeys[key.ID]; exists {
		return fmt.Errorf("SSH key already exists: %s", key.ID)
	}
	if _, exists := s.byFingerprint[key.Fingerprint]; exists {
		return fmt.Errorf("SSH key fingerprint already registered: %s", key.Fingerprint)
	}
	s.sshKeys[key.ID] = key
	s.byFingerprint[key.Fingerprint] = key.ID
	return s.saveLocked()
}

func (s *MemoryUserStore) DeleteSSHKey(_ context.Context, userID, keyID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	key, ok := s.sshKeys[keyID]
	if !ok {
		return fmt.Errorf("SSH key not found: %s", keyID)
	}
	if key.UserID != userID {
		return fmt.Errorf("SSH key does not belong to user")
	}
	delete(s.byFingerprint, key.Fingerprint)
	delete(s.sshKeys, keyID)
	return s.saveLocked()
}

func (s *MemoryUserStore) ListSSHKeys(_ context.Context, userID string) ([]SSHKey, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var keys []SSHKey
	for _, key := range s.sshKeys {
		if key.UserID == userID {
			keys = append(keys, *key)
		}
	}
	return keys, nil
}

func (s *MemoryUserStore) FindByFingerprint(_ context.Context, fingerprint string) (*User, *SSHKey, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	keyID, ok := s.byFingerprint[fingerprint]
	if !ok {
		return nil, nil, fmt.Errorf("no user found for fingerprint: %s", fingerprint)
	}
	key := s.sshKeys[keyID]
	user, ok := s.users[key.UserID]
	if !ok {
		return nil, nil, fmt.Errorf("user not found for key: %s", keyID)
	}
	return user, key, nil
}

func (s *MemoryUserStore) saveLocked() error {
	if s.savePath == "" {
		return nil
	}
	dir := filepath.Dir(s.savePath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating user store dir: %w", err)
	}
	// Serialize both users and SSH keys
	data, err := json.MarshalIndent(struct {
		Users   map[string]*User   `json:"users"`
		SSHKeys map[string]*SSHKey `json:"ssh_keys"`
	}{
		Users:   s.users,
		SSHKeys: s.sshKeys,
	}, "", "  ")
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
