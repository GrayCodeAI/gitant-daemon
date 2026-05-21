package storage

import (
	"fmt"
	"sync"

	"github.com/go-git/go-git/v6/plumbing"
)

// Blockstore provides content-addressed storage for git objects
// This is the bridge between git objects and IPFS blocks
type Blockstore struct {
	mu      sync.RWMutex
	blocks  map[string][]byte
	index   map[string]plumbing.Hash
}

// NewBlockstore creates a new blockstore
func NewBlockstore() *Blockstore {
	return &Blockstore{
		blocks: make(map[string][]byte),
		index:  make(map[string]plumbing.Hash),
	}
}

// Put stores a block with its content hash as key
func (bs *Blockstore) Put(hash plumbing.Hash, data []byte) error {
	bs.mu.Lock()
	defer bs.mu.Unlock()

	key := hash.String()
	bs.blocks[key] = data
	bs.index[key] = hash

	return nil
}

// Get retrieves a block by its hash
func (bs *Blockstore) Get(hash plumbing.Hash) ([]byte, error) {
	bs.mu.RLock()
	defer bs.mu.RUnlock()

	key := hash.String()
	data, ok := bs.blocks[key]
	if !ok {
		return nil, fmt.Errorf("block not found: %s", key)
	}

	return data, nil
}

// Has checks if a block exists in the store
func (bs *Blockstore) Has(hash plumbing.Hash) bool {
	bs.mu.RLock()
	defer bs.mu.RUnlock()

	_, ok := bs.blocks[hash.String()]
	return ok
}

// Delete removes a block from the store
func (bs *Blockstore) Delete(hash plumbing.Hash) error {
	bs.mu.Lock()
	defer bs.mu.Unlock()

	key := hash.String()
	delete(bs.blocks, key)
	delete(bs.index, key)

	return nil
}

// List returns all block hashes in the store
func (bs *Blockstore) List() []plumbing.Hash {
	bs.mu.RLock()
	defer bs.mu.RUnlock()

	hashes := make([]plumbing.Hash, 0, len(bs.index))
	for _, hash := range bs.index {
		hashes = append(hashes, hash)
	}

	return hashes
}

// Size returns the number of blocks in the store
func (bs *Blockstore) Size() int {
	bs.mu.RLock()
	defer bs.mu.RUnlock()

	return len(bs.blocks)
}

// PutAll stores multiple blocks
func (bs *Blockstore) PutAll(blocks map[plumbing.Hash][]byte) error {
	bs.mu.Lock()
	defer bs.mu.Unlock()

	for hash, data := range blocks {
		key := hash.String()
		bs.blocks[key] = data
		bs.index[key] = hash
	}

	return nil
}

// GetAll retrieves multiple blocks by their hashes
func (bs *Blockstore) GetAll(hashes []plumbing.Hash) (map[plumbing.Hash][]byte, error) {
	bs.mu.RLock()
	defer bs.mu.RUnlock()

	result := make(map[plumbing.Hash][]byte)
	for _, hash := range hashes {
		key := hash.String()
		data, ok := bs.blocks[key]
		if !ok {
			return nil, fmt.Errorf("block not found: %s", key)
		}
		result[hash] = data
	}

	return result, nil
}
