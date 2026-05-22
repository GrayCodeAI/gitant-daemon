package storage

import (
	"testing"

	"github.com/go-git/go-git/v6/plumbing"
)

func TestPackfileWriter(t *testing.T) {
	writer := NewPackfileWriter()

	objects := []*GitObject{
		{
			Type:    plumbing.BlobObject,
			Content: []byte("Hello, gitant!"),
		},
		{
			Type:    plumbing.BlobObject,
			Content: []byte("Another blob"),
		},
	}

	data, err := writer.WritePackfile(objects)
	if err != nil {
		t.Fatal(err)
	}

	if len(data) == 0 {
		t.Fatal("expected non-empty packfile data")
	}

	// Verify PACK header
	if string(data[:4]) != "PACK" {
		t.Fatalf("expected PACK header, got %q", string(data[:4]))
	}
}

func TestPackfileRoundtrip(t *testing.T) {
	writer := NewPackfileWriter()

	// Create objects with known content
	blob1Content := []byte("Hello, gitant!")
	blob2Content := []byte("Another blob with different content")
	commitContent := []byte("tree abc123\nauthor Test <test@test.com> 1234567890 +0000\n\nTest commit\n")

	objects := []*GitObject{
		{Type: plumbing.BlobObject, Content: blob1Content},
		{Type: plumbing.BlobObject, Content: blob2Content},
		{Type: plumbing.CommitObject, Content: commitContent},
	}

	// Write packfile
	data, err := writer.WritePackfile(objects)
	if err != nil {
		t.Fatalf("WritePackfile: %v", err)
	}

	// Read it back
	reader := NewPackfileReader()
	readObjects, err := reader.ReadPackfile(data)
	if err != nil {
		t.Fatalf("ReadPackfile: %v", err)
	}

	if len(readObjects) != len(objects) {
		t.Fatalf("expected %d objects, got %d", len(objects), len(readObjects))
	}

	// Build a map by type for comparison
	byType := make(map[plumbing.ObjectType][]*GitObject)
	for _, obj := range readObjects {
		byType[obj.Type] = append(byType[obj.Type], obj)
	}

	// Verify blobs
	blobs := byType[plumbing.BlobObject]
	if len(blobs) != 2 {
		t.Fatalf("expected 2 blobs, got %d", len(blobs))
	}

	blobContents := map[string]bool{}
	for _, b := range blobs {
		blobContents[string(b.Content)] = true
		if b.Hash.IsZero() {
			t.Fatal("expected non-zero hash for blob")
		}
	}
	if !blobContents[string(blob1Content)] {
		t.Fatal("missing blob1 content")
	}
	if !blobContents[string(blob2Content)] {
		t.Fatal("missing blob2 content")
	}

	// Verify commit
	commits := byType[plumbing.CommitObject]
	if len(commits) != 1 {
		t.Fatalf("expected 1 commit, got %d", len(commits))
	}
	if string(commits[0].Content) != string(commitContent) {
		t.Fatalf("commit content mismatch: got %q", string(commits[0].Content))
	}
}

func TestExtractObjects(t *testing.T) {
	writer := NewPackfileWriter()

	objects := []*GitObject{
		{Type: plumbing.BlobObject, Content: []byte("test content")},
	}

	data, err := writer.WritePackfile(objects)
	if err != nil {
		t.Fatalf("WritePackfile: %v", err)
	}

	// Use the convenience function
	extracted, err := ExtractObjects(data)
	if err != nil {
		t.Fatalf("ExtractObjects: %v", err)
	}

	if len(extracted) != 1 {
		t.Fatalf("expected 1 object, got %d", len(extracted))
	}
	if extracted[0].Type != plumbing.BlobObject {
		t.Fatalf("expected BlobObject, got %v", extracted[0].Type)
	}
	if string(extracted[0].Content) != "test content" {
		t.Fatalf("content mismatch: got %q", string(extracted[0].Content))
	}
}

func TestPackfileRoundtripMultipleTypes(t *testing.T) {
	writer := NewPackfileWriter()

	treeContent := []byte("100644 file.txt\x00" + string(make([]byte, 20)))
	objects := []*GitObject{
		{Type: plumbing.BlobObject, Content: []byte("blob data")},
		{Type: plumbing.TreeObject, Content: treeContent},
		{Type: plumbing.TagObject, Content: []byte("object abc123\ntype commit\ntag v1.0\ntagger Test <t@t> 123 +0000\n\nRelease\n")},
	}

	data, err := writer.WritePackfile(objects)
	if err != nil {
		t.Fatalf("WritePackfile: %v", err)
	}

	reader := NewPackfileReader()
	readObjects, err := reader.ReadPackfile(data)
	if err != nil {
		t.Fatalf("ReadPackfile: %v", err)
	}

	if len(readObjects) != 3 {
		t.Fatalf("expected 3 objects, got %d", len(readObjects))
	}

	types := map[plumbing.ObjectType]bool{}
	for _, obj := range readObjects {
		types[obj.Type] = true
	}
	if !types[plumbing.BlobObject] || !types[plumbing.TreeObject] || !types[plumbing.TagObject] {
		t.Fatalf("missing expected object types: %v", types)
	}
}
