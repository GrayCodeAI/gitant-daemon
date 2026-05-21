package ipfs

import (
	"context"
	"testing"

	"github.com/ipfs/go-cid"
)

func TestNode(t *testing.T) {
	ctx := context.Background()

	cfg := &Config{
		ListenAddr: "/ip4/127.0.0.1/tcp/0",
	}

	node, err := NewNode(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer node.Close()

	// Test PutBlock
	data := []byte("Hello, IPFS!")
	c, err := node.PutBlock(ctx, data)
	if err != nil {
		t.Fatal(err)
	}

	if !c.Defined() {
		t.Fatal("expected defined CID")
	}

	// Test GetBlock
	retrieved, err := node.GetBlock(ctx, c)
	if err != nil {
		t.Fatal(err)
	}

	if string(retrieved) != string(data) {
		t.Fatalf("expected %q, got %q", data, retrieved)
	}

	// Test Peers
	peers := node.Peers()
	if peers == nil {
		t.Fatal("expected non-nil peers slice")
	}
}

func TestGitDAG(t *testing.T) {
	ctx := context.Background()

	cfg := &Config{
		ListenAddr: "/ip4/127.0.0.1/tcp/0",
	}

	node, err := NewNode(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer node.Close()

	dag := NewGitDAG(node)

	// Test PutGitObject
	data := []byte("git object content")
	c, err := dag.PutGitObject(ctx, 0, data)
	if err != nil {
		t.Fatal(err)
	}

	if !c.Defined() {
		t.Fatal("expected defined CID")
	}

	// Test GetGitObject
	retrieved, err := dag.GetGitObject(ctx, c)
	if err != nil {
		t.Fatal(err)
	}

	if string(retrieved) != string(data) {
		t.Fatalf("expected %q, got %q", data, retrieved)
	}
}

func TestPinManager(t *testing.T) {
	ctx := context.Background()

	cfg := &Config{
		ListenAddr: "/ip4/127.0.0.1/tcp/0",
	}

	node, err := NewNode(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer node.Close()

	pm := NewPinManager(node)

	// Test Pin
	data := []byte("pinned content")
	c, err := node.PutBlock(ctx, data)
	if err != nil {
		t.Fatal(err)
	}

	err = pm.Pin(ctx, c)
	if err != nil {
		t.Fatal(err)
	}

	// Test IsPinned
	if !pm.IsPinned(c) {
		t.Fatal("expected CID to be pinned")
	}

	// Test ListPins
	pins := pm.ListPins()
	if len(pins) != 1 {
		t.Fatalf("expected 1 pin, got %d", len(pins))
	}

	// Test Unpin
	err = pm.Unpin(ctx, c)
	if err != nil {
		t.Fatal(err)
	}

	if pm.IsPinned(c) {
		t.Fatal("expected CID to be unpinned")
	}
}

func TestPinRepo(t *testing.T) {
	ctx := context.Background()

	cfg := &Config{
		ListenAddr: "/ip4/127.0.0.1/tcp/0",
	}

	node, err := NewNode(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer node.Close()

	pm := NewPinManager(node)

	// Create test CIDs
	data1 := []byte("object 1")
	data2 := []byte("object 2")

	c1, err := node.PutBlock(ctx, data1)
	if err != nil {
		t.Fatal(err)
	}

	c2, err := node.PutBlock(ctx, data2)
	if err != nil {
		t.Fatal(err)
	}

	// Pin repo
	err = pm.PinRepo(ctx, "test-repo", []cid.Cid{c1, c2})
	if err != nil {
		t.Fatal(err)
	}

	// List repo pins
	pins := pm.ListRepoPins("test-repo")
	if len(pins) != 2 {
		t.Fatalf("expected 2 pins, got %d", len(pins))
	}

	// Unpin repo
	err = pm.UnpinRepo(ctx, "test-repo")
	if err != nil {
		t.Fatal(err)
	}

	// Verify unpinned
	pins = pm.ListRepoPins("test-repo")
	if len(pins) != 0 {
		t.Fatalf("expected 0 pins, got %d", len(pins))
	}
}
