// Package service provides application services (use case orchestrators).
// These services depend on repository ports (interfaces) and orchestrate business logic.
package service

import (
	"github.com/lakshmanpatel/gitant/internal/application/ports"
	"github.com/lakshmanpatel/gitant/internal/domain/repositories"
)

// Factory creates and wires all application services.
// This will be populated in Phase 2 to instantiate concrete services with their dependencies.
type ServiceFactory struct {
	// Dependencies (interfaces, not concrete types)
	repoRepo    repositories.RepositoryRepository
	issueRepo   repositories.IssueRepository
	prRepo      repositories.PullRequestRepository
	labelRepo   repositories.LabelRepository
	taskRepo    repositories.TaskRepository
	releaseRepo repositories.ReleaseRepository
}

// NewServiceFactory creates a new factory with repository dependencies.
func NewServiceFactory(
	repoRepo repositories.RepositoryRepository,
	issueRepo repositories.IssueRepository,
	prRepo repositories.PullRequestRepository,
	labelRepo repositories.LabelRepository,
	taskRepo repositories.TaskRepository,
	releaseRepo repositories.ReleaseRepository,
) *ServiceFactory {
	return &ServiceFactory{
		repoRepo:    repoRepo,
		issueRepo:   issueRepo,
		prRepo:      prRepo,
		labelRepo:   labelRepo,
		taskRepo:    taskRepo,
		releaseRepo: releaseRepo,
	}
}

// CreateRepositoryService returns a service implementation.
func (f *ServiceFactory) CreateRepositoryService() ports.RepositoryService {
	return NewRepositoryService(f.repoRepo)
}

// CreateIssueService returns a service implementation.
func (f *ServiceFactory) CreateIssueService() ports.IssueService {
	if f.issueRepo == nil {
		return nil
	}
	return NewIssueService(f.issueRepo)
}

// CreatePRService returns a service implementation.
func (f *ServiceFactory) CreatePRService() ports.PRService {
	if f.prRepo == nil {
		return nil
	}
	return NewPRService(f.prRepo)
}

// CreateLabelService returns a service implementation.
func (f *ServiceFactory) CreateLabelService() ports.LabelService {
	if f.labelRepo == nil {
		return nil
	}
	return NewLabelService(f.labelRepo)
}

// CreateTaskService returns a service implementation.
func (f *ServiceFactory) CreateTaskService() ports.TaskService {
	if f.taskRepo == nil {
		return nil
	}
	return NewTaskService(f.taskRepo)
}

// CreateReleaseService returns a service implementation.
func (f *ServiceFactory) CreateReleaseService() ports.ReleaseService {
	if f.releaseRepo == nil {
		return nil
	}
	return NewReleaseService(f.releaseRepo)
}
