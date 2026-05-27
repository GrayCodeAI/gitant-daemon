package identity

import (
	"sync"
	"time"
)

const (
	defaultNonceTTL      = 10 * time.Minute
	defaultEvictInterval = 1 * time.Minute
)

// NonceCache tracks seen UCAN nonces to prevent replay attacks.
type NonceCache struct {
	mu      sync.RWMutex
	entries map[string]time.Time // nonce -> expiry
	ttl     time.Duration
	stop    chan struct{}
}

// NewNonceCache creates a started NonceCache with the given TTL.
// If ttl <= 0, defaultNonceTTL is used.
func NewNonceCache(ttl time.Duration) *NonceCache {
	if ttl <= 0 {
		ttl = defaultNonceTTL
	}
	nc := &NonceCache{
		entries: make(map[string]time.Time),
		ttl:     ttl,
		stop:    make(chan struct{}),
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
