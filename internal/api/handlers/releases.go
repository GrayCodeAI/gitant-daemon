package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/lakshmanpatel/gitant/internal/crdt"
	"github.com/lakshmanpatel/gitant/internal/webhooks"
	"github.com/lakshmanpatel/gitant/internal/api/middleware"
)

// CreateRelease creates a new release
func CreateRelease(store *crdt.ReleaseStore, wm *webhooks.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		repoID := chi.URLParam(r, "id")

		var req struct {
			Tag   string `json:"tag"`
			Title string `json:"title"`
			Body  string `json:"body"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if req.Tag == "" {
			http.Error(w, "Tag is required", http.StatusBadRequest)
			return
		}

		if req.Title == "" {
			http.Error(w, "Title is required", http.StatusBadRequest)
			return
		}

		author := middleware.GetIdentity(r)
		if author == "" {
			author = "anonymous"
		}

		release, err := store.Create(repoID, req.Tag, req.Title, req.Body, author)
		if err != nil {
			http.Error(w, SanitizeError(err, "failed to create release"), http.StatusConflict)
			return
		}

		// Dispatch webhook
		wm.Dispatch(webhooks.Event{
			Type: "release.created",
			Repo: repoID,
			Data: map[string]interface{}{
				"release_id": release.ID,
				"tag":        release.Tag,
				"title":      release.Title,
			},
		})

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(release)
	}
}

// ListReleases lists all releases for a repository
func ListReleases(store *crdt.ReleaseStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		offset, limit := ParsePagination(r)
		repoID := chi.URLParam(r, "id")

		releases := store.List(repoID)

		result := make([]map[string]interface{}, 0, len(releases))
		for _, rel := range releases {
			result = append(result, map[string]interface{}{
				"id":         rel.ID,
				"repo_id":    rel.RepoID,
				"tag":        rel.Tag,
				"title":      rel.Title,
				"body":       rel.Body,
				"author":     rel.Author,
				"created_at": rel.CreatedAt,
			})
		}

		paged, total := PaginateSlice(result, offset, limit)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"releases": paged,
			"total":    total,
			"offset":   offset,
			"limit":    limit,
		})
	}
}

// GetRelease returns a specific release
func GetRelease(store *crdt.ReleaseStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		repoID := chi.URLParam(r, "id")
		releaseID := chi.URLParam(r, "releaseId")

		release, err := store.Get(repoID, releaseID)
		if err != nil {
			http.Error(w, SanitizeError(err, "release not found"), http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(release)
	}
}

// DeleteRelease deletes a release
func DeleteRelease(store *crdt.ReleaseStore, wm *webhooks.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		repoID := chi.URLParam(r, "id")
		releaseID := chi.URLParam(r, "releaseId")

		if err := store.Delete(repoID, releaseID); err != nil {
			http.Error(w, SanitizeError(err, "release not found"), http.StatusNotFound)
			return
		}

		wm.Dispatch(webhooks.Event{
			Type: "release.deleted",
			Repo: repoID,
			Data: map[string]interface{}{
				"release_id": releaseID,
			},
		})

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"deleted": true,
			"id":      releaseID,
		})
	}
}
