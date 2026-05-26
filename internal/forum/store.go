package forum

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Thread represents a forum thread
type Thread struct {
	ID        string    `json:"id"`
	RepoID    string    `json:"repo_id"`
	Title     string    `json:"title"`
	Body      string    `json:"body"`
	Author    string    `json:"author"`
	Category  string    `json:"category"` // "general", "help", "announcement", "rfc"
	Pinned    bool      `json:"pinned"`
	Locked    bool      `json:"locked"`
	Replies   []Reply   `json:"replies"`
	Views     int       `json:"views"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Reply represents a forum reply
type Reply struct {
	ID        string    `json:"id"`
	Body      string    `json:"body"`
	Author    string    `json:"author"`
	CreatedAt time.Time `json:"created_at"`
}

// Store manages forum threads
type Store struct {
	mu      sync.RWMutex
	baseDir string
	threads map[string][]*Thread
}

// NewStore creates a new forum store
func NewStore(baseDir string) *Store {
	return &Store{
		baseDir: baseDir,
		threads: make(map[string][]*Thread),
	}
}

// Load loads forum data from disk
func (s *Store) Load() error {
	path := filepath.Join(s.baseDir, "forum.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return json.Unmarshal(data, &s.threads)
}

// Save saves forum data to disk
func (s *Store) Save() error {
	path := filepath.Join(s.baseDir, "forum.json")
	data, err := json.MarshalIndent(s.threads, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// CreateThread creates a new thread
func (s *Store) CreateThread(thread *Thread) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	thread.ID = fmt.Sprintf("thread-%d", time.Now().UnixNano())
	thread.CreatedAt = time.Now()
	thread.UpdatedAt = time.Now()
	if thread.Replies == nil {
		thread.Replies = []Reply{}
	}

	s.threads[thread.RepoID] = append(s.threads[thread.RepoID], thread)
	return s.Save()
}

// GetThread gets a thread by ID
func (s *Store) GetThread(repoID, threadID string) (*Thread, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, t := range s.threads[repoID] {
		if t.ID == threadID {
			t.Views++
			return t, nil
		}
	}
	return nil, fmt.Errorf("thread not found")
}

// ListThreads lists threads for a repository
func (s *Store) ListThreads(repoID, category string) []*Thread {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*Thread
	for _, t := range s.threads[repoID] {
		if category != "" && t.Category != category {
			continue
		}
		result = append(result, t)
	}
	return result
}

// AddReply adds a reply to a thread
func (s *Store) AddReply(repoID, threadID, author, body string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, t := range s.threads[repoID] {
		if t.ID == threadID {
			reply := Reply{
				ID:        fmt.Sprintf("reply-%d", time.Now().UnixNano()),
				Body:      body,
				Author:    author,
				CreatedAt: time.Now(),
			}
			t.Replies = append(t.Replies, reply)
			t.UpdatedAt = time.Now()
			return s.Save()
		}
	}
	return fmt.Errorf("thread not found")
}

// PinThread pins a thread
func (s *Store) PinThread(repoID, threadID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, t := range s.threads[repoID] {
		if t.ID == threadID {
			t.Pinned = true
			t.UpdatedAt = time.Now()
			return s.Save()
		}
	}
	return fmt.Errorf("thread not found")
}

// LockThread locks a thread
func (s *Store) LockThread(repoID, threadID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, t := range s.threads[repoID] {
		if t.ID == threadID {
			t.Locked = true
			t.UpdatedAt = time.Now()
			return s.Save()
		}
	}
	return fmt.Errorf("thread not found")
}

// DeleteThread deletes a thread
func (s *Store) DeleteThread(repoID, threadID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	threads := s.threads[repoID]
	for i, t := range threads {
		if t.ID == threadID {
			s.threads[repoID] = append(threads[:i], threads[i+1:]...)
			return s.Save()
		}
	}
	return fmt.Errorf("thread not found")
}
