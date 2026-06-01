// Package repositories defines the interface contracts for all data persistence.
// These are the "ports" in the hexagonal architecture — the domain defines what it needs,
// and infrastructure provides implementations.
package repositories

import (
	"context"

	"github.com/lakshmanpatel/gitant/internal/domain/models"
)

// RepositoryRepository defines the contract for repository persistence.
type RepositoryRepository interface {
	// Create a new repository.
	Create(ctx context.Context, id, name, description string, private bool, owner string) (*models.Repository, error)
	// Get retrieves a repository by ID.
	Get(ctx context.Context, id string) (*models.Repository, error)
	// List repositories with optional filters.
	List(ctx context.Context, filters models.RepositoryFilters) ([]*models.Repository, int, error)
	// Delete a repository.
	Delete(ctx context.Context, id string) error
	// Star a repository for a user.
	Star(ctx context.Context, repoID, did string) error
	// Unstar a repository for a user.
	Unstar(ctx context.Context, repoID, did string) error
	// Fork a repository.
	Fork(ctx context.Context, sourceID, forkID, owner string) (*models.Repository, error)
}

// IssueRepository defines the contract for issue persistence.
type IssueRepository interface {
	Create(ctx context.Context, repoID, title, body string, labels []string, author string) (*models.Issue, error)
	Get(ctx context.Context, repoID, issueID string) (*models.Issue, error)
	List(ctx context.Context, repoID string, filters models.IssueFilters) ([]*models.Issue, int, error)
	Update(ctx context.Context, repoID, issueID string, fn func(*models.Issue) error) error
	Delete(ctx context.Context, repoID, issueID string) error
}

// PullRequestRepository defines the contract for pull request persistence.
type PullRequestRepository interface {
	Create(ctx context.Context, repoID, title, body, author, sourceBranch, targetBranch string) (*models.PullRequest, error)
	Get(ctx context.Context, repoID, prID string) (*models.PullRequest, error)
	List(ctx context.Context, repoID string, filters models.PRFilters) ([]*models.PullRequest, int, error)
	Update(ctx context.Context, repoID, prID string, fn func(*models.PullRequest) error) error
	Delete(ctx context.Context, repoID, prID string) error
}

// LabelRepository defines the contract for label persistence.
type LabelRepository interface {
	Create(ctx context.Context, repoID, name, color string) (*models.Label, error)
	Get(ctx context.Context, repoID, labelID string) (*models.Label, error)
	List(ctx context.Context, repoID string) ([]*models.Label, error)
	Delete(ctx context.Context, repoID, labelID string) error
}

// TaskRepository defines the contract for task persistence.
type TaskRepository interface {
	Create(ctx context.Context, repoID, title, description, author string) (*models.Task, error)
	Get(ctx context.Context, repoID, taskID string) (*models.Task, error)
	List(ctx context.Context, repoID string, status string) ([]*models.Task, error)
	Update(ctx context.Context, repoID, taskID string, fn func(*models.Task) error) error
	Delete(ctx context.Context, repoID, taskID string) error
}

// ReleaseRepository defines the contract for release persistence.
type ReleaseRepository interface {
	Create(ctx context.Context, repoID, tagName, name, body, author string, draft, prerelease bool) (*models.Release, error)
	Get(ctx context.Context, repoID, releaseID string) (*models.Release, error)
	List(ctx context.Context, repoID string) ([]*models.Release, error)
	Delete(ctx context.Context, repoID, releaseID string) error
}
