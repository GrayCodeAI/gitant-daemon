package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/lakshmanpatel/gitant/internal/storage"
)

// GetProtection returns protection rules for a branch
func GetProtection(store *storage.ProtectionStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		repoID := chi.URLParam(r, "id")
		branch := chi.URLParam(r, "branch")

		protection := store.Get(repoID, branch)
		if protection == nil {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"branch":           branch,
				"require_pr":       false,
				"require_approval": false,
				"no_force_push":    false,
				"protected":        false,
			})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"branch":           protection.Branch,
			"require_pr":       protection.RequirePR,
			"require_approval": protection.RequireApproval,
			"no_force_push":    protection.NoForcePush,
			"protected":        true,
		})
	}
}

// ListProtections returns all protection rules for a repo
func ListProtections(store *storage.ProtectionStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		offset, limit := ParsePagination(r)
		repoID := chi.URLParam(r, "id")

		protections := store.List(repoID)

		result := make([]map[string]interface{}, 0, len(protections))
		for _, p := range protections {
			result = append(result, map[string]interface{}{
				"branch":           p.Branch,
				"require_pr":       p.RequirePR,
				"require_approval": p.RequireApproval,
				"no_force_push":    p.NoForcePush,
			})
		}

		paged, total := PaginateSlice(result, offset, limit)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"protections": paged,
			"total":       total,
			"offset":      offset,
			"limit":       limit,
		})
	}
}

// SetProtection creates or updates protection rules for a branch
func SetProtection(store *storage.ProtectionStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		repoID := chi.URLParam(r, "id")
		branch := chi.URLParam(r, "branch")

		var req struct {
			RequirePR       bool `json:"require_pr"`
			RequireApproval bool `json:"require_approval"`
			NoForcePush     bool `json:"no_force_push"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		protection := storage.BranchProtection{
			Branch:          branch,
			RequirePR:       req.RequirePR,
			RequireApproval: req.RequireApproval,
			NoForcePush:     req.NoForcePush,
		}

		if err := store.Set(repoID, protection); err != nil {
			http.Error(w, SanitizeError(err, "failed to set protection"), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success":        true,
			"branch":         branch,
			"require_pr":     req.RequirePR,
			"require_approval": req.RequireApproval,
			"no_force_push":  req.NoForcePush,
		})
	}
}

// RemoveProtection removes protection rules for a branch
func RemoveProtection(store *storage.ProtectionStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		repoID := chi.URLParam(r, "id")
		branch := chi.URLParam(r, "branch")

		if err := store.Remove(repoID, branch); err != nil {
			http.Error(w, SanitizeError(err, "protection not found"), http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"branch":  branch,
		})
	}
}
