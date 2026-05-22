package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-git/go-git/v6/plumbing"
	"github.com/go-git/go-git/v6/plumbing/filemode"
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
		ref := r.URL.Query().Get("ref")

		if query == "" {
			http.Error(w, "Query parameter 'q' is required", http.StatusBadRequest)
			return
		}

		repo, err := registry.Open(repoID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		// Get starting commit
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
			for _, ref := range refs {
				if strings.HasSuffix(ref.Name, "/HEAD") || strings.HasSuffix(ref.Name, "/main") || strings.HasSuffix(ref.Name, "/master") {
					startHash = plumbing.NewHash(ref.Hash)
					break
				}
			}
			if startHash.IsZero() {
				startHash = plumbing.NewHash(refs[0].Hash)
			}
		}

		// Get commit to access tree
		commit, err := repo.GetCommit(startHash)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Walk tree and search blobs
		offset, limit := ParsePagination(r)
		results := searchTree(repo, commit.TreeHash, commit.TreeHash, "", query)

		paged, total := PaginateSlice(results, offset, limit)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"query":   query,
			"results": paged,
			"total":   total,
		})
	}
}

// searchTree recursively walks the tree and finds blobs containing the query
// rootHash is used for GetFileFromTree (full path resolution), currentHash is the subtree being listed
func searchTree(repo *storage.Repository, rootHash, currentHash plumbing.Hash, path string, query string) []map[string]interface{} {
	var results []map[string]interface{}

	entries, err := repo.ListTreeEntries(currentHash, "")
	if err != nil {
		return results
	}

	lowerQuery := strings.ToLower(query)

	for _, entry := range entries {
		entryPath := entry.Name
		if path != "" {
			entryPath = path + "/" + entry.Name
		}

		if entry.Mode == filemode.Dir {
			subResults := searchTree(repo, rootHash, entry.Hash, entryPath, query)
			results = append(results, subResults...)
		} else {
			content, err := repo.GetFileFromTree(rootHash, entryPath)
			if err != nil {
				continue
			}

			lines := strings.Split(string(content), "\n")
			for i, line := range lines {
				if strings.Contains(strings.ToLower(line), lowerQuery) {
					context := line
					if len(context) > 200 {
						context = context[:200] + "..."
					}
					results = append(results, map[string]interface{}{
						"file":    entryPath,
						"line":    i + 1,
						"context": fmt.Sprintf("%s:%d: %s", entryPath, i+1, strings.TrimSpace(context)),
					})

					if len(results) >= 1000 {
						return results
					}
				}
			}
		}
	}

	return results
}
