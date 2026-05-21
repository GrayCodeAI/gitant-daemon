package core

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/go-git/go-git/v6/plumbing"
)

func TestCIDStore(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "gitant-cid-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	storePath := filepath.Join(tmpDir, "cid-store.json")
	store := NewCIDStore(storePath)

	// Test AddMapping and GetCID
	gitHash := plumbing.NewHash("abc123def456abc123def456abc123def456abc1")
	cid := NewCID([]byte("test content"))

	store.AddMapping(gitHash, cid)

	retrievedCID, ok := store.GetCID(gitHash)
	if !ok {
		t.Fatal("expected to find CID for git hash")
	}
	if !retrievedCID.Equals(cid) {
		t.Fatalf("expected %s, got %s", cid, retrievedCID)
	}

	// Test GetGitHash
	retrievedHash, ok := store.GetGitHash(cid)
	if !ok {
		t.Fatal("expected to find git hash for CID")
	}
	if retrievedHash != gitHash {
		t.Fatalf("expected %s, got %s", gitHash, retrievedHash)
	}

	// Test Save and Load
	if err := store.Save(); err != nil {
		t.Fatal(err)
	}

	// Create new store and load
	store2 := NewCIDStore(storePath)
	if err := store2.Load(); err != nil {
		t.Fatal(err)
	}

	// Verify data persisted
	retrievedCID2, ok := store2.GetCID(gitHash)
	if !ok {
		t.Fatal("expected to find CID after load")
	}
	if !retrievedCID2.Equals(cid) {
		t.Fatalf("expected %s, got %s", cid, retrievedCID2)
	}
}

func TestCIDFromGitHash(t *testing.T) {
	gitHash := plumbing.NewHash("abc123def456abc123def456abc123def456abc1")
	cid := CIDFromGitHash(gitHash)

	if cid == nil {
		t.Fatal("expected CID, got nil")
	}

	if cid.Hash == "" {
		t.Fatal("expected non-empty hash")
	}

	// Same git hash should produce same CID
	cid2 := CIDFromGitHash(gitHash)
	if !cid.Equals(cid2) {
		t.Fatal("expected same CID for same git hash")
	}
}

func TestGitObjectToCID(t *testing.T) {
	content := []byte("Hello, gitant!")
	cid := GitObjectToCID(plumbing.BlobObject, content)

	if cid == nil {
		t.Fatal("expected CID, got nil")
	}

	// Same content should produce same CID
	cid2 := GitObjectToCID(plumbing.BlobObject, content)
	if !cid.Equals(cid2) {
		t.Fatal("expected same CID for same content")
	}

	// Different content should produce different CID
	cid3 := GitObjectToCID(plumbing.BlobObject, []byte("different content"))
	if cid.Equals(cid3) {
		t.Fatal("expected different CID for different content")
	}
}
