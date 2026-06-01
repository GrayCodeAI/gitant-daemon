package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-git/go-git/v6/plumbing"
	"github.com/lakshmanpatel/gitant/internal/search"
	"github.com/lakshmanpatel/gitant/internal/storage"
)

// SearchCodeIndexed serves code search from the in-memory search index. The
// response shape is identical to the legacy SearchCode handler
// ({query, results:[{file,line,context}], total}) so all consumers keep working.
func SearchCodeIndexed(ix *search.Index) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		repoID := chi.URLParam(r, "id")
		query := r.URL.Query().Get("q")
		ref := r.URL.Query().Get("ref")

		if query == "" {
			http.Error(w, "Query parameter 'q' is required", http.StatusBadRequest)
			return
		}

		offset, limit := ParsePagination(r)
		results, total, err := ix.Search(repoID, query, ref, offset, limit)
		if err != nil {
			http.Error(w, SanitizeError(err, "search failed"), http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"query":   query,
			"results": results,
			"total":   total,
		})
	}
}

// ListFiles lists files in a repository tree
func ListFiles(registry *storage.RepositoryRegistry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		repoID := chi.URLParam(r, "id")
		path := r.URL.Query().Get("path")
		ref := r.URL.Query().Get("ref")

		repo, err := registry.Open(repoID)
		if err != nil {
			http.Error(w, SanitizeError(err, "repository not found"), http.StatusNotFound)
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
			http.Error(w, SanitizeError(err, "failed to list entries"), http.StatusInternalServerError)
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

		// Validate path to prevent traversal
		if strings.Contains(path, "..") || strings.Contains(path, "\x00") || strings.HasPrefix(path, "/") {
			http.Error(w, "invalid file path", http.StatusBadRequest)
			return
		}

		repo, err := registry.Open(repoID)
		if err != nil {
			http.Error(w, SanitizeError(err, "repository not found"), http.StatusNotFound)
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
			http.Error(w, SanitizeError(err, "file not found"), http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/octet-stream")
		w.Write(content)
	}
}

// Code search is served by SearchCodeIndexed (see top of this file), backed by
// the in-memory index in internal/search. The previous per-request tree-walk
// scanner (SearchCode/searchTree) was removed in favor of the cached index.
