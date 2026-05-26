package network

import (
	"context"
	"testing"
	"time"
)

func TestStartNode(t *testing.T) {
	ctx := context.Background()
	node, err := StartNode(ctx, NodeConfig{
		ListenAddr: "/ip4/127.0.0.1/tcp/0",
		EnableMDNS: false,
		ServerDID:  "did:key:test",
		HTTPPort:   7777,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer node.Close()

	if node.PeerCount() != 0 {
		t.Fatalf("expected 0 peers, got %d", node.PeerCount())
	}
	if len(node.AdvertisedAddrs()) == 0 {
		t.Fatal("expected advertised addrs")
	}

	event := FederatedEvent{Type: "push", Repo: "demo"}
	if err := node.PublishRepoEvent("demo", event); err != nil {
		t.Fatal(err)
	}

	record := node.LocalFederationRecord()
	if record.DID != "did:key:test" {
		t.Fatalf("unexpected federation DID: %s", record.DID)
	}
}

func TestNodeConnectAndGossip(t *testing.T) {
	ctx := context.Background()

	node1, err := StartNode(ctx, NodeConfig{
		ListenAddr: "/ip4/127.0.0.1/tcp/0",
		EnableMDNS: false,
		ServerDID:  "did:key:node1",
	})
	if err != nil {
		t.Fatal(err)
	}
	defer node1.Close()

	node2, err := StartNode(ctx, NodeConfig{
		ListenAddr: "/ip4/127.0.0.1/tcp/0",
		EnableMDNS: false,
		ServerDID:  "did:key:node2",
	})
	if err != nil {
		t.Fatal(err)
	}
	defer node2.Close()

	addr := node2.AdvertisedAddrs()[0]
	if err := node1.Host.Connect(ctx, addr); err != nil {
		t.Fatal(err)
	}

	time.Sleep(200 * time.Millisecond)
	if node1.PeerCount() == 0 {
		t.Fatal("expected node1 to connect to node2")
	}

	if err := node1.PublishRepoEvent("demo", FederatedEvent{Type: "push", Repo: "demo"}); err != nil {
		t.Fatal(err)
	}
}
