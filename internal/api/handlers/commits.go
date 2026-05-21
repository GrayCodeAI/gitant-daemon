package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-git/go-git/v6/plumbing"
	"github.com/lakshmanpatel/gitant/internal/storage"
)

// GetCommitLog returns the commit history
func GetCommitLog(registry *storage.RepositoryRegistry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		repoID := chi.URLParam(r, "id")
		ref := r.URL.Query().Get("ref")
		limitStr := r.URL.Query().Get("limit")

		limit := 20
		if limitStr != "" {
			if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
				limit = l
			}
		}

		repo, err := registry.Open(repoID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		// Get starting commit hash
		var startHash plumbing.Hash
		if ref != "" {
			hash, err := repo.GetBranch(ref)
			if err != nil {
				http.Error(w, "Branch not found: "+ref, http.StatusNotFound)
				return
			}
			startHash = hash
		} else {
			refs, err := repo.ListAllRefs()
			if err != nil || len(refs) == 0 {
				http.Error(w, "No refs found", http.StatusNotFound)
				return
			}
			for _, r := range refs {
				if strings.HasSuffix(r.Name, "/HEAD") || strings.HasSuffix(r.Name, "/main") || strings.HasSuffix(r.Name, "/master") {
					startHash = plumbing.NewHash(r.Hash)
					break
				}
			}
			if startHash.IsZero() {
				startHash = plumbing.NewHash(refs[0].Hash)
			}
		}

		commits, err := repo.WalkCommits(startHash, limit)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"commits": commits,
			"total":   len(commits),
		})
	}
}

// DiffCommits shows the diff between two commits
func DiffCommits(registry *storage.RepositoryRegistry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		repoID := chi.URLParam(r, "id")
		fromHash := r.URL.Query().Get("from")
		toHash := r.URL.Query().Get("to")

		if fromHash == "" || toHash == "" {
			http.Error(w, "Both 'from' and 'to' parameters are required", http.StatusBadRequest)
			return
		}

		repo, err := registry.Open(repoID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		from, err := repo.GetCommit(plumbing.NewHash(fromHash))
		if err != nil {
			http.Error(w, "From commit not found", http.StatusNotFound)
			return
		}

		to, err := repo.GetCommit(plumbing.NewHash(toHash))
		if err != nil {
			http.Error(w, "To commit not found", http.StatusNotFound)
			return
		}

		// Simple diff: list changed files by comparing tree entries
		fromEntries, _ := repo.ListTreeEntries(from.TreeHash, "")
		toEntries, _ := repo.ListTreeEntries(to.TreeHash, "")

		fromMap := make(map[string]string)
		for _, e := range fromEntries {
			fromMap[e.Name] = e.Hash.String()
		}

		toMap := make(map[string]string)
		for _, e := range toEntries {
			toMap[e.Name] = e.Hash.String()
		}

		changes := make([]map[string]string, 0)

		// Added or modified
		for name, hash := range toMap {
			if oldHash, exists := fromMap[name]; !exists {
				changes = append(changes, map[string]string{"file": name, "type": "added", "hash": hash})
			} else if oldHash != hash {
				changes = append(changes, map[string]string{"file": name, "type": "modified", "old_hash": oldHash, "new_hash": hash})
			}
		}

		// Deleted
		for name, hash := range fromMap {
			if _, exists := toMap[name]; !exists {
				changes = append(changes, map[string]string{"file": name, "type": "deleted", "hash": hash})
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"from":    fromHash,
			"to":      toHash,
			"changes": changes,
			"total":   len(changes),
		})
	}
}
