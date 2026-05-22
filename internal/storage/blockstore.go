package storage

import (
	"encoding/base64"
	"fmt"
	"sync"

	"github.com/go-git/go-git/v6/plumbing"
	"github.com/lakshmanpatel/gitant/internal/persistence"
)

// blockstoreData is the JSON-serializable representation
type blockstoreData struct {
	Blocks map[string]string    `json:"blocks"` // hash -> base64 content
	Index  map[string]plumbing.Hash `json:"index"`
}

// Blockstore provides content-addressed storage for git objects
// This is the bridge between git objects and IPFS blocks
type Blockstore struct {
	mu      sync.RWMutex
	blocks  map[string][]byte
	index   map[string]plumbing.Hash
	path    string // persistence file path
}

// NewBlockstore creates a new blockstore
func NewBlockstore(path string) *Blockstore {
	return &Blockstore{
		blocks: make(map[string][]byte),
		index:  make(map[string]plumbing.Hash),
		path:   path,
	}
}

// Load reads persisted blocks from disk
func (bs *Blockstore) Load() error {
	if bs.path == "" {
		return nil
	}
	bs.mu.Lock()
	defer bs.mu.Unlock()

	var data blockstoreData
	if err := persistence.LoadJSON(bs.path, &data); err != nil {
		return err
	}
	if data.Blocks != nil {
		for k, v := range data.Blocks {
			decoded, err := base64.StdEncoding.DecodeString(v)
			if err != nil {
				continue
			}
			bs.blocks[k] = decoded
		}
	}
	if data.Index != nil {
		bs.index = data.Index
	}
	return nil
}

// Save writes all blocks to disk
func (bs *Blockstore) Save() error {
	if bs.path == "" {
		return nil
	}
	bs.mu.RLock()
	blocks := make(map[string][]byte, len(bs.blocks))
	for k, v := range bs.blocks {
		blocks[k] = v
	}
	index := make(map[string]plumbing.Hash, len(bs.index))
	for k, v := range bs.index {
		index[k] = v
	}
	bs.mu.RUnlock()

	encoded := make(map[string]string, len(blocks))
	for k, v := range blocks {
		encoded[k] = base64.StdEncoding.EncodeToString(v)
	}
	data := blockstoreData{
		Blocks: encoded,
		Index:  index,
	}
	return persistence.SaveJSON(bs.path, data)
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

	return bs.Save()
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

// SaveBlock persists after a mutation (convenience for handlers)
func (bs *Blockstore) SaveBlock() error {
	return bs.Save()
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
