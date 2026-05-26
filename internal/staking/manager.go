package staking

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Tier represents a staking tier
type Tier int

const (
	TierObserver  Tier = iota // 0 stake
	TierLight                 // 1,000+ tokens
	TierFull                  // 10,000+ tokens
	TierValidator             // 100,000+ tokens
)

func (t Tier) String() string {
	switch t {
	case TierLight:
		return "light"
	case TierFull:
		return "full"
	case TierValidator:
		return "validator"
	default:
		return "observer"
	}
}

func (t Tier) Multiplier() uint64 {
	switch t {
	case TierValidator:
		return 8
	case TierFull:
		return 4
	case TierLight:
		return 1
	default:
		return 1
	}
}

// StakeInfo represents a user's staking info
type StakeInfo struct {
	Address    string    `json:"address"`
	Amount     uint64    `json:"amount"`
	Tier       Tier      `json:"tier"`
	StakedAt   time.Time `json:"staked_at"`
	Active     bool      `json:"active"`
	RewardDebt uint64    `json:"reward_debt"`
}

// NodeInfo represents a staked node
type NodeInfo struct {
	ID             string    `json:"id"`
	Operator       string    `json:"operator"`
	Stake          uint64    `json:"stake"`
	LastHeartbeat  time.Time `json:"last_heartbeat"`
	Multiaddr      string    `json:"multiaddr"`
	Active         bool      `json:"active"`
	SlashCount     int       `json:"slash_count"`
	TotalSlashed   uint64    `json:"total_slashed"`
}

// SlashingReason represents why a node was slashed
type SlashingReason string

const (
	SlashCorruptObjects  SlashingReason = "corrupt_objects"
	SlashCensorship      SlashingReason = "censorship"
	SlashDowntime        SlashingReason = "downtime"
	SlashInvalidCerts    SlashingReason = "invalid_certs"
	SlashDoubleSign      SlashingReason = "double_sign"
)

// SlashingSeverity represents the severity of slashing
type SlashingSeverity int

const (
	SlashLight   SlashingSeverity = 10  // 10%
	SlashMedium  SlashingSeverity = 50  // 50%
	SlashHeavy   SlashingSeverity = 100 // 100%
)

// Manager manages staking operations
type Manager struct {
	mu            sync.RWMutex
	stakes        map[string]*StakeInfo
	nodes         map[string]*NodeInfo
	epochDuration time.Duration
	lastEpoch     time.Time
}

// NewManager creates a new staking manager
func NewManager(epochDuration time.Duration) *Manager {
	return &Manager{
		stakes:        make(map[string]*StakeInfo),
		nodes:         make(map[string]*NodeInfo),
		epochDuration: epochDuration,
		lastEpoch:     time.Now(),
	}
}

// Stake tokens
func (m *Manager) Stake(ctx context.Context, address string, amount uint64) (*StakeInfo, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	info, exists := m.stakes[address]
	if !exists {
		info = &StakeInfo{
			Address:  address,
			StakedAt: time.Now(),
		}
		m.stakes[address] = info
	}

	info.Amount += amount
	info.Tier = calculateTier(info.Amount)
	info.Active = true

	return info, nil
}

// Unstake tokens
func (m *Manager) Unstake(ctx context.Context, address string, amount uint64) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	info, exists := m.stakes[address]
	if !exists {
		return fmt.Errorf("no stake found for %s", address)
	}
	if info.Amount < amount {
		return fmt.Errorf("insufficient stake: have %d, want %d", info.Amount, amount)
	}

	info.Amount -= amount
	info.Tier = calculateTier(info.Amount)
	if info.Amount == 0 {
		info.Active = false
	}

	return nil
}

// RegisterNode registers a staked node
func (m *Manager) RegisterNode(ctx context.Context, id, operator, multiaddr string, stake uint64) (*NodeInfo, error) {
	if stake < 10000 {
		return nil, fmt.Errorf("minimum stake is 10000, got %d", stake)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.nodes[id]; exists {
		return nil, fmt.Errorf("node %s already registered", id)
	}

	node := &NodeInfo{
		ID:            id,
		Operator:      operator,
		Stake:         stake,
		LastHeartbeat: time.Now(),
		Multiaddr:     multiaddr,
		Active:        true,
	}
	m.nodes[id] = node

	return node, nil
}

// Heartbeat updates node liveness
func (m *Manager) Heartbeat(ctx context.Context, nodeID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	node, exists := m.nodes[nodeID]
	if !exists {
		return fmt.Errorf("node %s not found", nodeID)
	}

	node.LastHeartbeat = time.Now()
	return nil
}

// Slash a node for misbehavior
func (m *Manager) Slash(ctx context.Context, nodeID string, severity SlashingSeverity, reason SlashingReason) (uint64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	node, exists := m.nodes[nodeID]
	if !exists {
		return 0, fmt.Errorf("node %s not found", nodeID)
	}

	slashAmount := (node.Stake * uint64(severity)) / 100
	node.Stake -= slashAmount
	node.TotalSlashed += slashAmount
	node.SlashCount++

	if node.Stake < 10000 {
		node.Active = false
	}

	return slashAmount, nil
}

// GetStaleNodes returns nodes that missed heartbeat
func (m *Manager) GetStaleNodes(ctx context.Context, timeout time.Duration) []*NodeInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var stale []*NodeInfo
	cutoff := time.Now().Add(-timeout)

	for _, node := range m.nodes {
		if node.Active && node.LastHeartbeat.Before(cutoff) {
			stale = append(stale, node)
		}
	}

	return stale
}

// GetStakeInfo returns staking info for an address
func (m *Manager) GetStakeInfo(address string) *StakeInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.stakes[address]
}

// GetNodeInfo returns node info
func (m *Manager) GetNodeInfo(nodeID string) *NodeInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.nodes[nodeID]
}

// GetActiveNodes returns all active nodes
func (m *Manager) GetActiveNodes() []*NodeInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var nodes []*NodeInfo
	for _, node := range m.nodes {
		if node.Active {
			nodes = append(nodes, node)
		}
	}
	return nodes
}

// GetTotalStaked returns total staked amount
func (m *Manager) GetTotalStaked() uint64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var total uint64
	for _, info := range m.stakes {
		if info.Active {
			total += info.Amount
		}
	}
	return total
}

func calculateTier(amount uint64) Tier {
	if amount >= 100000 {
		return TierValidator
	}
	if amount >= 10000 {
		return TierFull
	}
	if amount >= 1000 {
		return TierLight
	}
	return TierObserver
}
