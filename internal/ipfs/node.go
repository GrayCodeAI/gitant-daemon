package ipfs

import (
	"context"
	"fmt"
	"sync"

	"github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
	"github.com/multiformats/go-multihash"
)

// Node represents an IPFS node
type Node struct {
	mu sync.RWMutex

	host   host.Host
	blocks map[string][]byte
	ctx    context.Context
	cancel context.CancelFunc
}

// Config holds IPFS node configuration
type Config struct {
	ListenAddr string
}

// NewNode creates a new IPFS node
func NewNode(ctx context.Context, cfg *Config) (*Node, error) {
	ctx, cancel := context.WithCancel(ctx)

	// Create libp2p host
	h, err := libp2p.New(
		libp2p.ListenAddrStrings(cfg.ListenAddr),
	)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("creating libp2p host: %w", err)
	}

	return &Node{
		host:   h,
		blocks: make(map[string][]byte),
		ctx:    ctx,
		cancel: cancel,
	}, nil
}

// PutBlock stores a block in IPFS
func (n *Node) PutBlock(ctx context.Context, data []byte) (cid.Cid, error) {
	n.mu.Lock()
	defer n.mu.Unlock()

	// Create multihash from data
	mh, err := multihash.Sum(data, multihash.SHA2_256, -1)
	if err != nil {
		return cid.Undef, fmt.Errorf("creating multihash: %w", err)
	}

	// Create CID
	c := cid.NewCidV1(cid.Raw, mh)

	// Store block
	n.blocks[c.String()] = data

	return c, nil
}

// GetBlock retrieves a block from IPFS
func (n *Node) GetBlock(ctx context.Context, c cid.Cid) ([]byte, error) {
	n.mu.RLock()
	defer n.mu.RUnlock()

	data, ok := n.blocks[c.String()]
	if !ok {
		return nil, fmt.Errorf("block not found: %s", c.String())
	}

	return data, nil
}

// Connect connects to a peer
func (n *Node) Connect(ctx context.Context, addr string) error {
	maddr, err := multiaddr.NewMultiaddr(addr)
	if err != nil {
		return fmt.Errorf("parsing multiaddr: %w", err)
	}

	info, err := peer.AddrInfoFromP2pAddr(maddr)
	if err != nil {
		return fmt.Errorf("parsing peer info: %w", err)
	}

	return n.host.Connect(ctx, *info)
}

// Peers returns connected peers
func (n *Node) Peers() []peer.ID {
	return n.host.Network().Peers()
}

// Close closes the IPFS node
func (n *Node) Close() error {
	n.cancel()
	return n.host.Close()
}
