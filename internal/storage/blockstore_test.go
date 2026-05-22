package storage

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/go-git/go-git/v6/plumbing"
)

func TestBlockstore(t *testing.T) {
	tmpDir := t.TempDir()
	bs := NewBlockstore(filepath.Join(tmpDir, "index.json"), filepath.Join(tmpDir, "blocks"))

	// Test Put and Get
	hash := plumbing.NewHash("abc123def456abc123def456abc123def456abc1")
	data := []byte("Hello, gitant!")

	err := bs.Put(hash, data)
	if err != nil {
		t.Fatal(err)
	}

	retrieved, err := bs.Get(hash)
	if err != nil {
		t.Fatal(err)
	}

	if string(retrieved) != string(data) {
		t.Fatalf("expected %q, got %q", data, retrieved)
	}

	// Test Has
	if !bs.Has(hash) {
		t.Fatal("expected block to exist")
	}

	// Verify block file exists on disk
	blockPath := filepath.Join(tmpDir, "blocks", hash.String())
	if _, err := os.Stat(blockPath); os.IsNotExist(err) {
		t.Fatal("expected block file on disk")
	}

	// Test Delete
	err = bs.Delete(hash)
	if err != nil {
		t.Fatal(err)
	}

	if bs.Has(hash) {
		t.Fatal("expected block to be deleted")
	}

	// Verify block file removed from disk
	if _, err := os.Stat(blockPath); !os.IsNotExist(err) {
		t.Fatal("expected block file to be removed from disk")
	}

	// Test Get non-existent
	_, err = bs.Get(hash)
	if err == nil {
		t.Fatal("expected error for non-existent block")
	}
}

func TestBlockstoreList(t *testing.T) {
	tmpDir := t.TempDir()
	bs := NewBlockstore(filepath.Join(tmpDir, "index.json"), filepath.Join(tmpDir, "blocks"))

	// Add multiple blocks
	hashes := []plumbing.Hash{
		plumbing.NewHash("abc123def456abc123def456abc123def456abc1"),
		plumbing.NewHash("def456abc123def456abc123def456abc123def4"),
		plumbing.NewHash("789abc123def456abc123def456abc123def4567"),
	}

	for i, hash := range hashes {
		data := []byte("content " + string(rune('A'+i)))
		err := bs.Put(hash, data)
		if err != nil {
			t.Fatal(err)
		}
	}

	// Test List
	list := bs.List()
	if len(list) != 3 {
		t.Fatalf("expected 3 blocks, got %d", len(list))
	}

	// Test Size
	if bs.Size() != 3 {
		t.Fatalf("expected size 3, got %d", bs.Size())
	}
}

func TestBlockstoreBulk(t *testing.T) {
	tmpDir := t.TempDir()
	bs := NewBlockstore(filepath.Join(tmpDir, "index.json"), filepath.Join(tmpDir, "blocks"))

	// Test PutAll
	blocks := map[plumbing.Hash][]byte{
		plumbing.NewHash("abc123def456abc123def456abc123def456abc1"): []byte("block 1"),
		plumbing.NewHash("def456abc123def456abc123def456abc123def4"): []byte("block 2"),
	}

	err := bs.PutAll(blocks)
	if err != nil {
		t.Fatal(err)
	}

	// Test GetAll
	hashes := []plumbing.Hash{
		plumbing.NewHash("abc123def456abc123def456abc123def456abc1"),
		plumbing.NewHash("def456abc123def456abc123def456abc123def4"),
	}

	result, err := bs.GetAll(hashes)
	if err != nil {
		t.Fatal(err)
	}

	if len(result) != 2 {
		t.Fatalf("expected 2 blocks, got %d", len(result))
	}
}

func TestBlockstorePersistence(t *testing.T) {
	tmpDir := t.TempDir()
	indexFile := filepath.Join(tmpDir, "index.json")
	blocksDir := filepath.Join(tmpDir, "blocks")

	// Create and populate
	bs := NewBlockstore(indexFile, blocksDir)
	hash := plumbing.NewHash("abc123def456abc123def456abc123def456abc1")
	data := []byte("persisted data")
	if err := bs.Put(hash, data); err != nil {
		t.Fatal(err)
	}

	// Reload from disk
	bs2 := NewBlockstore(indexFile, blocksDir)
	if err := bs2.Load(); err != nil {
		t.Fatal(err)
	}

	// Verify data survives reload
	retrieved, err := bs2.Get(hash)
	if err != nil {
		t.Fatal(err)
	}
	if string(retrieved) != string(data) {
		t.Fatalf("expected %q, got %q", data, retrieved)
	}
}

func TestBlockstoreMigration(t *testing.T) {
	tmpDir := t.TempDir()
	indexFile := filepath.Join(tmpDir, "blockstore.json")
	blocksDir := filepath.Join(tmpDir, "blocks")

	// Write legacy format JSON (with base64-encoded blocks, no index)
	hash := "abc123def456abc123def456abc123def456abc1"
	legacyJSON := `{
		"blocks": {
			"` + hash + `": "bGVnYWN5IGRhdGE="
		}
	}`
	os.WriteFile(indexFile, []byte(legacyJSON), 0644)

	// Load should migrate
	bs := NewBlockstore(indexFile, blocksDir)
	if err := bs.Load(); err != nil {
		t.Fatal(err)
	}

	// Verify block was migrated to file
	blockPath := filepath.Join(blocksDir, hash)
	if _, err := os.Stat(blockPath); os.IsNotExist(err) {
		t.Fatal("expected migrated block file on disk")
	}

	data, err := os.ReadFile(blockPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "legacy data" {
		t.Fatalf("expected 'legacy data', got %q", data)
	}

	// Verify index still works
	if !bs.Has(plumbing.NewHash(hash)) {
		t.Fatal("expected block in index after migration")
	}
}
