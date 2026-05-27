package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-git/go-git/v6/plumbing"
	"github.com/go-git/go-git/v6/plumbing/filemode"
	"github.com/lakshmanpatel/gitant/internal/storage"
)

// GetCommitLog returns the commit history
func GetCommitLog(registry *storage.RepositoryRegistry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		offset, limit := ParsePagination(r)
		repoID := chi.URLParam(r, "id")
		ref := r.URL.Query().Get("ref")

		repo, err := registry.Open(repoID)
		if err != nil {
			http.Error(w, SanitizeError(err, "repository not found"), http.StatusNotFound)
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

		// Walk enough commits for the requested page
		commits, err := repo.WalkCommits(startHash, offset+limit)
		if err != nil {
			http.Error(w, SanitizeError(err, "failed to walk commits"), http.StatusInternalServerError)
			return
		}

		paged, total := PaginateSlice(commits, offset, limit)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"commits": paged,
			"total":   total,
			"offset":  offset,
			"limit":   limit,
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
			http.Error(w, SanitizeError(err, "repository not found"), http.StatusNotFound)
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

		changes := diffTrees(repo, from.TreeHash, to.TreeHash)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"from":    fromHash,
			"to":      toHash,
			"changes": changes,
			"total":   len(changes),
		})
	}
}

// DiffCommitAllParents shows diffs against all parent commits (for merge commits)
func DiffCommitAllParents(registry *storage.RepositoryRegistry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		repoID := chi.URLParam(r, "id")
		commitHash := chi.URLParam(r, "hash")

		repo, err := registry.Open(repoID)
		if err != nil {
			http.Error(w, SanitizeError(err, "repository not found"), http.StatusNotFound)
			return
		}

		commit, err := repo.GetCommit(plumbing.NewHash(commitHash))
		if err != nil {
			http.Error(w, "Commit not found", http.StatusNotFound)
			return
		}

		if len(commit.ParentHashes) == 0 {
			// Root commit: diff against empty tree
			changes := diffTrees(repo, plumbing.ZeroHash, commit.TreeHash)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"commit":   commitHash,
				"parents":  []map[string]interface{}{{"hash": "root", "changes": changes, "total": len(changes)}},
			})
			return
		}

		parentDiffs := make([]map[string]interface{}, 0, len(commit.ParentHashes))
		for _, parentHash := range commit.ParentHashes {
			parent, err := repo.GetCommit(parentHash)
			if err != nil {
				continue
			}
			changes := diffTrees(repo, parent.TreeHash, commit.TreeHash)
			parentDiffs = append(parentDiffs, map[string]interface{}{
				"hash":    parentHash.String(),
				"changes": changes,
				"total":   len(changes),
			})
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"commit":  commitHash,
			"parents": parentDiffs,
		})
	}
}

// GetDiff returns a unified diff between two commits.
func GetDiff(registry *storage.RepositoryRegistry) http.HandlerFunc {
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
			http.Error(w, SanitizeError(err, "repository not found"), http.StatusNotFound)
			return
		}

		fromCommit, err := repo.GetCommit(plumbing.NewHash(fromHash))
		if err != nil {
			http.Error(w, "From commit not found", http.StatusNotFound)
			return
		}

		toCommit, err := repo.GetCommit(plumbing.NewHash(toHash))
		if err != nil {
			http.Error(w, "To commit not found", http.StatusNotFound)
			return
		}

		patch := computePatch(repo, fromCommit.TreeHash, toCommit.TreeHash)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"from":  fromHash,
			"to":    toHash,
			"patch": patch,
		})
	}
}

func computePatch(repo *storage.Repository, fromTree, toTree plumbing.Hash) string {
	var patch strings.Builder
	fromEntries := treeEntryMap(repo, fromTree)
	toEntries := treeEntryMap(repo, toTree)

	// Check all files in toTree
	for name, toHash := range toEntries {
		fromHash, existed := fromEntries[name]
		if !existed {
			// Added file
			content := fileContent(repo, toHash)
			patch.WriteString("--- /dev/null\n")
			patch.WriteString("+++ b/" + name + "\n")
			for _, line := range strings.Split(content, "\n") {
				patch.WriteString("+" + line + "\n")
			}
			patch.WriteString("\n")
		} else if fromHash != toHash {
			// Modified file
			oldContent := fileContent(repo, fromHash)
			newContent := fileContent(repo, toHash)
			patch.WriteString("--- a/" + name + "\n")
			patch.WriteString("+++ b/" + name + "\n")
			writeSimpleDiff(&patch, oldContent, newContent)
			patch.WriteString("\n")
		}
	}

	// Check deleted files
	for name, fromHash := range fromEntries {
		if _, exists := toEntries[name]; !exists {
			content := fileContent(repo, fromHash)
			patch.WriteString("--- a/" + name + "\n")
			patch.WriteString("+++ /dev/null\n")
			for _, line := range strings.Split(content, "\n") {
				patch.WriteString("-" + line + "\n")
			}
			patch.WriteString("\n")
		}
	}

	return patch.String()
}

func treeEntryMap(repo *storage.Repository, treeHash plumbing.Hash) map[string]plumbing.Hash {
	entries, err := repo.ListTreeEntries(treeHash, "")
	if err != nil {
		return nil
	}
	m := make(map[string]plumbing.Hash, len(entries))
	for _, e := range entries {
		if e.Mode == filemode.Regular || e.Mode == filemode.Executable {
			m[e.Name] = e.Hash
		}
	}
	return m
}

func fileContent(repo *storage.Repository, hash plumbing.Hash) string {
	_, data, err := repo.GetObject(hash)
	if err != nil {
		return ""
	}
	return string(data)
}

func writeSimpleDiff(w *strings.Builder, old, new string) {
	oldLines := strings.Split(old, "\n")
	newLines := strings.Split(new, "\n")
	w.WriteString("@@ -1," + strconv.Itoa(len(oldLines)) + " +1," + strconv.Itoa(len(newLines)) + " @@\n")
	for _, line := range oldLines {
		w.WriteString("-" + line + "\n")
	}
	for _, line := range newLines {
		w.WriteString("+" + line + "\n")
	}
}

// diffTrees compares two tree hashes and returns changes
func diffTrees(repo *storage.Repository, fromTree, toTree plumbing.Hash) []map[string]string {
	fromEntries, err := repo.ListTreeEntries(fromTree, "")
	if err != nil {
		return nil
	}
	toEntries, err := repo.ListTreeEntries(toTree, "")
	if err != nil {
		return nil
	}

	fromMap := make(map[string]string)
	for _, e := range fromEntries {
		fromMap[e.Name] = e.Hash.String()
	}
	toMap := make(map[string]string)
	for _, e := range toEntries {
		toMap[e.Name] = e.Hash.String()
	}

	changes := make([]map[string]string, 0)
	for name, hash := range toMap {
		if oldHash, exists := fromMap[name]; !exists {
			changes = append(changes, map[string]string{"file": name, "type": "added", "hash": hash})
		} else if oldHash != hash {
			changes = append(changes, map[string]string{"file": name, "type": "modified", "old_hash": oldHash, "new_hash": hash})
		}
	}
	for name, hash := range fromMap {
		if _, exists := toMap[name]; !exists {
			changes = append(changes, map[string]string{"file": name, "type": "deleted", "hash": hash})
		}
	}
	return changes
}
