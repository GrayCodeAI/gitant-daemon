package identity

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// RevocationStore manages revoked UCAN nonces with persistence.
type RevocationStore struct {
	mu          sync.RWMutex
	revocations map[string]time.Time // nonce -> revocation time
	path        string
}

// NewRevocationStore creates a new empty revocation store.
func NewRevocationStore(dataDir string) *RevocationStore {
	return &RevocationStore{
		revocations: make(map[string]time.Time),
		path:        filepath.Join(dataDir, "revocations.json"),
	}
}

// Revoke marks a UCAN nonce as revoked.
func (rs *RevocationStore) Revoke(nonce string) {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	rs.revocations[nonce] = time.Now()
}

// IsRevoked checks if a UCAN nonce has been revoked.
func (rs *RevocationStore) IsRevoked(nonce string) bool {
	rs.mu.RLock()
	defer rs.mu.RUnlock()
	_, revoked := rs.revocations[nonce]
	return revoked
}

// List returns all revoked nonces and their revocation times.
func (rs *RevocationStore) List() map[string]time.Time {
	rs.mu.RLock()
	defer rs.mu.RUnlock()
	result := make(map[string]time.Time, len(rs.revocations))
	for k, v := range rs.revocations {
		result[k] = v
	}
	return result
}

// Load reads the revocation store from disk.
func (rs *RevocationStore) Load() error {
	data, err := os.ReadFile(rs.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // no file yet, not an error
		}
		return fmt.Errorf("reading revocations file: %w", err)
	}

	var entries map[string]time.Time
	if err := json.Unmarshal(data, &entries); err != nil {
		return fmt.Errorf("unmarshaling revocations: %w", err)
	}

	rs.mu.Lock()
	rs.revocations = entries
	rs.mu.Unlock()

	return nil
}

// Save persists the revocation store to disk.
func (rs *RevocationStore) Save() error {
	rs.mu.RLock()
	data, err := json.MarshalIndent(rs.revocations, "", "  ")
	rs.mu.RUnlock()
	if err != nil {
		return fmt.Errorf("marshaling revocations: %w", err)
	}

	dir := filepath.Dir(rs.path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating directory: %w", err)
	}

	if err := os.WriteFile(rs.path, data, 0644); err != nil {
		return fmt.Errorf("writing revocations file: %w", err)
	}

	return nil
}
