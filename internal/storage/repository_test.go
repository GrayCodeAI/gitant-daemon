package storage

import (
	"os"
	"path/filepath"
	"testing"
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
