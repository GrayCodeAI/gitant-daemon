package core

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/go-git/go-git/v6/plumbing"
)

// CIDStore manages the mapping between git hashes and CIDs
type CIDStore struct {
	mu       sync.RWMutex
	gitToCID map[string]*CID
	cidToGit map[string]plumbing.Hash
	path     string
}

// NewCIDStore creates a new CID store
func NewCIDStore(path string) *CIDStore {
	return &CIDStore{
		gitToCID: make(map[string]*CID),
		cidToGit: make(map[string]plumbing.Hash),
		path:     path,
	}
}

// AddMapping adds a mapping between a git hash and a CID
func (s *CIDStore) AddMapping(gitHash plumbing.Hash, cid *CID) {
	s.mu.Lock()
	defer s.mu.Unlock()

	gitStr := gitHash.String()
	s.gitToCID[gitStr] = cid
	s.cidToGit[cid.Hash] = gitHash
}

// GetCID returns the CID for a git hash
func (s *CIDStore) GetCID(gitHash plumbing.Hash) (*CID, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	cid, ok := s.gitToCID[gitHash.String()]
	return cid, ok
}

// GetGitHash returns the git hash for a CID
func (s *CIDStore) GetGitHash(cid *CID) (plumbing.Hash, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	hash, ok := s.cidToGit[cid.Hash]
	return hash, ok
}

// Save persists the CID store to disk
func (s *CIDStore) Save() error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data := map[string]string{
		"gitToCID": marshalMap(s.gitToCID),
		"cidToGit": marshalHashMap(s.cidToGit),
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshaling CID store: %w", err)
	}

	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating directory: %w", err)
	}

	return os.WriteFile(s.path, jsonData, 0644)
}

// Load reads the CID store from disk
func (s *CIDStore) Load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("reading CID store: %w", err)
	}

	var stored map[string]string
	if err := json.Unmarshal(data, &stored); err != nil {
		return fmt.Errorf("unmarshaling CID store: %w", err)
	}

	s.gitToCID = unmarshalMap(stored["gitToCID"])
	s.cidToGit = unmarshalHashMap(stored["cidToGit"])

	return nil
}

func marshalMap(m map[string]*CID) string {
	data := make(map[string]string)
	for k, v := range m {
		data[k] = v.Hash
	}
	jsonData, _ := json.Marshal(data)
	return string(jsonData)
}

func marshalHashMap(m map[string]plumbing.Hash) string {
	data := make(map[string]string)
	for k, v := range m {
		data[k] = v.String()
	}
	jsonData, _ := json.Marshal(data)
	return string(jsonData)
}

func unmarshalMap(s string) map[string]*CID {
	var data map[string]string
	json.Unmarshal([]byte(s), &data)

	result := make(map[string]*CID)
	for k, v := range data {
		result[k] = &CID{Hash: v}
	}
	return result
}

func unmarshalHashMap(s string) map[string]plumbing.Hash {
	var data map[string]string
	json.Unmarshal([]byte(s), &data)

	result := make(map[string]plumbing.Hash)
	for k, v := range data {
		result[k] = plumbing.NewHash(v)
	}
	return result
}
