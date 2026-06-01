package search

import (
	"testing"

	"github.com/go-git/go-git/v6/plumbing"
	"github.com/go-git/go-git/v6/plumbing/filemode"
	"github.com/lakshmanpatel/gitant/internal/storage"
)

// newTestRepo creates a registry + a repo seeded with the given files on "main".
// files maps path -> content. Returns the registry and repo id.
func newTestRepo(t *testing.T, files map[string]string) (*storage.RepositoryRegistry, string) {
	t.Helper()
	reg, err := storage.NewRepositoryRegistry(t.TempDir(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	const id = "repo1"
	if _, err := reg.Create(id, id, "", false); err != nil {
		t.Fatalf("create repo: %v", err)
	}
	commitFiles(t, reg, id, files, nil)
	return reg, id
}

// commitFiles writes files as a new commit on "main" and updates the branch.
func commitFiles(t *testing.T, reg *storage.RepositoryRegistry, id string, files map[string]string, parent *plumbing.Hash) plumbing.Hash {
	t.Helper()
	repo, err := reg.Open(id)
	if err != nil {
		t.Fatalf("open repo: %v", err)
	}
	var entries []storage.TreeEntry
	for path, content := range files {
		blob, err := repo.CreateBlob([]byte(content))
		if err != nil {
			t.Fatalf("create blob: %v", err)
		}
		entries = append(entries, storage.TreeEntry{Name: path, Hash: blob, Mode: filemode.Regular})
	}
	tree, err := repo.CreateTree(entries)
	if err != nil {
		t.Fatalf("create tree: %v", err)
	}
	var parents []plumbing.Hash
	if parent != nil {
		parents = []plumbing.Hash{*parent}
	}
	commit, err := repo.CreateCommit(tree, parents, "alice", "commit")
	if err != nil {
		t.Fatalf("create commit: %v", err)
	}
	if err := repo.CreateBranch("main", commit); err != nil {
		// CreateBranch may fail if it already exists; fall back to UpdateRef.
		if err := repo.UpdateRef("main", commit); err != nil {
			t.Fatalf("update main: %v", err)
		}
	}
	return commit
}

func TestSearch_FindsMatch(t *testing.T) {
	reg, id := newTestRepo(t, map[string]string{
		"main.go": "package main\n\nfunc main() {\n\tprintln(\"hello world\")\n}\n",
	})
	ix := New(reg)

	results, total, err := ix.Search(id, "hello", "", 0, 50)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if total != 1 || len(results) != 1 {
		t.Fatalf("expected 1 result, got total=%d len=%d", total, len(results))
	}
	r := results[0]
	if r.File != "main.go" {
		t.Errorf("expected file main.go, got %q", r.File)
	}
	if r.Line != 4 {
		t.Errorf("expected line 4, got %d", r.Line)
	}
	if r.Context == "" {
		t.Error("expected non-empty context")
	}
}

func TestSearch_CaseInsensitive(t *testing.T) {
	reg, id := newTestRepo(t, map[string]string{"a.txt": "Hello World"})
	ix := New(reg)
	results, total, err := ix.Search(id, "WORLD", "", 0, 50)
	if err != nil {
		t.Fatal(err)
	}
	if total != 1 || len(results) != 1 {
		t.Fatalf("expected 1 case-insensitive match, got %d", total)
	}
}

func TestSearch_NoMatch(t *testing.T) {
	reg, id := newTestRepo(t, map[string]string{"a.txt": "nothing here"})
	ix := New(reg)
	results, total, err := ix.Search(id, "absent", "", 0, 50)
	if err != nil {
		t.Fatal(err)
	}
	if total != 0 || len(results) != 0 {
		t.Fatalf("expected 0 matches, got %d", total)
	}
}

func TestSearch_EmptyQuery(t *testing.T) {
	reg, id := newTestRepo(t, map[string]string{"a.txt": "x"})
	ix := New(reg)
	if _, _, err := ix.Search(id, "", "", 0, 50); err == nil {
		t.Fatal("expected error for empty query")
	}
}

func TestSearch_RepoNotFound(t *testing.T) {
	reg, err := storage.NewRepositoryRegistry(t.TempDir(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	ix := New(reg)
	if _, _, err := ix.Search("missing", "x", "", 0, 50); err == nil {
		t.Fatal("expected error for missing repo")
	}
}

func TestSearch_Pagination(t *testing.T) {
	// 5 lines all matching "row".
	reg, id := newTestRepo(t, map[string]string{
		"data.txt": "row1\nrow2\nrow3\nrow4\nrow5\n",
	})
	ix := New(reg)

	results, total, err := ix.Search(id, "row", "", 1, 2)
	if err != nil {
		t.Fatal(err)
	}
	if total != 5 {
		t.Fatalf("expected total 5, got %d", total)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 paged results, got %d", len(results))
	}
	// offset 1 -> first returned should be line 2.
	if results[0].Line != 2 {
		t.Errorf("expected first paged line 2, got %d", results[0].Line)
	}
}

func TestSearch_CacheServedAfterFirst(t *testing.T) {
	reg, id := newTestRepo(t, map[string]string{"a.txt": "alpha"})
	ix := New(reg)

	if _, _, err := ix.Search(id, "alpha", "", 0, 50); err != nil {
		t.Fatal(err)
	}
	// Cache entry should now exist for repo+default ref.
	ix.mu.RLock()
	n := len(ix.repos)
	ix.mu.RUnlock()
	if n == 0 {
		t.Fatal("expected a cached repoIndex after first search")
	}
}

func TestInvalidate_DropsCacheAndRebuilds(t *testing.T) {
	reg, id := newTestRepo(t, map[string]string{"a.txt": "original"})
	ix := New(reg)

	// Prime the cache.
	if _, total, err := ix.Search(id, "original", "", 0, 50); err != nil || total != 1 {
		t.Fatalf("prime search: total=%d err=%v", total, err)
	}

	// Change repo content (new commit on main) and invalidate.
	parent, _ := openHead(t, reg, id)
	commitFiles(t, reg, id, map[string]string{"a.txt": "updated content"}, &parent)
	ix.Invalidate(id)

	// Old content gone, new content found.
	if _, total, err := ix.Search(id, "original", "", 0, 50); err != nil || total != 0 {
		t.Fatalf("expected old content gone after invalidate: total=%d err=%v", total, err)
	}
	if _, total, err := ix.Search(id, "updated", "", 0, 50); err != nil || total != 1 {
		t.Fatalf("expected new content found after invalidate: total=%d err=%v", total, err)
	}
}

// openHead returns the current commit hash of main.
func openHead(t *testing.T, reg *storage.RepositoryRegistry, id string) (plumbing.Hash, error) {
	t.Helper()
	repo, err := reg.Open(id)
	if err != nil {
		t.Fatal(err)
	}
	return repo.GetBranch("main")
}
