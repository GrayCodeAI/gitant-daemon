package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	authMiddleware "github.com/lakshmanpatel/gitant/internal/api/middleware"
	"github.com/lakshmanpatel/gitant/internal/crdt"
)

// ListTasks lists tasks for a repository
func ListTasks(store *crdt.TaskStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		offset, limit := ParsePagination(r)
		repoID := chi.URLParam(r, "id")
		status := crdt.TaskStatus(r.URL.Query().Get("status"))

		tasks := store.List(repoID, status)

		// Convert to map slice for pagination
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
func CreateTask(store *crdt.TaskStore) http.HandlerFunc {
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

		taskID := fmt.Sprintf("task-%d", time.Now().UnixNano())
		task := store.Create(repoID, taskID, author, req.Title, req.Description)
		_ = store.Save()

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(task)
	}
}

// ClaimTask claims a task
func ClaimTask(store *crdt.TaskStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		repoID := chi.URLParam(r, "id")
		taskID := chi.URLParam(r, "taskId")

		author := authMiddleware.GetIdentity(r)
		if author == "" {
			author = "anonymous"
		}

		if err := store.Claim(repoID, taskID, author); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"id":      taskID,
			"status":  string(crdt.TaskClaimed),
		})
	}
}

// CompleteTask completes a task
func CompleteTask(store *crdt.TaskStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		repoID := chi.URLParam(r, "id")
		taskID := chi.URLParam(r, "taskId")

		var req struct {
			Result string `json:"result"`
		}
		json.NewDecoder(r.Body).Decode(&req)

		if err := store.Complete(repoID, taskID, req.Result); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"id":      taskID,
			"status":  string(crdt.TaskCompleted),
		})
	}
}
