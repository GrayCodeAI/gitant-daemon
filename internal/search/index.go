// Package search provides an in-memory code search index for repositories.
//
// The index is lazy: a repository's file contents are read and cached on the
// first search, then served from memory on subsequent searches. The cache is
// invalidated on push (see Invalidate), so the next search rebuilds it. This
// removes the repeated full tree-walk + blob reads that the previous per-request
// scan performed on every query.
//
// It is pure Go with no external dependencies (substring matching only).
package search

import (
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/go-git/go-git/v6/plumbing"
	"github.com/go-git/go-git/v6/plumbing/filemode"
	"github.com/lakshmanpatel/gitant/internal/storage"
)

const (
	// maxSearchDepth bounds tree recursion (matches the previous scanner).
	maxSearchDepth = 32
	// maxResults caps the number of matches returned for a single query.
	maxResults = 1000
	// maxContextLen truncates a matched line in the context string.
	maxContextLen = 200
)

// Result is a single search match. The JSON keys are kept identical to the
// previous scan-based handler so all consumers (web, CLI, MCP) keep working.
type Result struct {
	File    string `json:"file"`
	Line    int    `json:"line"`
	Context string `json:"context"`
}

// repoIndex is the cached, parsed contents of one repository at a given ref.
type repoIndex struct {
	// commit is the resolved commit hash the cache was built from. If the ref
	// now resolves to a different commit, the cache is stale and rebuilt.
	commit string
	// files maps a file path to its pre-split lines.
	files map[string][]string
}

// Index is a lazy, in-memory, per-repository code search index.
type Index struct {
	mu       sync.RWMutex
	registry *storage.RepositoryRegistry
	// repos is keyed by "<repoID>\x00<ref>" so different refs are cached
	// independently.
	repos map[string]*repoIndex
}

// New creates an Index backed by the given repository registry.
func New(registry *storage.RepositoryRegistry) *Index {
	return &Index{
		registry: registry,
		repos:    make(map[string]*repoIndex),
	}
}

// cacheKey builds the map key for a repo+ref pair.
func cacheKey(repoID, ref string) string {
	return repoID + "\x00" + ref
}

// Invalidate drops all cached entries for a repository (all refs). Call this
// when the repository's contents change (e.g. on push).
func (ix *Index) Invalidate(repoID string) {
	ix.mu.Lock()
	defer ix.mu.Unlock()
	prefix := repoID + "\x00"
	for k := range ix.repos {
		if strings.HasPrefix(k, prefix) {
			delete(ix.repos, k)
		}
	}
}

// Search returns matches for query in the given repository/ref. ref may be
// empty, in which case the default branch (HEAD/main/master, else the first
// ref) is used. Results are paginated by offset/limit; total is the full match
// count before pagination.
func (ix *Index) Search(repoID, query, ref string, offset, limit int) ([]Result, int, error) {
	if query == "" {
		return nil, 0, fmt.Errorf("query is required")
	}

	repo, err := ix.registry.Open(repoID)
	if err != nil {
		return nil, 0, fmt.Errorf("repository not found: %w", err)
	}

	commitHash, err := resolveCommit(repo, ref)
	if err != nil {
		return nil, 0, err
	}

	ri, err := ix.getOrBuild(repo, repoID, ref, commitHash)
	if err != nil {
		return nil, 0, err
	}

	all := searchFiles(ri.files, query)
	total := len(all)
	paged := paginate(all, offset, limit)
	return paged, total, nil
}

// getOrBuild returns a fresh cached repoIndex, rebuilding it if absent or stale.
func (ix *Index) getOrBuild(repo *storage.Repository, repoID, ref string, commitHash plumbing.Hash) (*repoIndex, error) {
	key := cacheKey(repoID, ref)
	commitStr := commitHash.String()

	ix.mu.RLock()
	cached, ok := ix.repos[key]
	ix.mu.RUnlock()
	if ok && cached.commit == commitStr {
		return cached, nil
	}

	// Resolve the tree and walk it once (outside the lock — disk I/O).
	commit, err := repo.GetCommit(commitHash)
	if err != nil {
		return nil, fmt.Errorf("commit not found: %w", err)
	}
	files := make(map[string][]string)
	buildTree(repo, commit.TreeHash, commit.TreeHash, "", files, 0)

	ri := &repoIndex{commit: commitStr, files: files}

	ix.mu.Lock()
	ix.repos[key] = ri
	ix.mu.Unlock()
	return ri, nil
}

// resolveCommit picks the commit hash to index for the given ref (or default).
func resolveCommit(repo *storage.Repository, ref string) (plumbing.Hash, error) {
	if ref != "" {
		hash, err := repo.GetBranch(ref)
		if err != nil {
			return plumbing.ZeroHash, fmt.Errorf("branch not found: %s", ref)
		}
		return hash, nil
	}

	refs, err := repo.ListAllRefs()
	if err != nil || len(refs) == 0 {
		return plumbing.ZeroHash, fmt.Errorf("no refs found")
	}
	for _, r := range refs {
		if strings.HasSuffix(r.Name, "/HEAD") ||
			strings.HasSuffix(r.Name, "/main") ||
			strings.HasSuffix(r.Name, "/master") {
			return plumbing.NewHash(r.Hash), nil
		}
	}
	return plumbing.NewHash(refs[0].Hash), nil
}

// buildTree recursively reads file contents into files (path -> lines).
func buildTree(repo *storage.Repository, rootHash, currentHash plumbing.Hash, path string, files map[string][]string, depth int) {
	if depth >= maxSearchDepth {
		return
	}
	entries, err := repo.ListTreeEntries(currentHash, "")
	if err != nil {
		return
	}
	for _, entry := range entries {
		entryPath := entry.Name
		if path != "" {
			entryPath = path + "/" + entry.Name
		}
		if entry.Mode == filemode.Dir {
			buildTree(repo, rootHash, entry.Hash, entryPath, files, depth+1)
			continue
		}
		content, err := repo.GetFileFromTree(rootHash, entryPath)
		if err != nil {
			continue
		}
		files[entryPath] = strings.Split(string(content), "\n")
	}
}

// searchFiles scans cached file lines for the (case-insensitive) query.
func searchFiles(files map[string][]string, query string) []Result {
	lowerQuery := strings.ToLower(query)
	var results []Result
	// Iterate in deterministic path order so pagination is stable.
	for _, path := range sortedKeys(files) {
		for i, line := range files[path] {
			if strings.Contains(strings.ToLower(line), lowerQuery) {
				ctx := line
				if len(ctx) > maxContextLen {
					ctx = ctx[:maxContextLen] + "..."
				}
				results = append(results, Result{
					File:    path,
					Line:    i + 1,
					Context: fmt.Sprintf("%s:%d: %s", path, i+1, strings.TrimSpace(ctx)),
				})
				if len(results) >= maxResults {
					return results
				}
			}
		}
	}
	return results
}

// sortedKeys returns map keys sorted for stable, deterministic iteration.
func sortedKeys(files map[string][]string) []string {
	keys := make([]string, 0, len(files))
	for k := range files {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// paginate returns the offset/limit window of results.
func paginate(results []Result, offset, limit int) []Result {
	if offset < 0 {
		offset = 0
	}
	if offset >= len(results) {
		return []Result{}
	}
	end := len(results)
	if limit > 0 && offset+limit < end {
		end = offset + limit
	}
	return results[offset:end]
}
