package identity

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (
	defaultNonceTTL      = 10 * time.Minute
	defaultEvictInterval = 1 * time.Minute
)

// NonceCache tracks seen UCAN nonces to prevent replay attacks.
// Entries are persisted to disk so nonces survive server restarts.
type NonceCache struct {
	mu      sync.RWMutex
	entries map[string]time.Time // nonce -> expiry
	ttl     time.Duration
	path    string
	stop    chan struct{}
}

// NewNonceCache creates a started NonceCache with the given TTL.
// If ttl <= 0, defaultNonceTTL is used. If dataDir is non-empty, entries
// are persisted to dataDir/nonce_cache.json.
func NewNonceCache(ttl time.Duration, dataDir string) *NonceCache {
	if ttl <= 0 {
		ttl = defaultNonceTTL
	}
	nc := &NonceCache{
		entries: make(map[string]time.Time),
		ttl:     ttl,
		stop:    make(chan struct{}),
	}
	if dataDir != "" {
		nc.path = filepath.Join(dataDir, "nonce_cache.json")
	}
	go nc.evictLoop()
	return nc
}

// Check returns true if the nonce has NOT been seen before, and records it.
// If the nonce was already seen, it returns false (replay detected).
func (nc *NonceCache) Check(nonce string) bool {
	nc.mu.Lock()
	defer nc.mu.Unlock()

	if _, exists := nc.entries[nonce]; exists {
		return false
	}
	nc.entries[nonce] = time.Now().Add(nc.ttl)
	return true
}

// Stop halts the background eviction goroutine.
func (nc *NonceCache) Stop() {
	close(nc.stop)
}

// Load reads the nonce cache from disk. Expired entries are discarded.
func (nc *NonceCache) Load() error {
	if nc.path == "" {
		return nil
	}
	data, err := os.ReadFile(nc.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("reading nonce cache: %w", err)
	}

	var entries map[string]time.Time
	if err := json.Unmarshal(data, &entries); err != nil {
		return fmt.Errorf("unmarshaling nonce cache: %w", err)
	}

	now := time.Now()
	nc.mu.Lock()
	for nonce, expiry := range entries {
		if now.Before(expiry) {
			nc.entries[nonce] = expiry
		}
	}
	nc.mu.Unlock()
	return nil
}

// Save persists the nonce cache to disk.
func (nc *NonceCache) Save() error {
	if nc.path == "" {
		return nil
	}
	nc.mu.RLock()
	data, err := json.MarshalIndent(nc.entries, "", "  ")
	nc.mu.RUnlock()
	if err != nil {
		return fmt.Errorf("marshaling nonce cache: %w", err)
	}

	dir := filepath.Dir(nc.path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating directory: %w", err)
	}

	tmpPath := nc.path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0600); err != nil {
		return fmt.Errorf("writing nonce cache: %w", err)
	}
	if err := os.Rename(tmpPath, nc.path); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("renaming nonce cache: %w", err)
	}
	return nil
}

func (nc *NonceCache) evictLoop() {
	ticker := time.NewTicker(defaultEvictInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			nc.evict()
		case <-nc.stop:
			return
		}
	}
}

func (nc *NonceCache) evict() {
	now := time.Now()
	nc.mu.Lock()
	defer nc.mu.Unlock()
	for nonce, expiry := range nc.entries {
		if now.After(expiry) {
			delete(nc.entries, nonce)
		}
	}
}
