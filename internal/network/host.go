package network

import (
	"context"
	"fmt"
	"sync"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
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

// Close closes the host
func (h *Host) Close() error {
	h.cancel()
	return h.host.Close()
}

// startMDNS starts mDNS discovery
func (h *Host) startMDNS() {
	// TODO: Implement mDNS discovery
}
