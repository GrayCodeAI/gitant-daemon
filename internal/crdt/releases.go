package crdt

import (
	crypto_rand "crypto/rand"
	"encoding/json"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/lakshmanpatel/gitant/internal/persistence"
)

// Release represents a CRDT release
type Release struct {
	ID        string    `json:"id"`
	RepoID    string    `json:"repo_id"`
	Tag       string    `json:"tag"`
	Title     string    `json:"title"`
	Body      string    `json:"body"`
	Author    string    `json:"author"` // DID
	CreatedAt time.Time `json:"created_at"`
	log       *OperationLog
}

// MarshalJSON serializes a Release including its operation log
func (r *Release) MarshalJSON() ([]byte, error) {
	type releaseJSON struct {
		ID        string       `json:"id"`
		RepoID    string       `json:"repo_id"`
		Tag       string       `json:"tag"`
		Title     string       `json:"title"`
		Body      string       `json:"body"`
		Author    string       `json:"author"`
		CreatedAt time.Time    `json:"created_at"`
		Log       []*Operation `json:"log,omitempty"`
	}
	return json.Marshal(releaseJSON{
		ID:        r.ID,
		RepoID:    r.RepoID,
		Tag:       r.Tag,
		Title:     r.Title,
		Body:      r.Body,
		Author:    r.Author,
		CreatedAt: r.CreatedAt,
		Log:       r.log.Operations(),
	})
}

// UnmarshalJSON deserializes a Release and rebuilds its operation log
func (r *Release) UnmarshalJSON(data []byte) error {
	type releaseJSON struct {
		ID        string       `json:"id"`
		RepoID    string       `json:"repo_id"`
		Tag       string       `json:"tag"`
		Title     string       `json:"title"`
		Body      string       `json:"body"`
		Author    string       `json:"author"`
		CreatedAt time.Time    `json:"created_at"`
		Log       []*Operation `json:"log,omitempty"`
	}
	var snap releaseJSON
	if err := json.Unmarshal(data, &snap); err != nil {
		return err
	}
	r.ID = snap.ID
	r.RepoID = snap.RepoID
	r.Tag = snap.Tag
	r.Title = snap.Title
	r.Body = snap.Body
	r.Author = snap.Author
	r.CreatedAt = snap.CreatedAt
	r.log = NewOperationLog()
	for _, op := range snap.Log {
		r.log.Add(op)
	}
	return nil
}

// Log returns the operation log
func (r *Release) Log() *OperationLog {
	return r.log
}

// ReleaseStore manages releases using CRDT operations
type ReleaseStore struct {
	mu       sync.RWMutex
	releases map[string]map[string]*Release // repoID -> releaseID -> release
	path     string
	counter  uint64
}

// NewReleaseStore creates a new release store
func NewReleaseStore(path string) *ReleaseStore {
	return &ReleaseStore{
		releases: make(map[string]map[string]*Release),
		path:     path,
	}
}

// Load reads persisted releases from disk
func (s *ReleaseStore) Load() error {
	if s.path == "" {
		return nil
	}
	return persistence.LoadJSON(s.path, &s.releases)
}

// Save writes releases to disk
func (s *ReleaseStore) Save() error {
	if s.path == "" {
		return nil
	}
	return persistence.SaveJSON(s.path, s.releases)
}

// nextLamport returns the next Lamport timestamp
func (s *ReleaseStore) nextLamport() uint64 {
	s.counter++
	return s.counter
}

// Create creates a new release
func (s *ReleaseStore) Create(repoID, tag, title, body, author string) (*Release, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.releases[repoID]; !ok {
		s.releases[repoID] = make(map[string]*Release)
	}

	// Check for duplicate tag
	for _, r := range s.releases[repoID] {
		if r.Tag == tag {
			return nil, fmt.Errorf("release with tag %s already exists", tag)
		}
	}

	b := make([]byte, 8)
	_, _ = crypto_rand.Read(b)
	id := fmt.Sprintf("rel-%d-%x", time.Now().UnixNano(), b)
	now := time.Now()

	release := &Release{
		ID:        id,
		RepoID:    repoID,
		Tag:       tag,
		Title:     title,
		Body:      body,
		Author:    author,
		CreatedAt: now,
		log:       NewOperationLog(),
	}

	op := &Operation{
		ID:        id,
		Type:      OpCreate,
		Author:    author,
		Timestamp: now,
		Lamport:   s.nextLamport(),
		Data: map[string]interface{}{
			"tag":   tag,
			"title": title,
			"body":  body,
		},
	}
	release.log.Add(op)

	s.releases[repoID][id] = release
	return release, nil
}

// Get returns a specific release
func (s *ReleaseStore) Get(repoID, releaseID string) (*Release, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if _, ok := s.releases[repoID]; !ok {
		return nil, fmt.Errorf("repo not found: %s", repoID)
	}
	if release, ok := s.releases[repoID][releaseID]; ok {
		return release, nil
	}
	return nil, fmt.Errorf("release not found: %s", releaseID)
}

// List returns all releases for a repository, sorted by created_at descending
func (s *ReleaseStore) List(repoID string) []*Release {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*Release, 0)
	if _, ok := s.releases[repoID]; !ok {
		return result
	}

	for _, release := range s.releases[repoID] {
		result = append(result, release)
	}

	// Sort by created_at descending (newest first)
	sort.Slice(result, func(i, j int) bool {
		return result[i].CreatedAt.After(result[j].CreatedAt)
	})

	return result
}

// Delete removes a release
func (s *ReleaseStore) Delete(repoID, releaseID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.releases[repoID]; !ok {
		return fmt.Errorf("repo not found: %s", repoID)
	}
	if _, ok := s.releases[repoID][releaseID]; !ok {
		return fmt.Errorf("release not found: %s", releaseID)
	}

	delete(s.releases[repoID], releaseID)
	return nil
}
