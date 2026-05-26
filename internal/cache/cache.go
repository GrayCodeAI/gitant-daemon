package cache

import (
	"sync"
	"time"
)

// CacheItem represents a cached item
type CacheItem struct {
	Value     interface{}
	ExpiresAt time.Time
}

// Cache is an in-memory cache with TTL support
type Cache struct {
	mu       sync.RWMutex
	items    map[string]*CacheItem
	ttl      time.Duration
	stopChan chan struct{}
}

// New creates a new cache with the given TTL
func New(ttl time.Duration) *Cache {
	c := &Cache{
		items:    make(map[string]*CacheItem),
		ttl:      ttl,
		stopChan: make(chan struct{}),
	}

	// Start cleanup goroutine
	go c.cleanup()

	return c
}

// Get gets an item from the cache
func (c *Cache) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	item, ok := c.items[key]
	if !ok {
		return nil, false
	}

	if time.Now().After(item.ExpiresAt) {
		return nil, false
	}

	return item.Value, true
}

// Set sets an item in the cache
func (c *Cache) Set(key string, value interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items[key] = &CacheItem{
		Value:     value,
		ExpiresAt: time.Now().Add(c.ttl),
	}
}

// SetWithTTL sets an item in the cache with a custom TTL
func (c *Cache) SetWithTTL(key string, value interface{}, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items[key] = &CacheItem{
		Value:     value,
		ExpiresAt: time.Now().Add(ttl),
	}
}

// Delete deletes an item from the cache
func (c *Cache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.items, key)
}

// Clear clears all items from the cache
func (c *Cache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items = make(map[string]*CacheItem)
}

// Size returns the number of items in the cache
func (c *Cache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return len(c.items)
}

// Close stops the cleanup goroutine
func (c *Cache) Close() {
	close(c.stopChan)
}

func (c *Cache) cleanup() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.removeExpired()
		case <-c.stopChan:
			return
		}
	}
}

func (c *Cache) removeExpired() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	for key, item := range c.items {
		if now.After(item.ExpiresAt) {
			delete(c.items, key)
		}
	}
}

// CacheMiddleware provides HTTP response caching
type CacheMiddleware struct {
	cache *Cache
}

// NewCacheMiddleware creates a new cache middleware
func NewCacheMiddleware(ttl time.Duration) *CacheMiddleware {
	return &CacheMiddleware{
		cache: New(ttl),
	}
}

// Get gets a cached response
func (m *CacheMiddleware) Get(key string) (interface{}, bool) {
	return m.cache.Get(key)
}

// Set caches a response
func (m *CacheMiddleware) Set(key string, value interface{}) {
	m.cache.Set(key, value)
}

// Delete deletes a cached response
func (m *CacheMiddleware) Delete(key string) {
	m.cache.Delete(key)
}
