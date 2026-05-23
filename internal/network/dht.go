package network

import (
	"context"
	"fmt"
	"sync"

	"github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/peer"
)

// DHT wraps the Kademlia DHT for content routing
type DHT struct {
	_   sync.RWMutex
	dht  *dht.IpfsDHT
	host *Host
	ctx  context.Context
}

// NewDHT creates a new DHT
func NewDHT(ctx context.Context, host *Host) (*DHT, error) {
	// Create DHT
	d, err := dht.New(ctx, host.host, dht.Mode(dht.ModeAuto))
	if err != nil {
		return nil, fmt.Errorf("creating DHT: %w", err)
	}

	// Bootstrap DHT
	if err := d.Bootstrap(ctx); err != nil {
		return nil, fmt.Errorf("bootstrapping DHT: %w", err)
	}

	return &DHT{
		dht:  d,
		host: host,
		ctx:  ctx,
	}, nil
}

// Provide announces that this node has the content
func (d *DHT) Provide(ctx context.Context, c cid.Cid) error {
	return d.dht.Provide(ctx, c, true)
}

// FindProviders finds nodes that have the content
func (d *DHT) FindProviders(ctx context.Context, c cid.Cid) ([]peer.AddrInfo, error) {
	providers := d.dht.FindProvidersAsync(ctx, c, 20)

	var results []peer.AddrInfo
	for p := range providers {
		results = append(results, p)
	}

	return results, nil
}

// FindPeer finds a peer by ID
func (d *DHT) FindPeer(ctx context.Context, pid peer.ID) (peer.AddrInfo, error) {
	return d.dht.FindPeer(ctx, pid)
}

// PutValue stores a value in the DHT
func (d *DHT) PutValue(ctx context.Context, key string, value []byte) error {
	return d.dht.PutValue(ctx, key, value)
}

// GetValue retrieves a value from the DHT
func (d *DHT) GetValue(ctx context.Context, key string) ([]byte, error) {
	return d.dht.GetValue(ctx, key)
}

// Close closes the DHT
func (d *DHT) Close() error {
	return d.dht.Close()
}
