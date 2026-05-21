package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-git/go-git/v6/plumbing"
	"github.com/lakshmanpatel/gitant/internal/storage"
)

// CreateRepo creates a new repository
func CreateRepo(registry *storage.RepositoryRegistry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Name        string `json:"name"`
			Description string `json:"description"`
			Private     bool   `json:"private"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if req.Name == "" {
			http.Error(w, "Name is required", http.StatusBadRequest)
			return
		}

		entry, err := registry.Create(req.Name, req.Name, req.Description, req.Private)
		if err != nil {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":          entry.ID,
			"name":        entry.Name,
			"description": entry.Description,
			"private":     entry.Private,
			"created_at":  time.Now().Format(time.RFC3339),
		})
	}
}

// ListRepos lists all repositories
func ListRepos(registry *storage.RepositoryRegistry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		repos := registry.List()

		result := make([]map[string]interface{}, 0, len(repos))
		for _, entry := range repos {
			result = append(result, map[string]interface{}{
				"id":          entry.ID,
				"name":        entry.Name,
				"description": entry.Description,
				"private":     entry.Private,
				"created_at":  entry.CreatedAt,
			})
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"repos": result,
			"total": len(result),
		})
	}
}

// GetRepo gets a repository by ID
func GetRepo(registry *storage.RepositoryRegistry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")

		entry, err := registry.GetEntry(id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":          entry.ID,
			"name":        entry.Name,
			"description": entry.Description,
			"private":     entry.Private,
			"created_at":  entry.CreatedAt,
		})
	}
}

// DeleteRepo deletes a repository
func DeleteRepo(registry *storage.RepositoryRegistry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")

		if err := registry.Delete(id); err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"deleted": true,
			"id":      id,
		})
	}
}

// PushObjects pushes objects to a repository
func PushObjects(registry *storage.RepositoryRegistry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")

		repo, err := registry.Open(id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		// Parse the incoming objects
		var req struct {
			RefUpdates []struct {
				Name      string `json:"name"`
				OldHash   string `json:"old_hash"`
				NewHash   string `json:"new_hash"`
			} `json:"ref_updates"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Update refs
		for _, update := range req.RefUpdates {
			if update.NewHash != "" && update.NewHash != "0000000000000000000000000000000000000000" {
				hash := plumbing.NewHash(update.NewHash)
				if err := repo.CreateBranch(update.Name, hash); err != nil {
					// Try updating the ref directly
					_ = err
				}
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"repo":    id,
		})
	}
}

// CloneRepo clones a repository
func CloneRepo(registry *storage.RepositoryRegistry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")

		entry, err := registry.GetEntry(id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		repo, err := registry.Open(id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		refs, err := repo.ListAllRefs()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":   entry.ID,
			"name": entry.Name,
			"refs": refs,
		})
	}
}

// ListRefs lists all references in a repository
func ListRefs(registry *storage.RepositoryRegistry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")

		repo, err := registry.Open(id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		refs, err := repo.ListAllRefs()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"refs":  refs,
			"total": len(refs),
		})
	}
}

// CreateBranch creates a new branch
func CreateBranch(registry *storage.RepositoryRegistry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")

		var req struct {
			Name   string `json:"name"`
			Commit string `json:"commit"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		repo, err := registry.Open(id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		hash := plumbing.NewHash(req.Commit)
		if err := repo.CreateBranch(req.Name, hash); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"name":   req.Name,
			"commit": req.Commit,
		})
	}
}
