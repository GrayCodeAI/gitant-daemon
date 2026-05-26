package storage

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/go-git/go-git/v6/plumbing"
	"github.com/go-git/go-git/v6/plumbing/filemode"
)

func TestInitRepository(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "gitant-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	repoPath := filepath.Join(tmpDir, "test-repo")

	// Initialize repository
	repo, err := InitRepository(repoPath)
	if err != nil {
		t.Fatal(err)
	}

	if repo == nil {
		t.Fatal("expected repository, got nil")
	}

	// Verify directory exists
	if _, err := os.Stat(repoPath); os.IsNotExist(err) {
		t.Fatal("repository directory does not exist")
	}
}

func TestCreateBlob(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "gitant-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	repoPath := filepath.Join(tmpDir, "test-repo")

	// Initialize repository
	repo, err := InitRepository(repoPath)
	if err != nil {
		t.Fatal(err)
	}

	// Create a blob
	content := []byte("Hello, gitant!")
	hash, err := repo.CreateBlob(content)
	if err != nil {
		t.Fatal(err)
	}

	if hash.IsZero() {
		t.Fatal("expected non-zero hash")
	}

	// Retrieve the blob
	retrieved, err := repo.GetBlob(hash)
	if err != nil {
		t.Fatal(err)
	}

	if string(retrieved) != string(content) {
		t.Fatalf("expected %q, got %q", content, retrieved)
	}
}

func TestBranchOperations(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "gitant-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	repoPath := filepath.Join(tmpDir, "test-repo")

	// Initialize repository
	repo, err := InitRepository(repoPath)
	if err != nil {
		t.Fatal(err)
	}

	// Create a blob to get a commit hash
	content := []byte("test content")
	blobHash, err := repo.CreateBlob(content)
	if err != nil {
		t.Fatal(err)
	}

	// Create a branch
	err = repo.CreateBranch("main", blobHash)
	if err != nil {
		t.Fatal(err)
	}

	// Get the branch
	branchHash, err := repo.GetBranch("main")
	if err != nil {
		t.Fatal(err)
	}

	if branchHash != blobHash {
		t.Fatalf("expected %s, got %s", blobHash, branchHash)
	}

	// Delete the branch
	err = repo.DeleteBranch("main")
	if err != nil {
		t.Fatal(err)
	}

	// Verify branch is deleted
	_, err = repo.GetBranch("main")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestMergeBranchesCreatesMergeCommit(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "gitant-merge-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	repo, err := InitRepository(filepath.Join(tmpDir, "repo"))
	if err != nil {
		t.Fatal(err)
	}

	mainBlob, err := repo.CreateBlob([]byte("main"))
	if err != nil {
		t.Fatal(err)
	}
	mainTree, err := repo.CreateTree([]TreeEntry{{Name: "README", Mode: filemode.Regular, Hash: mainBlob}})
	if err != nil {
		t.Fatal(err)
	}
	mainCommit, err := repo.CreateCommit(mainTree, nil, "alice", "init")
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.CreateBranch("main", mainCommit); err != nil {
		t.Fatal(err)
	}

	featureBlob, err := repo.CreateBlob([]byte("feature"))
	if err != nil {
		t.Fatal(err)
	}
	featureTree, err := repo.CreateTree([]TreeEntry{{Name: "README", Mode: filemode.Regular, Hash: featureBlob}})
	if err != nil {
		t.Fatal(err)
	}
	featureCommit, err := repo.CreateCommit(featureTree, []plumbing.Hash{mainCommit}, "bob", "feature")
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.CreateBranch("feature", featureCommit); err != nil {
		t.Fatal(err)
	}

	mainBlob2, err := repo.CreateBlob([]byte("main v2"))
	if err != nil {
		t.Fatal(err)
	}
	mainTree2, err := repo.CreateTree([]TreeEntry{{Name: "README", Mode: filemode.Regular, Hash: mainBlob2}})
	if err != nil {
		t.Fatal(err)
	}
	mainCommit2, err := repo.CreateCommit(mainTree2, []plumbing.Hash{mainCommit}, "alice", "main advance")
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.UpdateRef("main", mainCommit2); err != nil {
		t.Fatal(err)
	}

	mergeHash, err := repo.MergeBranches("main", "feature", "carol", "merge", "merge")
	if err != nil {
		t.Fatal(err)
	}

	merged, err := repo.GetCommit(mergeHash)
	if err != nil {
		t.Fatal(err)
	}
	if len(merged.ParentHashes) != 2 {
		t.Fatalf("expected 2 parents, got %d", len(merged.ParentHashes))
	}
}
