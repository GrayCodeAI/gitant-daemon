package adapters

import (
	"context"

	"github.com/lakshmanpatel/gitant/internal/crdt"
	"github.com/lakshmanpatel/gitant/internal/domain/models"
	"github.com/lakshmanpatel/gitant/internal/domain/repositories"
)

// Compile-time check.
var _ repositories.PullRequestRepository = (*PRAdapter)(nil)

// PRAdapter adapts the CRDT PullRequestStore to the domain PullRequestRepository interface.
type PRAdapter struct {
	store *crdt.PullRequestStore
}

// NewPRAdapter creates a new adapter.
func NewPRAdapter(store *crdt.PullRequestStore) *PRAdapter {
	return &PRAdapter{store: store}
}

// Create implements repositories.PullRequestRepository.
func (a *PRAdapter) Create(ctx context.Context, repoID, title, body, author, sourceBranch, targetBranch string) (*models.PullRequest, error) {
	id := generateID("pr")
	pr := a.store.Create(repoID, id, author, title, body, sourceBranch, targetBranch)
	return toDomainPR(pr, repoID), nil
}

// Get implements repositories.PullRequestRepository.
func (a *PRAdapter) Get(ctx context.Context, repoID, prID string) (*models.PullRequest, error) {
	pr, err := a.store.Get(repoID, prID)
	if err != nil {
		return nil, err
	}
	return toDomainPR(pr, repoID), nil
}

// List implements repositories.PullRequestRepository.
func (a *PRAdapter) List(ctx context.Context, repoID string, filters models.PRFilters) ([]*models.PullRequest, int, error) {
	crPRs := a.store.List(repoID)
	var prs []*models.PullRequest
	for _, pr := range crPRs {
		prs = append(prs, toDomainPR(pr, repoID))
	}
	return prs, len(prs), nil
}

// Update implements repositories.PullRequestRepository.
func (a *PRAdapter) Update(ctx context.Context, repoID, prID string, fn func(*models.PullRequest) error) error {
	return a.store.Update(repoID, prID, func(ci *crdt.PullRequest) error {
		di := toDomainPR(ci, repoID)
		if err := fn(di); err != nil {
			return err
		}
		ci.Title = di.Title
		ci.Body = di.Body
		return nil
	})
}

// Delete implements repositories.PullRequestRepository.
func (a *PRAdapter) Delete(ctx context.Context, repoID, prID string) error {
	return a.store.Delete(repoID, prID)
}

func toDomainPR(pr *crdt.PullRequest, repoID string) *models.PullRequest {
	return &models.PullRequest{
		ID:           pr.ID,
		RepoID:       repoID,
		Title:        pr.Title,
		Body:         pr.Body,
		Author:       pr.Author,
		SourceBranch: pr.SourceBranch,
		TargetBranch: pr.TargetBranch,
		Status:       string(pr.Status),
		CreatedAt:    pr.CreatedAt,
		UpdatedAt:    pr.UpdatedAt,
	}
}
