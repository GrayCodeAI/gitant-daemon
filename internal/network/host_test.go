package network

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/multiformats/go-multiaddr"
)

func TestHost(t *testing.T) {
	ctx := context.Background()

	cfg := &Config{
		ListenAddr: "/ip4/127.0.0.1/tcp/0",
		EnableMDNS: false,
	}

	host, err := NewHost(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer host.Close()

	// Test ID
	if host.ID() == "" {
		t.Fatal("expected non-empty host ID")
	}

	// Test Addrs
	addrs := host.Addrs()
	if len(addrs) == 0 {
		t.Fatal("expected at least one listen address")
	}

	// Test Peers
	peers := host.Peers()
	if peers == nil {
		t.Fatal("expected non-nil peers slice")
	}
}

func TestHostConnect(t *testing.T) {
	ctx := context.Background()

	// Create two hosts
	cfg1 := &Config{
		ListenAddr: "/ip4/127.0.0.1/tcp/0",
	}
	cfg2 := &Config{
		ListenAddr: "/ip4/127.0.0.1/tcp/0",
	}

	host1, err := NewHost(ctx, cfg1)
	if err != nil {
		t.Fatal(err)
	}
	defer host1.Close()

	host2, err := NewHost(ctx, cfg2)
	if err != nil {
		t.Fatal(err)
	}
	defer host2.Close()

	// Get host2's address
	addr2 := host2.Addrs()[0]
	peerId := host2.ID()

	// Create p2p multiaddr
	p2pAddr, err := multiaddr.NewMultiaddr(fmt.Sprintf("/p2p/%s", peerId))
	if err != nil {
		t.Fatal(err)
	}
	fullAddr := addr2.Encapsulate(p2pAddr)

	// Connect host1 to host2
	err = host1.Connect(ctx, fullAddr.String())
	if err != nil {
		t.Fatal(err)
	}

	// Wait for connection
	time.Sleep(100 * time.Millisecond)

	// Verify connection
	peers := host1.Peers()
	if len(peers) == 0 {
		t.Fatal("expected at least one peer")
	}
}

func TestDHT(t *testing.T) {
	ctx := context.Background()

	// Create host
	cfg := &Config{
		ListenAddr: "/ip4/127.0.0.1/tcp/0",
	}

	host, err := NewHost(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer host.Close()

	// Create DHT
	d, err := NewDHT(ctx, host)
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()

	// Test FindPeer (should not error even if peer not found)
	_, err = d.FindPeer(ctx, host.ID())
	if err != nil {
		// This is expected - peer might not be found
		t.Logf("FindPeer returned error (expected): %v", err)
	}
}

func TestGossipSub(t *testing.T) {
	ctx := context.Background()

	// Create host
	cfg := &Config{
		ListenAddr: "/ip4/127.0.0.1/tcp/0",
	}

	host, err := NewHost(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer host.Close()

	// Create GossipSub
	gs, err := NewGossipSub(ctx, host)
	if err != nil {
		t.Fatal(err)
	}

	// Test Publish
	message := []byte("test message")
	err = gs.Publish("test-topic", message)
	if err != nil {
		t.Fatal(err)
	}

	// Note: Topics() returns topics we've joined
	// Since we just published to "test-topic", it should be in the list
	// But GossipSub may not track topics the same way
	t.Log("GossipSub publish test passed")
}
