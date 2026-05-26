package graphql

import (
	"context"
	"fmt"
	"time"

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

// Repository resolver
type repoResolver struct {
	*Resolver
}

func (r *repoResolver) ID(ctx context.Context, obj *storage.RepoEntry) (string, error) {
	return obj.ID, nil
}

func (r *repoResolver) Name(ctx context.Context, obj *storage.RepoEntry) (string, error) {
	return obj.Name, nil
}

func (r *repoResolver) Description(ctx context.Context, obj *storage.RepoEntry) (string, error) {
	return obj.Description, nil
}

func (r *repoResolver) Private(ctx context.Context, obj *storage.RepoEntry) (bool, error) {
	return obj.Private, nil
}

func (r *repoResolver) CreatedAt(ctx context.Context, obj *storage.RepoEntry) (string, error) {
	return obj.CreatedAt, nil
}

func (r *repoResolver) Issues(ctx context.Context, obj *storage.RepoEntry) ([]*crdt.Issue, error) {
	issues := r.issues.List(obj.ID)
	result := make([]*crdt.Issue, len(issues))
	copy(result, issues)
	return result, nil
}

func (r *repoResolver) PullRequests(ctx context.Context, obj *storage.RepoEntry) ([]*crdt.PullRequest, error) {
	prs := r.prs.List(obj.ID)
	result := make([]*crdt.PullRequest, len(prs))
	copy(result, prs)
	return result, nil
}

// Issue resolver
type issueResolver struct {
	*Resolver
}

func (r *issueResolver) ID(ctx context.Context, obj *crdt.Issue) (string, error) {
	return obj.ID, nil
}

func (r *issueResolver) Title(ctx context.Context, obj *crdt.Issue) (string, error) {
	return obj.Title, nil
}

func (r *issueResolver) Body(ctx context.Context, obj *crdt.Issue) (string, error) {
	return obj.Body, nil
}

func (r *issueResolver) Status(ctx context.Context, obj *crdt.Issue) (string, error) {
	return string(obj.Status), nil
}

func (r *issueResolver) Author(ctx context.Context, obj *crdt.Issue) (string, error) {
	return obj.Author, nil
}

func (r *issueResolver) Labels(ctx context.Context, obj *crdt.Issue) ([]string, error) {
	return obj.Labels, nil
}

func (r *issueResolver) CreatedAt(ctx context.Context, obj *crdt.Issue) (string, error) {
	return obj.CreatedAt.Format(time.RFC3339), nil
}

// PullRequest resolver
type prResolver struct {
	*Resolver
}

func (r *prResolver) ID(ctx context.Context, obj *crdt.PullRequest) (string, error) {
	return obj.ID, nil
}

func (r *prResolver) Title(ctx context.Context, obj *crdt.PullRequest) (string, error) {
	return obj.Title, nil
}

func (r *prResolver) Status(ctx context.Context, obj *crdt.PullRequest) (string, error) {
	return string(obj.Status), nil
}

func (r *prResolver) Author(ctx context.Context, obj *crdt.PullRequest) (string, error) {
	return obj.Author, nil
}

func (r *prResolver) SourceBranch(ctx context.Context, obj *crdt.PullRequest) (string, error) {
	return obj.SourceBranch, nil
}

func (r *prResolver) TargetBranch(ctx context.Context, obj *crdt.PullRequest) (string, error) {
	return obj.TargetBranch, nil
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
