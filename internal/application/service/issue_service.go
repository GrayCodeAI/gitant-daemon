package service

import (
	"context"

	"github.com/lakshmanpatel/gitant/internal/application/ports"
	"github.com/lakshmanpatel/gitant/internal/domain/models"
	"github.com/lakshmanpatel/gitant/internal/domain/repositories"
)

var _ ports.IssueService = (*IssueServiceImpl)(nil)

type IssueServiceImpl struct {
	issueRepo repositories.IssueRepository
}

func NewIssueService(issueRepo repositories.IssueRepository) *IssueServiceImpl {
	return &IssueServiceImpl{issueRepo: issueRepo}
}

func (s *IssueServiceImpl) CreateIssue(ctx context.Context, req ports.CreateIssueRequest) (*models.Issue, error) {
	return s.issueRepo.Create(ctx, req.RepoID, req.Title, req.Body, req.Labels, req.Author)
}

func (s *IssueServiceImpl) GetIssue(ctx context.Context, repoID, issueID string) (*models.Issue, error) {
	return s.issueRepo.Get(ctx, repoID, issueID)
}

func (s *IssueServiceImpl) ListIssues(ctx context.Context, repoID string, filters models.IssueFilters) ([]*models.Issue, int, error) {
	return s.issueRepo.List(ctx, repoID, filters)
}

func (s *IssueServiceImpl) CloseIssue(ctx context.Context, repoID, issueID string) error {
	return s.issueRepo.Update(ctx, repoID, issueID, func(issue *models.Issue) error {
		issue.Status = "closed"
		return nil
	})
}

func (s *IssueServiceImpl) CommentIssue(ctx context.Context, repoID, issueID, comment string) error {
	// Comments are stored separately in the CRDT store; this is a no-op placeholder
	// until the comment subsystem is integrated into the domain layer.
	return nil
}
