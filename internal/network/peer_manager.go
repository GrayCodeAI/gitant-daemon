package network

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

const (
	peerPingInterval    = 30 * time.Second
	peerPingTimeout     = 10 * time.Second
	peerScoreDecay      = 0.95
	peerScorePingBonus  = 1.0
	peerScoreFailPenalty = -2.0
	peerMinScore        = -10.0
	peerMaxMissedPings  = 5
)

// PeerScore tracks a peer's reliability score.
type PeerScore struct {
	MissedPings int
	Score       float64
	LastSeen    time.Time
}

// PeerManager periodically pings connected peers and reconnects bootstrap peers.
type PeerManager struct {
	mu     sync.Mutex
	node   *Node
	scores map[string]*PeerScore
	ctx    context.Context
	cancel context.CancelFunc
}

// NewPeerManager creates a PeerManager attached to a Node.
func NewPeerManager(node *Node) *PeerManager {
	ctx, cancel := context.WithCancel(node.ctx)
	pm := &PeerManager{
		node:   node,
		scores: make(map[string]*PeerScore),
		ctx:    ctx,
		cancel: cancel,
	}
	go pm.loop()
	return pm
}

// Stop terminates the peer management loop.
func (pm *PeerManager) Stop() {
	pm.cancel()
}

// Scores returns a copy of all peer scores.
func (pm *PeerManager) Scores() map[string]PeerScore {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	out := make(map[string]PeerScore, len(pm.scores))
	for id, s := range pm.scores {
		out[id] = PeerScore{
			MissedPings: s.MissedPings,
			Score:       s.Score,
			LastSeen:    s.LastSeen,
		}
	}
	return out
}

func (pm *PeerManager) loop() {
	ticker := time.NewTicker(peerPingInterval)
	defer ticker.Stop()

	pm.tick()
	for {
		select {
		case <-pm.ctx.Done():
			return
		case <-ticker.C:
			pm.tick()
		}
	}
}

func (pm *PeerManager) tick() {
	pm.pingPeers()
	pm.reconnectBootstrap()
}

func (pm *PeerManager) pingPeers() {
	peers := pm.node.Host.Peers()
	for _, pid := range peers {
		ctx, cancel := context.WithTimeout(pm.ctx, peerPingTimeout)
		_, err := pm.node.Host.Ping(ctx, pid)
		cancel()

		pm.mu.Lock()
		score, ok := pm.scores[pid.String()]
		if !ok {
			score = &PeerScore{Score: 5.0}
			pm.scores[pid.String()] = score
		}
		if err != nil {
			score.MissedPings++
			score.Score += peerScoreFailPenalty
			if score.Score < peerMinScore {
				score.Score = peerMinScore
			}
			slog.Debug("peer ping failed", "peer", pid, "missed", score.MissedPings, "err", err)
		} else {
			score.MissedPings = 0
			score.Score += peerScorePingBonus
			score.LastSeen = time.Now()
			score.Score *= peerScoreDecay
			slog.Debug("peer ping ok", "peer", pid, "rtt_score", score.Score)
		}
		pm.mu.Unlock()
	}
}

func (pm *PeerManager) reconnectBootstrap() {
	for _, addr := range pm.node.cfg.BootstrapPeers {
		if addr == "" {
			continue
		}
		ctx, cancel := context.WithTimeout(pm.ctx, 10*time.Second)
		if err := pm.node.Host.Connect(ctx, addr); err != nil {
			slog.Debug("bootstrap reconnect failed", "addr", addr, "error", err)
		}
		cancel()
	}
}
