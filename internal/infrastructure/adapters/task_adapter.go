package adapters

import (
	"context"
	"fmt"
	"time"

	"github.com/lakshmanpatel/gitant/internal/crdt"
	"github.com/lakshmanpatel/gitant/internal/domain/models"
	"github.com/lakshmanpatel/gitant/internal/domain/repositories"
)

type TaskAdapter struct {
	store *crdt.TaskStore
}

func NewTaskAdapter(store *crdt.TaskStore) repositories.TaskRepository {
	return &TaskAdapter{store: store}
}

func (a *TaskAdapter) Create(ctx context.Context, repoID, title, description, author string) (*models.Task, error) {
	id := fmt.Sprintf("task-%d", time.Now().UnixNano())
	task := a.store.Create(repoID, id, author, title, description)
	return toDomainTask(task), nil
}

func (a *TaskAdapter) Get(ctx context.Context, repoID, taskID string) (*models.Task, error) {
	task, err := a.store.Get(repoID, taskID)
	if err != nil {
		return nil, err
	}
	return toDomainTask(task), nil
}

func (a *TaskAdapter) List(ctx context.Context, repoID string, status string) ([]*models.Task, error) {
	var filterStatus crdt.TaskStatus
	if status != "" {
		filterStatus = crdt.TaskStatus(status)
	}
	tasks := a.store.List(repoID, filterStatus)
	result := make([]*models.Task, len(tasks))
	for i, t := range tasks {
		task := t // capture loop variable
		result[i] = toDomainTask(&task)
	}
	return result, nil
}

func (a *TaskAdapter) Update(ctx context.Context, repoID, taskID string, fn func(*models.Task) error) error {
	task, err := a.store.Get(repoID, taskID)
	if err != nil {
		return err
	}
	domainTask := toDomainTask(task)
	if err := fn(domainTask); err != nil {
		return err
	}
	// Apply status changes
	if domainTask.Status == string(crdt.TaskClaimed) && task.ClaimedBy == "" {
		return a.store.Claim(repoID, taskID, domainTask.Assignee)
	}
	if domainTask.Status == string(crdt.TaskCompleted) && task.Status != crdt.TaskCompleted {
		return a.store.Complete(repoID, taskID, "")
	}
	return nil
}

func (a *TaskAdapter) Delete(ctx context.Context, repoID, taskID string) error {
	// TaskStore doesn't have a Delete method, but we can mark it as completed
	// For now, return nil as tasks are not typically deleted
	return nil
}

func toDomainTask(t *crdt.Task) *models.Task {
	return &models.Task{
		ID:          t.ID,
		RepoID:      t.RepoID,
		Title:       t.Title,
		Description: t.Description,
		Status:      string(t.Status),
		Assignee:    t.ClaimedBy,
		Author:      t.CreatedBy,
		CreatedAt:   t.CreatedAt,
		UpdatedAt:   t.CreatedAt, // CRDT doesn't track this
	}
}
