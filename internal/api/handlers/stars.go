package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/lakshmanpatel/gitant/internal/storage"
)

// StarRepo stars a repository
func StarRepo(registry *storage.RepositoryRegistry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		repoID := chi.URLParam(r, "id")

		did := "anonymous"
		if d, ok := r.Context().Value("identity").(string); ok && d != "" {
			did = d
		}

		if err := registry.Star(repoID, did); err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		entry, _ := registry.GetEntry(repoID)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"stars":   entry.Stars,
		})
	}
}

// UnstarRepo unstars a repository
func UnstarRepo(registry *storage.RepositoryRegistry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		repoID := chi.URLParam(r, "id")

		did := "anonymous"
		if d, ok := r.Context().Value("identity").(string); ok && d != "" {
			did = d
		}

		if err := registry.Unstar(repoID, did); err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		entry, _ := registry.GetEntry(repoID)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"stars":   entry.Stars,
		})
	}
}

// GetStarCount returns the star count for a repository
func GetStarCount(registry *storage.RepositoryRegistry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		repoID := chi.URLParam(r, "id")

		entry, err := registry.GetEntry(repoID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"stars":     entry.Stars,
			"starred_by": entry.StarredBy,
		})
	}
}
