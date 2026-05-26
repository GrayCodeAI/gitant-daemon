package ipfs

import (
	"testing"
)

func TestPinningStorePinsGitObject(t *testing.T) {
	store := NewPinningStore()
	cid, err := store.PinGitObject(t.Context(), "demo", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", []byte("hello"))
	if err != nil {
		t.Fatal(err)
	}
	if cid == "" {
		t.Fatal("expected cid")
	}
	if !store.IsPinned("demo", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa") {
		t.Fatal("expected pinned object")
	}
	if store.PinCount() != 1 {
		t.Fatalf("expected 1 pin, got %d", store.PinCount())
	}
}
