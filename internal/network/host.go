package network

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/host"
	p2pnetwork "github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/libp2p/go-libp2p/p2p/discovery/mdns"
	ping "github.com/libp2p/go-libp2p/p2p/protocol/ping"
	"github.com/multiformats/go-multiaddr"
)

// Host represents a libp2p host with P2P networking
type Host struct {
	mu sync.RWMutex

	host    host.Host
	ctx     context.Context
	cancel  context.CancelFunc
	peers   map[peer.ID]peer.AddrInfo
}

// Config holds P2P host configuration
type Config struct {
	ListenAddr string
	EnableMDNS bool
}

// NewHost creates a new P2P host
func NewHost(ctx context.Context, cfg *Config) (*Host, error) {
	ctx, cancel := context.WithCancel(ctx)

	// Create libp2p host
	h, err := libp2p.New(
		libp2p.ListenAddrStrings(cfg.ListenAddr),
		libp2p.NATPortMap(),
		libp2p.EnableRelay(),
		libp2p.EnableHolePunching(),
	)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("creating libp2p host: %w", err)
	}

	host := &Host{
		host:   h,
		ctx:    ctx,
		cancel: cancel,
		peers:  make(map[peer.ID]peer.AddrInfo),
	}

	// Start mDNS discovery if enabled
	if cfg.EnableMDNS {
		host.startMDNS()
	}

	return host, nil
}

// Connect connects to a peer
func (h *Host) Connect(ctx context.Context, addr string) error {
	maddr, err := multiaddr.NewMultiaddr(addr)
	if err != nil {
		return fmt.Errorf("parsing multiaddr: %w", err)
	}

	info, err := peer.AddrInfoFromP2pAddr(maddr)
	if err != nil {
		return fmt.Errorf("parsing peer info: %w", err)
	}

	if err := h.host.Connect(ctx, *info); err != nil {
		return fmt.Errorf("connecting to peer: %w", err)
	}

	h.mu.Lock()
	h.peers[info.ID] = *info
	h.mu.Unlock()

	return nil
}

// Peers returns connected peers
func (h *Host) Peers() []peer.ID {
	return h.host.Network().Peers()
}

// PeerInfo returns info about a peer
func (h *Host) PeerInfo(id peer.ID) (peer.AddrInfo, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	info, ok := h.peers[id]
	if !ok {
		return peer.AddrInfo{}, fmt.Errorf("peer not found: %s", id)
	}

	return info, nil
}

// ID returns the host's peer ID
func (h *Host) ID() peer.ID {
	return h.host.ID()
}

// Addrs returns the host's listen addresses
func (h *Host) Addrs() []multiaddr.Multiaddr {
	return h.host.Addrs()
}

// SetStreamHandler registers a libp2p stream handler.
func (h *Host) SetStreamHandler(id protocol.ID, handler func(p2pnetwork.Stream)) {
	h.host.SetStreamHandler(id, handler)
}

// NewStream opens a stream to a peer for the given protocol.
func (h *Host) NewStream(ctx context.Context, peerID peer.ID, id protocol.ID) (p2pnetwork.Stream, error) {
	return h.host.NewStream(ctx, peerID, id)
}

// Ping sends a single ping to a peer and returns the RTT or an error.
func (h *Host) Ping(ctx context.Context, peerID peer.ID) (time.Duration, error) {
	result := <-ping.Ping(ctx, h.host, peerID)
	if result.Error != nil {
		return 0, result.Error
	}
	return result.RTT, nil
}

// Network returns the underlying libp2p network (for connection state checks).
func (h *Host) Network() p2pnetwork.Network {
	return h.host.Network()
}

// Close closes the host
func (h *Host) Close() error {
	h.cancel()
	return h.host.Close()
}

// discoveryNotifee handles mDNS peer discovery events
type discoveryNotifee struct {
	h host.Host
}

// HandlePeerFound is called when a peer is discovered via mDNS
func (n *discoveryNotifee) HandlePeerFound(pi peer.AddrInfo) {
	slog.Info("mDNS peer discovered", "peer", pi.ID.String(), "addrs", pi.Addrs)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := n.h.Connect(ctx, pi); err != nil {
		slog.Warn("failed to connect to mDNS peer", "peer", pi.ID.String(), "error", err)
	}
}

// startMDNS starts mDNS discovery for local network peers
func (h *Host) startMDNS() {
	notifee := &discoveryNotifee{h: h.host}
	service := mdns.NewMdnsService(h.host, "gitant", notifee)
	if err := service.Start(); err != nil {
		slog.Error("mDNS service failed to start", "error", err)
		return
	}
	slog.Info("mDNS discovery started")
}
