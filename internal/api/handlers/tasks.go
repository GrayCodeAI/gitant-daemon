package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/lakshmanpatel/gitant/internal/crdt"
)

// ListTasks lists tasks for a repository
func ListTasks(store *crdt.TaskStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		repoID := chi.URLParam(r, "id")
		status := crdt.TaskStatus(r.URL.Query().Get("status"))

		tasks := store.List(repoID, status)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"tasks": tasks,
			"total": len(tasks),
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

		author := "anonymous"
		if did, ok := r.Context().Value("identity").(string); ok && did != "" {
			author = did
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

		author := "anonymous"
		if did, ok := r.Context().Value("identity").(string); ok && did != "" {
			author = did
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
