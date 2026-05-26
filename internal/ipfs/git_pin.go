package ipfs

import (
	"context"
	"fmt"
	"sync"

	"github.com/ipfs/go-cid"
	"github.com/multiformats/go-multihash"
)

// PinningStore tracks git object bytes with content-addressed CIDs (no extra libp2p host).
type PinningStore struct {
	mu     sync.RWMutex
	blocks map[string][]byte
	pins   map[string]cid.Cid
}

// NewPinningStore creates an in-process pinning adapter for warm git object storage.
func NewPinningStore() *PinningStore {
	return &PinningStore{
		blocks: make(map[string][]byte),
		pins:   make(map[string]cid.Cid),
	}
}

// PinGitObject stores and pins a git object, returning its CID string.
func (s *PinningStore) PinGitObject(ctx context.Context, repoID, hash string, data []byte) (string, error) {
	if len(data) == 0 {
		return "", fmt.Errorf("empty object data")
	}

	mh, err := multihash.Sum(data, multihash.SHA2_256, -1)
	if err != nil {
		return "", fmt.Errorf("creating multihash: %w", err)
	}
	c := cid.NewCidV1(cid.Raw, mh)

	s.mu.Lock()
	defer s.mu.Unlock()

	s.blocks[c.String()] = append([]byte(nil), data...)
	s.pins[pinKey(repoID, hash)] = c
	return c.String(), nil
}

// IsPinned reports whether a repo object is pinned.
func (s *PinningStore) IsPinned(repoID, hash string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.pins[pinKey(repoID, hash)]
	return ok
}

// PinnedCID returns the CID for a pinned git object, if present.
func (s *PinningStore) PinnedCID(repoID, hash string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	c, ok := s.pins[pinKey(repoID, hash)]
	if !ok {
		return "", false
	}
	return c.String(), true
}

// PinCount returns the number of pinned git objects.
func (s *PinningStore) PinCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.pins)
}

func pinKey(repoID, hash string) string {
	return repoID + "/" + hash
}
