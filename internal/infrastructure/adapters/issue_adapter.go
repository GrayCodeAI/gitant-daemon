// Package adapters provides implementations that map domain repository interfaces
// to concrete infrastructure implementations.
package adapters

import (
	"context"
	"fmt"

	"github.com/lakshmanpatel/gitant/internal/crdt"
	"github.com/lakshmanpatel/gitant/internal/domain/models"
	"github.com/lakshmanpatel/gitant/internal/domain/repositories"
)

// Compile-time check: IssueAdapter must implement IssueRepository.
var _ repositories.IssueRepository = (*IssueAdapter)(nil)

// IssueAdapter adapts the CRDT IssueStore to the domain IssueRepository interface.
type IssueAdapter struct {
	store *crdt.IssueStore
}

// NewIssueAdapter creates a new adapter.
func NewIssueAdapter(store *crdt.IssueStore) *IssueAdapter {
	return &IssueAdapter{store: store}
}

// Create implements repositories.IssueRepository.
func (a *IssueAdapter) Create(ctx context.Context, repoID, title, body string, labels []string, author string) (*models.Issue, error) {
	id := generateID("issue")
	issue := a.store.Create(repoID, id, author, title, body)
	issue.Labels = labels // TODO: add label support to CRDT
	return toDomainIssue(issue, repoID), nil
}

// Get implements repositories.IssueRepository.
func (a *IssueAdapter) Get(ctx context.Context, repoID, issueID string) (*models.Issue, error) {
	issue, err := a.store.Get(repoID, issueID)
	if err != nil {
		return nil, err
	}
	return toDomainIssue(issue, repoID), nil
}

// List implements repositories.IssueRepository.
func (a *IssueAdapter) List(ctx context.Context, repoID string, filters models.IssueFilters) ([]*models.Issue, int, error) {
	crIssues := a.store.List(repoID)
	var issues []*models.Issue
	for _, ci := range crIssues {
		issues = append(issues, toDomainIssue(ci, repoID))
	}
	return issues, len(issues), nil
}

// Update implements repositories.IssueRepository.
func (a *IssueAdapter) Update(ctx context.Context, repoID, issueID string, fn func(*models.Issue) error) error {
	return a.store.Update(repoID, issueID, func(ci *crdt.Issue) error {
		// Convert to domain model, call fn, then sync back
		di := toDomainIssue(ci, repoID)
		if err := fn(di); err != nil {
			return err
		}
		// Sync back (simplified - just update fields for now)
		ci.Title = di.Title
		ci.Body = di.Body
		ci.Labels = di.Labels
		return nil
	})
}

// Delete implements repositories.IssueRepository.
func (a *IssueAdapter) Delete(ctx context.Context, repoID, issueID string) error {
	return a.store.Delete(repoID, issueID)
}

// toDomainIssue converts a CRDT Issue to a domain model Issue.
func toDomainIssue(ci *crdt.Issue, repoID string) *models.Issue {
	return &models.Issue{
		ID:        ci.ID,
		RepoID:    repoID,
		Title:     ci.Title,
		Body:      ci.Body,
		Author:    ci.Author,
		Status:    string(ci.Status),
		Labels:    ci.Labels,
		CreatedAt: ci.CreatedAt,
		UpdatedAt: ci.UpdatedAt,
	}
}

// generateID creates an ID with the given prefix.
func generateID(prefix string) string {
	return fmt.Sprintf("%s-%d", prefix, len(prefix)+42)
}
