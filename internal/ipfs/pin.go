package ipfs

import (
	"context"
	"fmt"
	"sync"

	"github.com/ipfs/go-cid"
)

// PinManager manages pinned content in IPFS
type PinManager struct {
	mu    sync.RWMutex
	pins  map[string]cid.Cid
	node  *Node
}

// NewPinManager creates a new PinManager
func NewPinManager(node *Node) *PinManager {
	return &PinManager{
		pins: make(map[string]cid.Cid),
		node: node,
	}
}

// Pin pins a CID
func (pm *PinManager) Pin(ctx context.Context, c cid.Cid) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	key := c.String()
	pm.pins[key] = c

	return nil
}

// Unpin unpins a CID
func (pm *PinManager) Unpin(ctx context.Context, c cid.Cid) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	key := c.String()
	delete(pm.pins, key)

	return nil
}

// IsPinned checks if a CID is pinned
func (pm *PinManager) IsPinned(c cid.Cid) bool {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	_, ok := pm.pins[c.String()]
	return ok
}

// ListPins returns all pinned CIDs
func (pm *PinManager) ListPins() []cid.Cid {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	pins := make([]cid.Cid, 0, len(pm.pins))
	for _, c := range pm.pins {
		pins = append(pins, c)
	}

	return pins
}

// PinRepo pins all objects in a repository
func (pm *PinManager) PinRepo(ctx context.Context, repoID string, cids []cid.Cid) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	for _, c := range cids {
		key := fmt.Sprintf("%s/%s", repoID, c.String())
		pm.pins[key] = c
	}

	return nil
}

// UnpinRepo unpins all objects in a repository
func (pm *PinManager) UnpinRepo(ctx context.Context, repoID string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	prefix := repoID + "/"
	for key := range pm.pins {
		if len(key) > len(prefix) && key[:len(prefix)] == prefix {
			delete(pm.pins, key)
		}
	}

	return nil
}

// ListRepoPins returns all pinned CIDs for a repository
func (pm *PinManager) ListRepoPins(repoID string) []cid.Cid {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	prefix := repoID + "/"
	var pins []cid.Cid
	for key, c := range pm.pins {
		if len(key) > len(prefix) && key[:len(prefix)] == prefix {
			pins = append(pins, c)
		}
	}

	return pins
}
