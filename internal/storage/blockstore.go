package storage

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/go-git/go-git/v6/plumbing"
	"github.com/lakshmanpatel/gitant/internal/persistence"
)

// Blockstore provides content-addressed storage for git objects.
// Blocks are stored as individual files on disk; only the index is kept in memory.
type Blockstore struct {
	mu        sync.RWMutex
	index     map[string]plumbing.Hash
	path      string // index JSON file path
	blocksDir string // directory for block files
}

// NewBlockstore creates a new file-per-block blockstore
func NewBlockstore(path, blocksDir string) *Blockstore {
	return &Blockstore{
		index:     make(map[string]plumbing.Hash),
		path:      path,
		blocksDir: blocksDir,
	}
}

// Load reads persisted index from disk, migrating from old format if needed
func (bs *Blockstore) Load() error {
	if bs.path == "" {
		return nil
	}
	bs.mu.Lock()
	defer bs.mu.Unlock()

	// Ensure blocks directory exists
	if bs.blocksDir != "" {
		if err := os.MkdirAll(bs.blocksDir, 0755); err != nil {
			return fmt.Errorf("creating blocks directory: %w", err)
		}
	}

	// Load raw JSON to detect format
	var raw map[string]json.RawMessage
	if err := persistence.LoadJSON(bs.path, &raw); err != nil {
		return err
	}
	if raw == nil {
		return nil // empty file
	}

	// Check if this is legacy format (has "blocks" key)
	if blocksRaw, ok := raw["blocks"]; ok {
		var blocks map[string]string
		if err := json.Unmarshal(blocksRaw, &blocks); err != nil {
			return fmt.Errorf("parsing legacy blocks: %w", err)
		}

		// Migrate: write each block to its own file and build index from keys
		if bs.blocksDir != "" {
			if err := bs.ensureBlocksDir(); err != nil {
				return err
			}
			for k, v := range blocks {
				decoded, err := base64.StdEncoding.DecodeString(v)
				if err != nil {
					continue
				}
				blockPath := filepath.Join(bs.blocksDir, k)
				if err := os.WriteFile(blockPath, decoded, 0644); err != nil {
					return fmt.Errorf("migrating block %s: %w", k, err)
				}
				// Build index from block keys
				bs.index[k] = plumbing.NewHash(k)
			}
		}

		// Save in new format to complete migration
		return bs.saveIndex()
	}

	// New format: has "index" key
	if indexRaw, ok := raw["index"]; ok {
		var idx map[string]plumbing.Hash
		if err := json.Unmarshal(indexRaw, &idx); err != nil {
			return fmt.Errorf("parsing index: %w", err)
		}
		bs.index = idx
	}

	return nil
}

// saveIndex persists just the index to disk (no block content)
func (bs *Blockstore) saveIndex() error {
	if bs.path == "" {
		return nil
	}
	return persistence.SaveJSON(bs.path, map[string]interface{}{"index": bs.index})
}

// Save writes the index to disk (block files are already on disk)
func (bs *Blockstore) Save() error {
	bs.mu.RLock()
	defer bs.mu.RUnlock()
	return bs.saveIndex()
}

// ensureBlocksDir creates the blocks directory if it doesn't exist
func (bs *Blockstore) ensureBlocksDir() error {
	if bs.blocksDir == "" {
		return nil
	}
	return os.MkdirAll(bs.blocksDir, 0755)
}

// Put stores a block by writing it to disk
func (bs *Blockstore) Put(hash plumbing.Hash, data []byte) error {
	bs.mu.Lock()
	defer bs.mu.Unlock()

	key := hash.String()

	// Write block file to disk
	if bs.blocksDir != "" {
		if err := bs.ensureBlocksDir(); err != nil {
			return err
		}
		blockPath := filepath.Join(bs.blocksDir, key)
		if err := os.WriteFile(blockPath, data, 0644); err != nil {
			return fmt.Errorf("writing block %s: %w", key, err)
		}
	}

	bs.index[key] = hash
	return bs.saveIndex()
}

// Get retrieves a block by reading from disk
func (bs *Blockstore) Get(hash plumbing.Hash) ([]byte, error) {
	bs.mu.RLock()
	key := hash.String()
	_, ok := bs.index[key]
	bs.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("block not found: %s", key)
	}

	if bs.blocksDir == "" {
		return nil, fmt.Errorf("blocks directory not configured")
	}

	blockPath := filepath.Join(bs.blocksDir, key)
	data, err := os.ReadFile(blockPath)
	if err != nil {
		return nil, fmt.Errorf("reading block %s: %w", key, err)
	}

	return data, nil
}

// Has checks if a block exists in the index
func (bs *Blockstore) Has(hash plumbing.Hash) bool {
	bs.mu.RLock()
	defer bs.mu.RUnlock()

	_, ok := bs.index[hash.String()]
	return ok
}

// Delete removes a block file and its index entry
func (bs *Blockstore) Delete(hash plumbing.Hash) error {
	bs.mu.Lock()
	defer bs.mu.Unlock()

	key := hash.String()
	delete(bs.index, key)

	// Remove block file
	if bs.blocksDir != "" {
		blockPath := filepath.Join(bs.blocksDir, key)
		os.Remove(blockPath) // ignore error if file doesn't exist
	}

	return bs.saveIndex()
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

	return len(bs.index)
}

// PutAll stores multiple blocks by writing each to disk
func (bs *Blockstore) PutAll(blocks map[plumbing.Hash][]byte) error {
	bs.mu.Lock()
	defer bs.mu.Unlock()

	if bs.blocksDir != "" {
		if err := bs.ensureBlocksDir(); err != nil {
			return err
		}
	}

	for hash, data := range blocks {
		key := hash.String()

		if bs.blocksDir != "" {
			blockPath := filepath.Join(bs.blocksDir, key)
			if err := os.WriteFile(blockPath, data, 0644); err != nil {
				return fmt.Errorf("writing block %s: %w", key, err)
			}
		}

		bs.index[key] = hash
	}

	return bs.saveIndex()
}

// SaveBlock persists after a mutation (convenience for handlers)
func (bs *Blockstore) SaveBlock() error {
	return bs.Save()
}

// GetAll retrieves multiple blocks by their hashes
func (bs *Blockstore) GetAll(hashes []plumbing.Hash) (map[plumbing.Hash][]byte, error) {
	result := make(map[plumbing.Hash][]byte)

	for _, hash := range hashes {
		data, err := bs.Get(hash)
		if err != nil {
			return nil, err
		}
		result[hash] = data
	}

	return result, nil
}
