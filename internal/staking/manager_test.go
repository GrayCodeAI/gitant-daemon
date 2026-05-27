package staking

import (
	"context"
	"testing"
	"time"
)

func TestManager_Stake(t *testing.T) {
	mgr := NewManager(7 * 24 * time.Hour)

	info, err := mgr.Stake(context.Background(), "addr1", 5000)
	if err != nil {
		t.Fatalf("Stake failed: %v", err)
	}

	if info.Amount != 5000 {
		t.Errorf("Expected amount 5000, got %d", info.Amount)
	}
	if info.Tier != TierLight {
		t.Errorf("Expected tier Light, got %v", info.Tier)
	}
	if !info.Active {
		t.Error("Expected stake to be active")
	}
}

func TestManager_Stake_Tiers(t *testing.T) {
	mgr := NewManager(7 * 24 * time.Hour)

	tests := []struct {
		amount uint64
		tier   Tier
	}{
		{0, TierObserver},
		{500, TierObserver},
		{1000, TierLight},
		{5000, TierLight},
		{10000, TierFull},
		{50000, TierFull},
		{100000, TierValidator},
		{500000, TierValidator},
	}

	for _, tt := range tests {
		info, err := mgr.Stake(context.Background(), "addr", tt.amount)
		if err != nil {
			t.Fatalf("Stake(%d) failed: %v", tt.amount, err)
		}
		if info.Tier != tt.tier {
			t.Errorf("Stake(%d): expected tier %v, got %v", tt.amount, tt.tier, info.Tier)
		}
	}
}

func TestManager_Unstake(t *testing.T) {
	mgr := NewManager(7 * 24 * time.Hour)

	mgr.Stake(context.Background(), "addr1", 5000)

	err := mgr.Unstake(context.Background(), "addr1", 2000)
	if err != nil {
		t.Fatalf("Unstake failed: %v", err)
	}

	info := mgr.GetStakeInfo("addr1")
	if info.Amount != 3000 {
		t.Errorf("Expected amount 3000, got %d", info.Amount)
	}
}

func TestManager_Unstake_Insufficient(t *testing.T) {
	mgr := NewManager(7 * 24 * time.Hour)

	mgr.Stake(context.Background(), "addr1", 1000)

	err := mgr.Unstake(context.Background(), "addr1", 2000)
	if err == nil {
		t.Error("Expected error for insufficient stake")
	}
}

func TestManager_RegisterNode(t *testing.T) {
	mgr := NewManager(7 * 24 * time.Hour)

	node, err := mgr.RegisterNode(context.Background(), "node1", "addr1", "/ip4/127.0.0.1/tcp/4001", 10000)
	if err != nil {
		t.Fatalf("RegisterNode failed: %v", err)
	}

	if node.ID != "node1" {
		t.Errorf("Expected node ID 'node1', got '%s'", node.ID)
	}
	if !node.Active {
		t.Error("Expected node to be active")
	}
}

func TestManager_RegisterNode_InsufficientStake(t *testing.T) {
	mgr := NewManager(7 * 24 * time.Hour)

	_, err := mgr.RegisterNode(context.Background(), "node1", "addr1", "/ip4/127.0.0.1/tcp/4001", 5000)
	if err == nil {
		t.Error("Expected error for insufficient stake")
	}
}

func TestManager_Heartbeat(t *testing.T) {
	mgr := NewManager(7 * 24 * time.Hour)

	mgr.RegisterNode(context.Background(), "node1", "addr1", "/ip4/127.0.0.1/tcp/4001", 10000)

	err := mgr.Heartbeat(context.Background(), "node1")
	if err != nil {
		t.Fatalf("Heartbeat failed: %v", err)
	}
}

func TestManager_Slash(t *testing.T) {
	mgr := NewManager(7 * 24 * time.Hour)

	mgr.RegisterNode(context.Background(), "node1", "addr1", "/ip4/127.0.0.1/tcp/4001", 10000)

	slashed, err := mgr.Slash(context.Background(), "node1", SlashLight, SlashDowntime)
	if err != nil {
		t.Fatalf("Slash failed: %v", err)
	}

	if slashed != 1000 {
		t.Errorf("Expected slashed amount 1000, got %d", slashed)
	}

	node := mgr.GetNodeInfo("node1")
	if node.Stake != 9000 {
		t.Errorf("Expected stake 9000, got %d", node.Stake)
	}
}

func TestManager_GetStaleNodes(t *testing.T) {
	mgr := NewManager(7 * 24 * time.Hour)

	mgr.RegisterNode(context.Background(), "node1", "addr1", "/ip4/127.0.0.1/tcp/4001", 10000)

	// No stale nodes immediately
	stale := mgr.GetStaleNodes(context.Background(), 1*time.Hour)
	if len(stale) != 0 {
		t.Errorf("Expected 0 stale nodes, got %d", len(stale))
	}
}

func TestManager_GetActiveNodes(t *testing.T) {
	mgr := NewManager(7 * 24 * time.Hour)

	mgr.RegisterNode(context.Background(), "node1", "addr1", "/ip4/127.0.0.1/tcp/4001", 10000)
	mgr.RegisterNode(context.Background(), "node2", "addr2", "/ip4/127.0.0.1/tcp/4002", 20000)

	nodes := mgr.GetActiveNodes()
	if len(nodes) != 2 {
		t.Errorf("Expected 2 active nodes, got %d", len(nodes))
	}
}

func TestManager_GetTotalStaked(t *testing.T) {
	mgr := NewManager(7 * 24 * time.Hour)

	mgr.Stake(context.Background(), "addr1", 5000)
	mgr.Stake(context.Background(), "addr2", 10000)

	total := mgr.GetTotalStaked()
	if total != 15000 {
		t.Errorf("Expected total staked 15000, got %d", total)
	}
}
