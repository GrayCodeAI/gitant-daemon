package crdt

import (
	"fmt"
	"sync"

	"github.com/lakshmanpatel/gitant/internal/persistence"
)

// IssueStore manages issues across repositories
type IssueStore struct {
	mu     sync.RWMutex
	issues map[string]map[string]*Issue // repoID -> issueID -> Issue
	path   string                       // persistence file path
}

// NewIssueStore creates a new issue store
func NewIssueStore(path string) *IssueStore {
	return &IssueStore{
		issues: make(map[string]map[string]*Issue),
		path:   path,
	}
}

// Load reads persisted issues from disk
func (s *IssueStore) Load() error {
	if s.path == "" {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return persistence.LoadJSON(s.path, &s.issues)
}

// Save writes all issues to disk
func (s *IssueStore) Save() error {
	if s.path == "" {
		return nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	// Deep copy to avoid race with concurrent mutations
	data := make(map[string]map[string]*Issue, len(s.issues))
	for repoID, repoIssues := range s.issues {
		repoCopy := make(map[string]*Issue, len(repoIssues))
		for k, v := range repoIssues {
			issueCopy := *v
			repoCopy[k] = &issueCopy
		}
		data[repoID] = repoCopy
	}
	return persistence.SaveJSON(s.path, data)
}

// Create creates a new issue in a repository
func (s *IssueStore) Create(repoID, id, author, title, body string) *Issue {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.issues[repoID]; !ok {
		s.issues[repoID] = make(map[string]*Issue)
	}

	issue := NewIssue(id, author, title, body)
	s.issues[repoID][id] = issue
	return issue
}

// Get returns an issue by repo and issue ID
func (s *IssueStore) Get(repoID, issueID string) (*Issue, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	repo, ok := s.issues[repoID]
	if !ok {
		return nil, fmt.Errorf("repository not found: %s", repoID)
	}

	issue, ok := repo[issueID]
	if !ok {
		return nil, fmt.Errorf("issue not found: %s", issueID)
	}

	return issue, nil
}

// List returns all issues in a repository
func (s *IssueStore) List(repoID string) []*Issue {
	s.mu.RLock()
	defer s.mu.RUnlock()

	repo, ok := s.issues[repoID]
	if !ok {
		return []*Issue{}
	}

	result := make([]*Issue, 0, len(repo))
	for _, issue := range repo {
		result = append(result, issue)
	}
	return result
}

// Delete removes an issue
func (s *IssueStore) Delete(repoID, issueID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	repo, ok := s.issues[repoID]
	if !ok {
		return fmt.Errorf("repository not found: %s", repoID)
	}

	if _, ok := repo[issueID]; !ok {
		return fmt.Errorf("issue not found: %s", issueID)
	}

	delete(repo, issueID)
	return s.saveLocked()
}

// saveLocked persists while the caller already holds the write lock.
func (s *IssueStore) saveLocked() error {
	if s.path == "" {
		return nil
	}
	data := make(map[string]map[string]*Issue, len(s.issues))
	for repoID, repoIssues := range s.issues {
		repoCopy := make(map[string]*Issue, len(repoIssues))
		for k, v := range repoIssues {
			issueCopy := *v
			repoCopy[k] = &issueCopy
		}
		data[repoID] = repoCopy
	}
	return persistence.SaveJSON(s.path, data)
}

// SaveIssue persists after a mutation (convenience for handlers)
func (s *IssueStore) SaveIssue(repoID, issueID string) error {
	return s.Save()
}

// Update atomically gets an issue, calls fn while holding the write lock,
// then persists. fn receives the issue and may mutate it freely.
func (s *IssueStore) Update(repoID, issueID string, fn func(*Issue) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	repo, ok := s.issues[repoID]
	if !ok {
		return fmt.Errorf("repository not found: %s", repoID)
	}
	issue, ok := repo[issueID]
	if !ok {
		return fmt.Errorf("issue not found: %s", issueID)
	}
	if err := fn(issue); err != nil {
		return err
	}
	return s.saveLocked()
}

// MergeRemote merges a remote issue snapshot into the local store.
func (s *IssueStore) MergeRemote(repoID string, remote *Issue) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.issues[repoID]; !ok {
		s.issues[repoID] = make(map[string]*Issue)
	}

	if local, ok := s.issues[repoID][remote.ID]; ok {
		local.Merge(remote)
	} else {
		issueCopy := *remote
		s.issues[repoID][remote.ID] = &issueCopy
	}
	return s.saveLocked()
}

// All returns deep copies of all issues across all repositories
func (s *IssueStore) All() map[string]map[string]*Issue {
	s.mu.RLock()
	defer s.mu.RUnlock()
	data := make(map[string]map[string]*Issue, len(s.issues))
	for repoID, repoIssues := range s.issues {
		repoCopy := make(map[string]*Issue, len(repoIssues))
		for k, v := range repoIssues {
			issueCopy := *v
			repoCopy[k] = &issueCopy
		}
		data[repoID] = repoCopy
	}
	return data
}
