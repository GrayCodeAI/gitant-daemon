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
	ID         string    `json:"id"`
	RepoID     string    `json:"repo_id"`
	Tag        string    `json:"tag"`
	Title      string    `json:"title"`
	Body       string    `json:"body"`
	Author     string    `json:"author"` // DID
	CreatedAt  time.Time `json:"created_at"`
	Tombstoned bool      `json:"tombstoned,omitempty"`
	log        *OperationLog
}

// Tombstone marks a release as deleted. The tombstone operation will replicate
// to peers, ensuring the deletion propagates correctly.
func (r *Release) Tombstone(author string) {
	r.Tombstoned = true
	r.log.Add(&Operation{
		ID:        generateID(),
		Type:      OpTombstone,
		Author:    author,
		Timestamp: time.Now(),
		Lamport:   r.log.clock.Increment(),
	})
}

// MarshalJSON serializes a Release including its operation log
func (r *Release) MarshalJSON() ([]byte, error) {
	type releaseJSON struct {
		ID         string       `json:"id"`
		RepoID     string       `json:"repo_id"`
		Tag        string       `json:"tag"`
		Title      string       `json:"title"`
		Body       string       `json:"body"`
		Author     string       `json:"author"`
		CreatedAt  time.Time    `json:"created_at"`
		Tombstoned bool         `json:"tombstoned,omitempty"`
		Log        []*Operation `json:"log,omitempty"`
	}
	return json.Marshal(releaseJSON{
		ID:         r.ID,
		RepoID:     r.RepoID,
		Tag:        r.Tag,
		Title:      r.Title,
		Body:       r.Body,
		Author:     r.Author,
		CreatedAt:  r.CreatedAt,
		Tombstoned: r.Tombstoned,
		Log:        r.log.Operations(),
	})
}

// UnmarshalJSON deserializes a Release and rebuilds its operation log
func (r *Release) UnmarshalJSON(data []byte) error {
	type releaseJSON struct {
		ID         string       `json:"id"`
		RepoID     string       `json:"repo_id"`
		Tag        string       `json:"tag"`
		Title      string       `json:"title"`
		Body       string       `json:"body"`
		Author     string       `json:"author"`
		CreatedAt  time.Time    `json:"created_at"`
		Tombstoned bool         `json:"tombstoned,omitempty"`
		Log        []*Operation `json:"log,omitempty"`
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
	r.Tombstoned = snap.Tombstoned
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
	s.mu.RLock()
	data := make(map[string]map[string]*Release, len(s.releases))
	for repoID, repoReleases := range s.releases {
		repoCopy := make(map[string]*Release, len(repoReleases))
		for k, v := range repoReleases {
			copy := *v
			repoCopy[k] = &copy
		}
		data[repoID] = repoCopy
	}
	s.mu.RUnlock()
	return persistence.SaveJSON(s.path, data)
}

// saveLocked persists while the caller already holds the write lock.
func (s *ReleaseStore) saveLocked() error {
	if s.path == "" {
		return nil
	}
	return persistence.SaveJSON(s.path, s.releases)
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
		Lamport:   release.log.clock.Increment(),
		Data: map[string]interface{}{
			"tag":   tag,
			"title": title,
			"body":  body,
		},
	}
	release.log.Add(op)

	s.releases[repoID][id] = release
	copy := *release
	return &copy, nil
}

// Get returns a specific non-tombstoned release
func (s *ReleaseStore) Get(repoID, releaseID string) (*Release, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if _, ok := s.releases[repoID]; !ok {
		return nil, fmt.Errorf("repo not found: %s", repoID)
	}
	release, ok := s.releases[repoID][releaseID]
	if !ok || release.Tombstoned {
		return nil, fmt.Errorf("release not found: %s", releaseID)
	}
	copy := *release
	return &copy, nil
}

// List returns all non-tombstoned releases for a repository, sorted by created_at descending
func (s *ReleaseStore) List(repoID string) []*Release {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*Release, 0)
	if _, ok := s.releases[repoID]; !ok {
		return result
	}

	for _, release := range s.releases[repoID] {
		if release.Tombstoned {
			continue
		}
		copy := *release
		result = append(result, &copy)
	}

	// Sort by created_at descending (newest first)
	sort.Slice(result, func(i, j int) bool {
		return result[i].CreatedAt.After(result[j].CreatedAt)
	})

	return result
}

// Delete tombstones a release so the deletion replicates to peers.
func (s *ReleaseStore) Delete(repoID, releaseID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.releases[repoID]; !ok {
		return fmt.Errorf("repo not found: %s", repoID)
	}
	release, ok := s.releases[repoID][releaseID]
	if !ok {
		return fmt.Errorf("release not found: %s", releaseID)
	}

	release.Tombstone("system")
	return s.saveLocked()
}

// Merge merges another release's operations into this one
func (r *Release) Merge(other *Release) {
	r.log.clock.Merge(other.log.clock)

	existingIDs := make(map[string]bool)
	for _, op := range r.log.Operations() {
		existingIDs[op.ID] = true
	}
	for _, op := range other.log.Operations() {
		if !existingIDs[op.ID] {
			r.log.ImportOperation(op)
		}
	}

	allOps := make([]*Operation, len(r.log.Operations()))
	copy(allOps, r.log.Operations())
	SortOps(allOps)

	// Replay title/body from ops, and check for tombstone
	for _, op := range allOps {
		switch op.Type {
		case OpSetTitle:
			if title, ok := op.Data["title"].(string); ok {
				r.Title = title
			}
		case OpSetBody:
			if body, ok := op.Data["body"].(string); ok {
				r.Body = body
			}
		case OpTombstone:
			r.Tombstoned = true
		}
	}
}

// MergeRemote merges a remote release snapshot into the local store
func (s *ReleaseStore) MergeRemote(repoID string, remote *Release) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.releases[repoID]; !ok {
		s.releases[repoID] = make(map[string]*Release)
	}

	if local, ok := s.releases[repoID][remote.ID]; ok {
		local.Merge(remote)
	} else {
		cp := *remote
		s.releases[repoID][remote.ID] = &cp
	}
	return s.saveLocked()
}
