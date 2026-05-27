package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	authMiddleware "github.com/lakshmanpatel/gitant/internal/api/middleware"
	"github.com/lakshmanpatel/gitant/internal/crdt"
	"github.com/lakshmanpatel/gitant/internal/webhooks"
)

// ListTasks lists tasks for a repository
func ListTasks(store *crdt.TaskStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		offset, limit := ParsePagination(r)
		repoID := chi.URLParam(r, "id")
		status := crdt.TaskStatus(r.URL.Query().Get("status"))

		tasks := store.List(repoID, status)

		result := make([]map[string]interface{}, len(tasks))
		for i, task := range tasks {
			result[i] = map[string]interface{}{
				"id":           task.ID,
				"repo":         task.RepoID,
				"title":        task.Title,
				"description":  task.Description,
				"status":       string(task.Status),
				"claimed_by":   task.ClaimedBy,
				"created_by":   task.CreatedBy,
				"created_at":   task.CreatedAt,
				"claimed_at":   task.ClaimedAt,
				"completed_at": task.CompletedAt,
				"result":       task.Result,
			}
		}

		paged, total := PaginateSlice(result, offset, limit)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"tasks":  paged,
			"total":  total,
			"offset": offset,
			"limit":  limit,
		})
	}
}

// CreateTask creates a new task
func CreateTask(store *crdt.TaskStore, wm *webhooks.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		repoID := chi.URLParam(r, "id")

		var req struct {
			Title       string `json:"title"`
			Description string `json:"description"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		if req.Title == "" {
			http.Error(w, "title is required", http.StatusBadRequest)
			return
		}

		author := authMiddleware.GetIdentity(r)
		if author == "" {
			author = "anonymous"
		}

		taskID := generateID("task")
		task := store.Create(repoID, taskID, author, req.Title, req.Description)
		if err := store.Save(); err != nil {
			slog.Error("failed to persist task", "error", err)
		}

		wm.Dispatch(webhooks.Event{
			Type: webhooks.EventTaskCreated,
			Repo: repoID,
			Data: map[string]interface{}{
				"task_id": taskID,
				"title":   req.Title,
				"author":  author,
			},
		})

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(task)
	}
}

// ClaimTask claims a task
func ClaimTask(store *crdt.TaskStore, wm *webhooks.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		repoID := chi.URLParam(r, "id")
		taskID := chi.URLParam(r, "taskId")

		author := authMiddleware.GetIdentity(r)
		if author == "" {
			author = "anonymous"
		}

		if err := store.Claim(repoID, taskID, author); err != nil {
			http.Error(w, SanitizeError(err, "failed to claim task"), http.StatusBadRequest)
			return
		}

		wm.Dispatch(webhooks.Event{
			Type: webhooks.EventTaskClaimed,
			Repo: repoID,
			Data: map[string]interface{}{
				"task_id": taskID,
				"author":  author,
			},
		})

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"id":      taskID,
			"status":  string(crdt.TaskClaimed),
		})
	}
}

// CompleteTask completes a task
func CompleteTask(store *crdt.TaskStore, wm *webhooks.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		repoID := chi.URLParam(r, "id")
		taskID := chi.URLParam(r, "taskId")

		var req struct {
			Result string `json:"result"`
		}
		if r.Body != nil {
			json.NewDecoder(r.Body).Decode(&req)
		}

		if err := store.Complete(repoID, taskID, req.Result); err != nil {
			http.Error(w, SanitizeError(err, "failed to complete task"), http.StatusBadRequest)
			return
		}

		wm.Dispatch(webhooks.Event{
			Type: webhooks.EventTaskCompleted,
			Repo: repoID,
			Data: map[string]interface{}{
				"task_id": taskID,
				"result":  req.Result,
			},
		})

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"id":      taskID,
			"status":  string(crdt.TaskCompleted),
		})
	}
}
