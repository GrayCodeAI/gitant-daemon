// Package ports defines the application service interfaces (input ports).
// These are the "use cases" that the transport layer (HTTP handlers) depends on.
package ports

import (
	"context"

	"github.com/lakshmanpatel/gitant/internal/domain/models"
)

// RepositoryService defines the application use cases for repository management.
type RepositoryService interface {
	CreateRepository(ctx context.Context, req CreateRepoRequest) (*models.Repository, error)
	GetRepository(ctx context.Context, id string) (*models.Repository, error)
	ListRepositories(ctx context.Context, filters models.RepositoryFilters) ([]*models.Repository, int, error)
	DeleteRepository(ctx context.Context, id string) error
	StarRepository(ctx context.Context, repoID, did string) error
	UnstarRepository(ctx context.Context, repoID, did string) error
	ForkRepository(ctx context.Context, req ForkRepoRequest) (*models.Repository, error)
}

// IssueService defines the application use cases for issue management.
type IssueService interface {
	CreateIssue(ctx context.Context, req CreateIssueRequest) (*models.Issue, error)
	GetIssue(ctx context.Context, repoID, issueID string) (*models.Issue, error)
	ListIssues(ctx context.Context, repoID string, filters models.IssueFilters) ([]*models.Issue, int, error)
	CloseIssue(ctx context.Context, repoID, issueID string) error
	CommentIssue(ctx context.Context, repoID, issueID, comment string) error
}

// PRService defines the application use cases for pull request management.
type PRService interface {
	CreatePR(ctx context.Context, req CreatePRRequest) (*models.PullRequest, error)
	GetPR(ctx context.Context, repoID, prID string) (*models.PullRequest, error)
	ListPRs(ctx context.Context, repoID string, filters models.PRFilters) ([]*models.PullRequest, int, error)
	MergePR(ctx context.Context, repoID, prID string) error
}

// LabelService defines the application use cases for label management.
type LabelService interface {
	CreateLabel(ctx context.Context, req CreateLabelRequest) (*models.Label, error)
	ListLabels(ctx context.Context, repoID string) ([]*models.Label, error)
	DeleteLabel(ctx context.Context, repoID, labelID string) error
}

// TaskService defines the application use cases for task management.
type TaskService interface {
	CreateTask(ctx context.Context, req CreateTaskRequest) (*models.Task, error)
	GetTask(ctx context.Context, repoID, taskID string) (*models.Task, error)
	ListTasks(ctx context.Context, repoID, status string) ([]*models.Task, error)
	ClaimTask(ctx context.Context, repoID, taskID, did string) error
	CompleteTask(ctx context.Context, repoID, taskID string) error
}

// ReleaseService defines the application use cases for release management.
type ReleaseService interface {
	CreateRelease(ctx context.Context, req CreateReleaseRequest) (*models.Release, error)
	GetRelease(ctx context.Context, repoID, releaseID string) (*models.Release, error)
	ListReleases(ctx context.Context, repoID string) ([]*models.Release, error)
}

// AgentService defines the application use cases for agent management.
type AgentService interface {
	GenerateDID(ctx context.Context) (string, error)
	GetIdentity(ctx context.Context, did string) (string, error)
}

// SearchService defines the application use cases for code search.
type SearchService interface {
	SearchCode(ctx context.Context, repoID, query, ref string, offset, limit int) ([]models.SearchResult, int, error)
}

// Request DTOs (simple structs for service method inputs)

type CreateRepoRequest struct {
	Name        string
	Description string
	Private     bool
	Owner       string
}

type ForkRepoRequest struct {
	SourceID    string
	ForkName    string
	Owner       string
}

type CreateIssueRequest struct {
	RepoID      string
	Title       string
	Body        string
	Labels      []string
	Author      string
}

type CreatePRRequest struct {
	RepoID        string
	Title         string
	Body          string
	Author        string
	SourceBranch  string
	TargetBranch  string
}

type CreateLabelRequest struct {
	RepoID string
	Name   string
	Color  string
}

type CreateTaskRequest struct {
	RepoID      string
	Title       string
	Description string
	Author      string
}

type CreateReleaseRequest struct {
	RepoID      string
	TagName     string
	Name        string
	Body        string
	Author      string
	Draft       bool
	Prerelease  bool
}
