package network

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	ma "github.com/multiformats/go-multiaddr"
)

const (
	GlobalEventsTopic     = "gitant/events"
	FederationKeyPrefix   = "/gitant/federation/"
	defaultListenAddr     = "/ip4/0.0.0.0/tcp/0"
	maxRemoteEvents       = 200
	federationAnnounceTTL = 5 * time.Minute
)

// NodeConfig configures libp2p startup for gitant serve.
type NodeConfig struct {
	ListenAddr     string
	EnableMDNS     bool
	BootstrapPeers []string
	ServerDID      string
	HTTPPort       int
}

// FederatedEvent is a repo event replicated over GossipSub.
type FederatedEvent struct {
	Type       string                 `json:"type"`
	Repo       string                 `json:"repo"`
	Timestamp  time.Time              `json:"timestamp"`
	Data       map[string]interface{} `json:"data,omitempty"`
	SourceDID  string                 `json:"source_did,omitempty"`
	SourcePeer string                 `json:"source_peer,omitempty"`
}

// FederationRecord is stored in the DHT for cross-instance discovery.
type FederationRecord struct {
	DID       string    `json:"did"`
	PeerID    string    `json:"peer_id"`
	Addrs     []string  `json:"addrs"`
	HTTPPort  int       `json:"http_port"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Node orchestrates libp2p host, DHT, and GossipSub for gitant.
type Node struct {
	mu              sync.RWMutex
	Host            *Host
	DHT             *DHT
	Gossip          *GossipSub
	cfg             NodeConfig
	ctx             context.Context
	cancel          context.CancelFunc
	remoteEvents    []FederatedEvent
	subscription    *Subscription
}

// RepoEventTopic returns the GossipSub topic for a repository.
func RepoEventTopic(repoID string) string {
	return fmt.Sprintf("gitant/repo/%s/events", repoID)
}

// StartNode creates and starts the P2P stack.
func StartNode(ctx context.Context, cfg NodeConfig) (*Node, error) {
	if cfg.ListenAddr == "" {
		cfg.ListenAddr = defaultListenAddr
	}

	ctx, cancel := context.WithCancel(ctx)

	host, err := NewHost(ctx, &Config{
		ListenAddr: cfg.ListenAddr,
		EnableMDNS: cfg.EnableMDNS,
	})
	if err != nil {
		cancel()
		return nil, fmt.Errorf("starting libp2p host: %w", err)
	}

	dht, err := NewDHT(ctx, host)
	if err != nil {
		host.Close()
		cancel()
		return nil, fmt.Errorf("starting DHT: %w", err)
	}

	gossip, err := NewGossipSub(ctx, host)
	if err != nil {
		dht.Close()
		host.Close()
		cancel()
		return nil, fmt.Errorf("starting GossipSub: %w", err)
	}

	node := &Node{
		Host:   host,
		DHT:    dht,
		Gossip: gossip,
		cfg:    cfg,
		ctx:    ctx,
		cancel: cancel,
	}

	for _, addr := range cfg.BootstrapPeers {
		if addr == "" {
			continue
		}
		connectCtx, connectCancel := context.WithTimeout(ctx, 15*time.Second)
		if err := host.Connect(connectCtx, addr); err != nil {
			slog.Warn("bootstrap peer connect failed", "addr", addr, "error", err)
		}
		connectCancel()
	}

	if err := node.startEventSubscriber(); err != nil {
		node.Close()
		return nil, err
	}

	if cfg.ServerDID != "" {
		go node.federationAnnounceLoop()
	}

	slog.Info("P2P network started",
		"peer_id", host.ID().String(),
		"listen_addrs", node.AdvertisedAddrs(),
		"mdns", cfg.EnableMDNS,
		"bootstrap_peers", len(cfg.BootstrapPeers),
	)

	return node, nil
}

func (n *Node) startEventSubscriber() error {
	sub, err := n.Gossip.Subscribe(GlobalEventsTopic)
	if err != nil {
		return fmt.Errorf("subscribing to global events: %w", err)
	}
	n.subscription = sub

	go func() {
		for {
			msg, err := sub.Next(n.ctx)
			if err != nil {
				if n.ctx.Err() != nil {
					return
				}
				slog.Debug("gossip subscription error", "error", err)
				continue
			}

			var event FederatedEvent
			if err := json.Unmarshal(msg.Data, &event); err != nil {
				slog.Debug("invalid federated event payload", "error", err)
				continue
			}
			if event.SourcePeer == n.Host.ID().String() {
				continue
			}
			if event.SourcePeer == "" {
				event.SourcePeer = msg.From.String()
			}

			n.mu.Lock()
			n.remoteEvents = append(n.remoteEvents, event)
			if len(n.remoteEvents) > maxRemoteEvents {
				n.remoteEvents = n.remoteEvents[len(n.remoteEvents)-maxRemoteEvents:]
			}
			n.mu.Unlock()

			slog.Info("federated event received",
				"type", event.Type,
				"repo", event.Repo,
				"from", event.SourcePeer,
			)
		}
	}()

	return nil
}

func (n *Node) federationAnnounceLoop() {
	ticker := time.NewTicker(federationAnnounceTTL)
	defer ticker.Stop()

	n.announceFederation(context.Background())
	for {
		select {
		case <-n.ctx.Done():
			return
		case <-ticker.C:
			n.announceFederation(context.Background())
		}
	}
}

// AnnounceFederation publishes this node's federation record to the DHT.
func (n *Node) AnnounceFederation(ctx context.Context) error {
	return n.announceFederation(ctx)
}

func (n *Node) announceFederation(ctx context.Context) error {
	if n == nil || n.DHT == nil || n.cfg.ServerDID == "" {
		return nil
	}

	record := FederationRecord{
		DID:       n.cfg.ServerDID,
		PeerID:    n.Host.ID().String(),
		Addrs:     n.AdvertisedAddrs(),
		HTTPPort:  n.cfg.HTTPPort,
		UpdatedAt: time.Now().UTC(),
	}

	data, err := json.Marshal(record)
	if err != nil {
		return err
	}

	key := FederationKeyPrefix + n.cfg.ServerDID
	if err := n.DHT.PutValue(ctx, key, data); err != nil {
		return fmt.Errorf("DHT federation announce: %w", err)
	}
	return nil
}

// LookupFederation fetches a federation record for a DID from the DHT.
func (n *Node) LookupFederation(ctx context.Context, did string) (*FederationRecord, error) {
	if n == nil || n.DHT == nil {
		return nil, fmt.Errorf("P2P not enabled")
	}
	data, err := n.DHT.GetValue(ctx, FederationKeyPrefix+did)
	if err != nil {
		return nil, err
	}
	var record FederationRecord
	if err := json.Unmarshal(data, &record); err != nil {
		return nil, fmt.Errorf("decoding federation record: %w", err)
	}
	return &record, nil
}

// DiscoverFederation re-announces this node and returns known federation records.
func (n *Node) DiscoverFederation(ctx context.Context, queryDID string) ([]FederationRecord, error) {
	if n == nil {
		return nil, fmt.Errorf("P2P not enabled")
	}

	if err := n.AnnounceFederation(ctx); err != nil {
		slog.Warn("federation self-announce failed", "error", err)
	}

	records := []FederationRecord{n.LocalFederationRecord()}

	if queryDID != "" && queryDID != n.cfg.ServerDID {
		if remote, err := n.LookupFederation(ctx, queryDID); err == nil && remote != nil {
			records = append(records, *remote)
		}
	}

	for _, pid := range n.Host.Peers() {
		if info, err := n.peerFederationRecord(pid); err == nil {
			records = appendIfMissingFederation(records, info)
		}
	}

	return records, nil
}

func appendIfMissingFederation(records []FederationRecord, record FederationRecord) []FederationRecord {
	for _, existing := range records {
		if existing.PeerID == record.PeerID || existing.DID == record.DID {
			return records
		}
	}
	return append(records, record)
}

func (n *Node) peerFederationRecord(pid peer.ID) (FederationRecord, error) {
	info, err := n.Host.PeerInfo(pid)
	if err != nil {
		return FederationRecord{}, err
	}
	addrs := make([]string, 0, len(info.Addrs))
	for _, addr := range info.Addrs {
		full := addr.Encapsulate(ma.StringCast("/p2p/" + pid.String()))
		addrs = append(addrs, full.String())
	}
	return FederationRecord{
		PeerID:    pid.String(),
		Addrs:     addrs,
		UpdatedAt: time.Now().UTC(),
	}, nil
}

// LocalFederationRecord returns this node's federation record.
func (n *Node) LocalFederationRecord() FederationRecord {
	return FederationRecord{
		DID:       n.cfg.ServerDID,
		PeerID:    n.Host.ID().String(),
		Addrs:     n.AdvertisedAddrs(),
		HTTPPort:  n.cfg.HTTPPort,
		UpdatedAt: time.Now().UTC(),
	}
}

// PublishRepoEvent broadcasts an event on global and repo-specific topics.
func (n *Node) PublishRepoEvent(repoID string, event FederatedEvent) error {
	if n == nil || n.Gossip == nil || repoID == "" {
		return nil
	}

	event.SourceDID = n.cfg.ServerDID
	event.SourcePeer = n.Host.ID().String()
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}
	if event.Repo == "" {
		event.Repo = repoID
	}

	data, err := json.Marshal(event)
	if err != nil {
		return err
	}

	if err := n.Gossip.Publish(GlobalEventsTopic, data); err != nil {
		slog.Warn("gossip publish failed", "topic", GlobalEventsTopic, "error", err)
	}
	if err := n.Gossip.Publish(RepoEventTopic(repoID), data); err != nil {
		return fmt.Errorf("gossip publish repo topic: %w", err)
	}
	return nil
}

// ProvideRepoHead stores the latest ref head in the DHT for WAN discovery.
func (n *Node) ProvideRepoHead(ctx context.Context, repoID, headHash string) {
	if n == nil || n.DHT == nil || repoID == "" || headHash == "" {
		return
	}
	key := fmt.Sprintf("/gitant/repo/%s/head", repoID)
	if err := n.DHT.PutValue(ctx, key, []byte(headHash)); err != nil {
		slog.Debug("DHT repo head provide failed", "repo", repoID, "error", err)
	}
}

// PeerCount returns the number of connected libp2p peers.
func (n *Node) PeerCount() int {
	if n == nil || n.Host == nil {
		return 0
	}
	return len(n.Host.Peers())
}

// PeerSummaries returns connected peer IDs and multiaddrs.
func (n *Node) PeerSummaries() []map[string]interface{} {
	if n == nil || n.Host == nil {
		return nil
	}

	peers := n.Host.Peers()
	out := make([]map[string]interface{}, 0, len(peers))
	for _, pid := range peers {
		entry := map[string]interface{}{
			"peer_id": pid.String(),
		}
		if info, err := n.peerFederationRecord(pid); err == nil {
			entry["addrs"] = info.Addrs
		}
		out = append(out, entry)
	}
	return out
}

// AdvertisedAddrs returns dialable multiaddrs including the /p2p/ suffix.
func (n *Node) AdvertisedAddrs() []string {
	if n == nil || n.Host == nil {
		return nil
	}

	pid := n.Host.ID()
	addrs := n.Host.Addrs()
	out := make([]string, 0, len(addrs))
	for _, addr := range addrs {
		full := addr.Encapsulate(ma.StringCast("/p2p/" + pid.String()))
		out = append(out, full.String())
	}
	return out
}

// RemoteEvents returns recently received federated events from peers.
func (n *Node) RemoteEvents(limit int) []FederatedEvent {
	if limit <= 0 {
		limit = 50
	}
	n.mu.RLock()
	defer n.mu.RUnlock()

	if len(n.remoteEvents) <= limit {
		out := make([]FederatedEvent, len(n.remoteEvents))
		copy(out, n.remoteEvents)
		return out
	}
	out := make([]FederatedEvent, limit)
	copy(out, n.remoteEvents[len(n.remoteEvents)-limit:])
	return out
}

// Close shuts down gossip, DHT, and the libp2p host.
func (n *Node) Close() error {
	if n == nil {
		return nil
	}
	n.cancel()
	if n.subscription != nil {
		n.subscription.Cancel()
	}
	var firstErr error
	closeStep := func(name string, fn func() error) {
		if fn == nil {
			return
		}
		if err := fn(); err != nil && firstErr == nil {
			firstErr = fmt.Errorf("%s: %w", name, err)
		}
	}
	closeStep("gossip", n.Gossip.Close)
	closeStep("dht", n.DHT.Close)
	closeStep("host", n.Host.Close)
	return firstErr
}
