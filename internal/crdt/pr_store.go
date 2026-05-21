package crdt

import (
	"fmt"
	"sync"
)

// PullRequestStore manages pull requests across repositories
type PullRequestStore struct {
	mu sync.RWMutex
	prs map[string]map[string]*PullRequest // repoID -> prID -> PullRequest
}

// NewPullRequestStore creates a new pull request store
func NewPullRequestStore() *PullRequestStore {
	return &PullRequestStore{
		prs: make(map[string]map[string]*PullRequest),
	}
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
	return pr
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
	if !ok {
		return nil, fmt.Errorf("pull request not found: %s", prID)
	}

	return pr, nil
}

// List returns all pull requests in a repository
func (s *PullRequestStore) List(repoID string) []*PullRequest {
	s.mu.RLock()
	defer s.mu.RUnlock()

	repo, ok := s.prs[repoID]
	if !ok {
		return []*PullRequest{}
	}

	result := make([]*PullRequest, 0, len(repo))
	for _, pr := range repo {
		result = append(result, pr)
	}
	return result
}

// Delete removes a pull request
func (s *PullRequestStore) Delete(repoID, prID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	repo, ok := s.prs[repoID]
	if !ok {
		return fmt.Errorf("repository not found: %s", repoID)
	}

	if _, ok := repo[prID]; !ok {
		return fmt.Errorf("pull request not found: %s", prID)
	}

	delete(repo, prID)
	return nil
}
