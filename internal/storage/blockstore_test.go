package storage

import (
	"testing"

	"github.com/go-git/go-git/v6/plumbing"
)

func TestBlockstore(t *testing.T) {
	bs := NewBlockstore()

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

	// Test Delete
	err = bs.Delete(hash)
	if err != nil {
		t.Fatal(err)
	}

	if bs.Has(hash) {
		t.Fatal("expected block to be deleted")
	}

	// Test Get non-existent
	_, err = bs.Get(hash)
	if err == nil {
		t.Fatal("expected error for non-existent block")
	}
}

func TestBlockstoreList(t *testing.T) {
	bs := NewBlockstore()

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
	bs := NewBlockstore()

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
