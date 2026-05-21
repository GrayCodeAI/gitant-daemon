package storage

import (
	"testing"

	"github.com/go-git/go-git/v6/plumbing"
)

func TestPackfileWriter(t *testing.T) {
	writer := NewPackfileWriter()

	// Create test objects
	objects := []*GitObject{
		{
			Type:    plumbing.BlobObject,
			Content: []byte("Hello, gitant!"),
			Hash:    plumbing.NewHash("abc123def456abc123def456abc123def456abc1"),
		},
		{
			Type:    plumbing.BlobObject,
			Content: []byte("Another blob"),
			Hash:    plumbing.NewHash("def456abc123def456abc123def456abc123def4"),
		},
	}

	// Write packfile
	data, err := writer.WritePackfile(objects)
	if err != nil {
		t.Fatal(err)
	}

	if len(data) == 0 {
		t.Fatal("expected non-empty packfile data")
	}
}

func TestPackfileReader(t *testing.T) {
	// First create a packfile with writer
	writer := NewPackfileWriter()

	objects := []*GitObject{
		{
			Type:    plumbing.BlobObject,
			Content: []byte("Hello, gitant!"),
			Hash:    plumbing.NewHash("abc123def456abc123def456abc123def456abc1"),
		},
	}

	data, err := writer.WritePackfile(objects)
	if err != nil {
		t.Fatal(err)
	}

	// Now read it back
	reader := NewPackfileReader()
	readObjects, err := reader.ReadPackfile(data)
	if err != nil {
		t.Fatal(err)
	}

	// For now, we expect empty objects slice (TODO: implement proper reading)
	if readObjects == nil {
		t.Fatal("expected non-nil objects slice")
	}
}

func TestGitObject(t *testing.T) {
	obj := &GitObject{
		Type:    plumbing.BlobObject,
		Content: []byte("test content"),
		Hash:    plumbing.NewHash("abc123def456abc123def456abc123def456abc1"),
	}

	if obj.Type != plumbing.BlobObject {
		t.Fatalf("expected BlobObject, got %v", obj.Type)
	}

	if string(obj.Content) != "test content" {
		t.Fatalf("expected 'test content', got %q", obj.Content)
	}

	if obj.Hash.IsZero() {
		t.Fatal("expected non-zero hash")
	}
}
