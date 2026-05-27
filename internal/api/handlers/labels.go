package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	authMiddleware "github.com/lakshmanpatel/gitant/internal/api/middleware"
	"github.com/lakshmanpatel/gitant/internal/crdt"
	"github.com/lakshmanpatel/gitant/internal/webhooks"
)

// ListLabels lists all labels for a repository
func ListLabels(store *crdt.LabelStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		offset, limit := ParsePagination(r)
		repoID := chi.URLParam(r, "id")
		labels := store.List(repoID)

		paged, total := PaginateSlice(labels, offset, limit)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"labels": paged,
			"total":  total,
			"offset": offset,
			"limit":  limit,
		})
	}
}

// CreateLabel creates a new label for a repository
func CreateLabel(store *crdt.LabelStore, wm *webhooks.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		repoID := chi.URLParam(r, "id")

		var req struct {
			Name  string `json:"name"`
			Color string `json:"color"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		if req.Name == "" {
			http.Error(w, "name is required", http.StatusBadRequest)
			return
		}

		author := authMiddleware.GetIdentity(r)
		if author == "" {
			author = "anonymous"
		}

		if err := store.Add(repoID, req.Name, req.Color, author); err != nil {
			http.Error(w, SanitizeError(err, "failed to create label"), http.StatusConflict)
			return
		}

		wm.Dispatch(webhooks.Event{
			Type: webhooks.EventLabelCreated,
			Repo: repoID,
			Data: map[string]interface{}{
				"label":  req.Name,
				"author": author,
			},
		})

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"name":  req.Name,
			"color": req.Color,
		})
	}
}

// DeleteLabel deletes a label from a repository
func DeleteLabel(store *crdt.LabelStore, wm *webhooks.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		repoID := chi.URLParam(r, "id")
		name := chi.URLParam(r, "name")

		author := authMiddleware.GetIdentity(r)
		if author == "" {
			author = "anonymous"
		}

		if err := store.Remove(repoID, name, author); err != nil {
			http.Error(w, SanitizeError(err, "label not found"), http.StatusNotFound)
			return
		}

		wm.Dispatch(webhooks.Event{
			Type: webhooks.EventLabelDeleted,
			Repo: repoID,
			Data: map[string]interface{}{
				"label":  name,
				"author": author,
			},
		})

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
		})
	}
}
