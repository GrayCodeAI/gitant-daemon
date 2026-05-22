package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-git/go-git/v6/plumbing"
	"github.com/lakshmanpatel/gitant/internal/storage"
)

// ListFiles lists files in a repository tree
func ListFiles(registry *storage.RepositoryRegistry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		repoID := chi.URLParam(r, "id")
		path := r.URL.Query().Get("path")
		ref := r.URL.Query().Get("ref")

		repo, err := registry.Open(repoID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		// Get the tree hash from ref or HEAD
		var treeHash plumbing.Hash
		if ref != "" {
			hash, err := repo.GetBranch(ref)
			if err != nil {
				http.Error(w, "Branch not found: "+ref, http.StatusNotFound)
				return
			}
			treeHash = hash
		} else {
			// Try to get HEAD
			refs, err := repo.ListAllRefs()
			if err != nil || len(refs) == 0 {
				http.Error(w, "No refs found", http.StatusNotFound)
				return
			}
			// Find HEAD or main/master
			for _, r := range refs {
				if strings.HasSuffix(r.Name, "/HEAD") || strings.HasSuffix(r.Name, "/main") || strings.HasSuffix(r.Name, "/master") {
					treeHash = plumbing.NewHash(r.Hash)
					break
				}
			}
			if treeHash.IsZero() {
				treeHash = plumbing.NewHash(refs[0].Hash)
			}
		}

		// Get the commit to find the tree hash
		commit, err := repo.GetCommit(treeHash)
		if err != nil {
			http.Error(w, "Commit not found", http.StatusNotFound)
			return
		}

		entries, err := repo.ListTreeEntries(commit.TreeHash, path)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		result := make([]map[string]interface{}, 0, len(entries))
		for _, entry := range entries {
			entryType := "tree"
			if entry.Mode.IsFile() {
				entryType = "blob"
			}
			result = append(result, map[string]interface{}{
				"name": entry.Name,
				"mode": entry.Mode.String(),
				"hash": entry.Hash.String(),
				"type": entryType,
			})
		}

		offset, limit := ParsePagination(r)
		paged, total := PaginateSlice(result, offset, limit)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"entries": paged,
			"total":   total,
			"offset":  offset,
			"limit":   limit,
			"path":    path,
		})
	}
}

// GetFile retrieves a file's content from a repository
func GetFile(registry *storage.RepositoryRegistry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		repoID := chi.URLParam(r, "id")
		path := chi.URLParam(r, "path")
		ref := r.URL.Query().Get("ref")

		repo, err := registry.Open(repoID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		// Get the tree hash from ref or HEAD
		var treeHash plumbing.Hash
		if ref != "" {
			hash, err := repo.GetBranch(ref)
			if err != nil {
				http.Error(w, "Branch not found: "+ref, http.StatusNotFound)
				return
			}
			treeHash = hash
		} else {
			refs, err := repo.ListAllRefs()
			if err != nil || len(refs) == 0 {
				http.Error(w, "No refs found", http.StatusNotFound)
				return
			}
			for _, r := range refs {
				if strings.HasSuffix(r.Name, "/HEAD") || strings.HasSuffix(r.Name, "/main") || strings.HasSuffix(r.Name, "/master") {
					treeHash = plumbing.NewHash(r.Hash)
					break
				}
			}
			if treeHash.IsZero() {
				treeHash = plumbing.NewHash(refs[0].Hash)
			}
		}

		commit, err := repo.GetCommit(treeHash)
		if err != nil {
			http.Error(w, "Commit not found", http.StatusNotFound)
			return
		}

		content, err := repo.GetFileFromTree(commit.TreeHash, path)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/octet-stream")
		w.Write(content)
	}
}

// SearchCode searches for text in repository blobs
func SearchCode(registry *storage.RepositoryRegistry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		repoID := chi.URLParam(r, "id")
		query := r.URL.Query().Get("q")

		if query == "" {
			http.Error(w, "Query parameter 'q' is required", http.StatusBadRequest)
			return
		}

		_ = repoID
		// TODO: Implement full-text search across blobs
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"query":   query,
			"results": []interface{}{},
			"total":   0,
		})
	}
}
