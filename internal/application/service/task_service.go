package service

import (
	"context"
	"fmt"

	"github.com/lakshmanpatel/gitant/internal/application/ports"
	"github.com/lakshmanpatel/gitant/internal/domain/models"
	"github.com/lakshmanpatel/gitant/internal/domain/repositories"
)

var _ ports.TaskService = (*TaskServiceImpl)(nil)

type TaskServiceImpl struct {
	taskRepo repositories.TaskRepository
}

func NewTaskService(taskRepo repositories.TaskRepository) *TaskServiceImpl {
	return &TaskServiceImpl{taskRepo: taskRepo}
}

func (s *TaskServiceImpl) CreateTask(ctx context.Context, req ports.CreateTaskRequest) (*models.Task, error) {
	return s.taskRepo.Create(ctx, req.RepoID, req.Title, req.Description, req.Author)
}

func (s *TaskServiceImpl) GetTask(ctx context.Context, repoID, taskID string) (*models.Task, error) {
	return s.taskRepo.Get(ctx, repoID, taskID)
}

func (s *TaskServiceImpl) ListTasks(ctx context.Context, repoID, status string) ([]*models.Task, error) {
	return s.taskRepo.List(ctx, repoID, status)
}

func (s *TaskServiceImpl) ClaimTask(ctx context.Context, repoID, taskID, did string) error {
	return s.taskRepo.Update(ctx, repoID, taskID, func(task *models.Task) error {
		if task.Status != "open" {
			return fmt.Errorf("task is not open")
		}
		task.Status = "claimed"
		task.Assignee = did
		return nil
	})
}

func (s *TaskServiceImpl) CompleteTask(ctx context.Context, repoID, taskID string) error {
	return s.taskRepo.Update(ctx, repoID, taskID, func(task *models.Task) error {
		task.Status = "completed"
		return nil
	})
}
