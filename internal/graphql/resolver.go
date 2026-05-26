package graphql

import (
	"context"
	"fmt"

	"github.com/lakshmanpatel/gitant/internal/crdt"
	"github.com/lakshmanpatel/gitant/internal/storage"
)

// Resolver is the root resolver
type Resolver struct {
	repos  *storage.RepositoryRegistry
	issues *crdt.IssueStore
	prs    *crdt.PullRequestStore
	tasks  *crdt.TaskStore
}

// NewResolver creates a new resolver
func NewResolver(
	repos *storage.RepositoryRegistry,
	issues *crdt.IssueStore,
	prs *crdt.PullRequestStore,
	tasks *crdt.TaskStore,
) *Resolver {
	return &Resolver{
		repos:  repos,
		issues: issues,
		prs:    prs,
		tasks:  tasks,
	}
}

// Query resolver
func (r *Resolver) Repos(ctx context.Context) ([]*storage.RepoEntry, error) {
	return r.repos.List(), nil
}

func (r *Resolver) Repo(ctx context.Context, id string) (*storage.RepoEntry, error) {
	entry, err := r.repos.GetEntry(id)
	if err != nil {
		return nil, fmt.Errorf("repo not found: %s", id)
	}
	return entry, nil
}

func (r *Resolver) Issues(ctx context.Context, repoID string) ([]*crdt.Issue, error) {
	return r.issues.List(repoID), nil
}

func (r *Resolver) Issue(ctx context.Context, repoID string, id string) (*crdt.Issue, error) {
	issue, err := r.issues.Get(repoID, id)
	if err != nil {
		return nil, fmt.Errorf("issue not found: %s", id)
	}
	return issue, nil
}

func (r *Resolver) PullRequests(ctx context.Context, repoID string) ([]*crdt.PullRequest, error) {
	return r.prs.List(repoID), nil
}

func (r *Resolver) PullRequest(ctx context.Context, repoID string, id string) (*crdt.PullRequest, error) {
	pr, err := r.prs.Get(repoID, id)
	if err != nil {
		return nil, fmt.Errorf("PR not found: %s", id)
	}
	return pr, nil
}

func (r *Resolver) Tasks(ctx context.Context, repoID string, status string) ([]*crdt.Task, error) {
	tasks := r.tasks.List(repoID, crdt.TaskStatus(status))
	result := make([]*crdt.Task, len(tasks))
	for i := range tasks {
		result[i] = &tasks[i]
	}
	return result, nil
}
