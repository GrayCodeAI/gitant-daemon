package crdt

import (
	"fmt"
	"sync"

	"github.com/lakshmanpatel/gitant/internal/persistence"
)

// PullRequestStore manages pull requests across repositories
type PullRequestStore struct {
	mu  sync.RWMutex
	prs map[string]map[string]*PullRequest // repoID -> prID -> PullRequest
	path string                            // persistence file path
}

// NewPullRequestStore creates a new pull request store
func NewPullRequestStore(path string) *PullRequestStore {
	return &PullRequestStore{
		prs:  make(map[string]map[string]*PullRequest),
		path: path,
	}
}

// Load reads persisted PRs from disk
func (s *PullRequestStore) Load() error {
	if s.path == "" {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return persistence.LoadJSON(s.path, &s.prs)
}

// Save writes all PRs to disk
func (s *PullRequestStore) Save() error {
	if s.path == "" {
		return nil
	}
	s.mu.RLock()
	data := make(map[string]map[string]*PullRequest, len(s.prs))
	for repoID, repoPRs := range s.prs {
		repoCopy := make(map[string]*PullRequest, len(repoPRs))
		for k, v := range repoPRs {
			prCopy := *v
			repoCopy[k] = &prCopy
		}
		data[repoID] = repoCopy
	}
	s.mu.RUnlock()
	return persistence.SaveJSON(s.path, data)
}

// Create creates a new pull request in a repository
func (s *PullRequestStore) Create(repoID, id, author, title, body, sourceBranch, targetBranch string) *PullRequest {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.prs[repoID]; !ok {
		s.prs[repoID] = make(map[string]*PullRequest)
	}

	pr := NewPullRequest(id, author, title, body, sourceBranch, targetBranch)
	s.prs[repoID][id] = pr
	copy := *pr
	return &copy
}

// Get returns a pull request by repo and PR ID
func (s *PullRequestStore) Get(repoID, prID string) (*PullRequest, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	repo, ok := s.prs[repoID]
	if !ok {
		return nil, fmt.Errorf("repository not found: %s", repoID)
	}

	pr, ok := repo[prID]
	if !ok || pr.Tombstoned {
		return nil, fmt.Errorf("pull request not found: %s", prID)
	}

	copy := *pr
	return &copy, nil
}

// List returns all non-tombstoned pull requests in a repository
func (s *PullRequestStore) List(repoID string) []*PullRequest {
	s.mu.RLock()
	defer s.mu.RUnlock()

	repo, ok := s.prs[repoID]
	if !ok {
		return []*PullRequest{}
	}

	result := make([]*PullRequest, 0, len(repo))
	for _, pr := range repo {
		if pr.Tombstoned {
			continue
		}
		copy := *pr
		result = append(result, &copy)
	}
	return result
}

// Delete tombstones a pull request
func (s *PullRequestStore) Delete(repoID, prID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	repo, ok := s.prs[repoID]
	if !ok {
		return fmt.Errorf("repository not found: %s", repoID)
	}

	pr, ok := repo[prID]
	if !ok {
		return fmt.Errorf("pull request not found: %s", prID)
	}

	pr.Tombstone("system")
	return s.saveLocked()
}

// saveLocked persists while the caller already holds the write lock.
func (s *PullRequestStore) saveLocked() error {
	if s.path == "" {
		return nil
	}
	data := make(map[string]map[string]*PullRequest, len(s.prs))
	for repoID, repoPRs := range s.prs {
		repoCopy := make(map[string]*PullRequest, len(repoPRs))
		for k, v := range repoPRs {
			prCopy := *v
			repoCopy[k] = &prCopy
		}
		data[repoID] = repoCopy
	}
	return persistence.SaveJSON(s.path, data)
}

// SavePR persists after a mutation (convenience for handlers)
func (s *PullRequestStore) SavePR(repoID, prID string) error {
	return s.Save()
}

// Update atomically gets a PR, calls fn while holding the write lock,
// then persists. fn receives the PR and may mutate it freely.
func (s *PullRequestStore) Update(repoID, prID string, fn func(*PullRequest) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	repo, ok := s.prs[repoID]
	if !ok {
		return fmt.Errorf("repository not found: %s", repoID)
	}
	pr, ok := repo[prID]
	if !ok {
		return fmt.Errorf("pull request not found: %s", prID)
	}
	if err := fn(pr); err != nil {
		return err
	}
	return s.saveLocked()
}

// MergeRemote merges a remote pull request snapshot into the local store.
func (s *PullRequestStore) MergeRemote(repoID string, remote *PullRequest) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.prs[repoID]; !ok {
		s.prs[repoID] = make(map[string]*PullRequest)
	}

	if local, ok := s.prs[repoID][remote.ID]; ok {
		local.Merge(remote)
	} else {
		prCopy := *remote
		s.prs[repoID][remote.ID] = &prCopy
	}
	return s.saveLocked()
}

// All returns deep copies of all pull requests across all repositories
func (s *PullRequestStore) All() map[string]map[string]*PullRequest {
	s.mu.RLock()
	defer s.mu.RUnlock()
	data := make(map[string]map[string]*PullRequest, len(s.prs))
	for repoID, repoPRs := range s.prs {
		repoCopy := make(map[string]*PullRequest, len(repoPRs))
		for k, v := range repoPRs {
			prCopy := *v
			repoCopy[k] = &prCopy
		}
		data[repoID] = repoCopy
	}
	return data
}
