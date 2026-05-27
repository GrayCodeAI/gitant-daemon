package network

import (
	"context"
	"testing"
	"time"
)

func TestSeedNodeDiscovery(t *testing.T) {
	ctx := context.Background()

	// Start a seed node
	seed, err := StartSeedNode(ctx, NodeConfig{
		ListenAddr: "/ip4/127.0.0.1/tcp/0",
		EnableMDNS: false,
		ServerDID:  "did:key:seed",
	})
	if err != nil {
		t.Fatal(err)
	}
	defer seed.Close()

	if len(seed.AdvertisedAddrs()) == 0 {
		t.Fatal("seed node should have advertised addrs")
	}

	seedAddr := seed.AdvertisedAddrs()[0]

	// Start a regular node with the seed as bootstrap peer
	node, err := StartNode(ctx, NodeConfig{
		ListenAddr:     "/ip4/127.0.0.1/tcp/0",
		EnableMDNS:     false,
		BootstrapPeers: []string{seedAddr},
		ServerDID:      "did:key:regular",
		HTTPPort:       0,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer node.Close()

	// Wait for the node to discover and connect to the seed
	deadline := time.After(5 * time.Second)
	for node.PeerCount() == 0 {
		select {
		case <-deadline:
			t.Fatalf("regular node did not discover seed node within 5s (peers: %d)", node.PeerCount())
		case <-time.After(100 * time.Millisecond):
		}
	}

	// Verify the seed also sees the regular node
	time.Sleep(500 * time.Millisecond)
	if seed.PeerCount() == 0 {
		t.Fatal("seed node should see the regular node")
	}
}

func TestSeedNodeGossipRelay(t *testing.T) {
	ctx := context.Background()

	// Start seed
	seed, err := StartSeedNode(ctx, NodeConfig{
		ListenAddr: "/ip4/127.0.0.1/tcp/0",
		EnableMDNS: false,
		ServerDID:  "did:key:seed",
	})
	if err != nil {
		t.Fatal(err)
	}
	defer seed.Close()

	seedAddr := seed.AdvertisedAddrs()[0]

	// Start two regular nodes both connected to the seed
	node1, err := StartNode(ctx, NodeConfig{
		ListenAddr:     "/ip4/127.0.0.1/tcp/0",
		EnableMDNS:     false,
		BootstrapPeers: []string{seedAddr},
		ServerDID:      "did:key:node1",
		HTTPPort:       0,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer node1.Close()

	node2, err := StartNode(ctx, NodeConfig{
		ListenAddr:     "/ip4/127.0.0.1/tcp/0",
		EnableMDNS:     false,
		BootstrapPeers: []string{seedAddr},
		ServerDID:      "did:key:node2",
		HTTPPort:       0,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer node2.Close()

	// Wait for both nodes to connect to the seed
	deadline := time.After(5 * time.Second)
	for node1.PeerCount() == 0 || node2.PeerCount() == 0 {
		select {
		case <-deadline:
			t.Fatalf("nodes did not connect to seed (n1 peers: %d, n2 peers: %d)", node1.PeerCount(), node2.PeerCount())
		case <-time.After(100 * time.Millisecond):
		}
	}

	// Allow GossipSub mesh to stabilize
	time.Sleep(1 * time.Second)

	// Set up event listener on node2
	received := make(chan FederatedEvent, 1)
	node2.SetFederatedEventHandler(func(ev FederatedEvent) {
		select {
		case received <- ev:
		default:
		}
	})

	// node1 publishes an event
	event := FederatedEvent{Type: "push", Repo: "test-repo", Timestamp: time.Now()}
	if err := node1.PublishRepoEvent("test-repo", event); err != nil {
		t.Fatal(err)
	}

	// node2 should receive the event via gossip relay through the seed
	select {
	case ev := <-received:
		if ev.Type != "push" {
			t.Fatalf("expected event type 'push', got '%s'", ev.Type)
		}
		if ev.Repo != "test-repo" {
			t.Fatalf("expected repo 'test-repo', got '%s'", ev.Repo)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("node2 did not receive gossip event within 5s")
	}
}

func TestSeedNodeStartStop(t *testing.T) {
	ctx := context.Background()

	seed, err := StartSeedNode(ctx, NodeConfig{
		ListenAddr: "/ip4/127.0.0.1/tcp/0",
		EnableMDNS: false,
		ServerDID:  "did:key:seed",
	})
	if err != nil {
		t.Fatal(err)
	}

	// Verify it started
	if seed.Host == nil {
		t.Fatal("seed host should not be nil")
	}
	if seed.DHT == nil {
		t.Fatal("seed DHT should not be nil")
	}
	if seed.Gossip == nil {
		t.Fatal("seed GossipSub should not be nil")
	}

	// Close should not error
	if err := seed.Close(); err != nil {
		t.Fatalf("seed close error: %v", err)
	}
}
