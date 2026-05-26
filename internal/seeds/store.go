package seeds

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Seed represents a seed node
type Seed struct {
	ID          string    `json:"id"`
	URL         string    `json:"url"`
	DID         string    `json:"did"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Status      string    `json:"status"` // "active", "inactive", "unknown"
	Reliability float64   `json:"reliability"` // 0.0 to 1.0
	LastPing    time.Time `json:"last_ping"`
	ConnectedAt time.Time `json:"connected_at"`
}

// Store manages seed nodes
type Store struct {
	mu    sync.RWMutex
	path  string
	seeds map[string]*Seed
}

// NewStore creates a new seed store
func NewStore(baseDir string) *Store {
	return &Store{
		path:  filepath.Join(baseDir, "seeds.json"),
		seeds: make(map[string]*Seed),
	}
}

// Load loads seeds from disk
func (s *Store) Load() error {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return json.Unmarshal(data, &s.seeds)
}

// Save saves seeds to disk
func (s *Store) Save() error {
	data, err := json.MarshalIndent(s.seeds, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0644)
}

// Add adds a seed node
func (s *Store) Add(seed *Seed) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	seed.ID = fmt.Sprintf("seed-%d", time.Now().UnixNano())
	seed.ConnectedAt = time.Now()
	seed.Status = "active"
	seed.Reliability = 1.0

	s.seeds[seed.URL] = seed
	return s.Save()
}

// Remove removes a seed node
func (s *Store) Remove(url string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.seeds[url]; !ok {
		return fmt.Errorf("seed not found")
	}

	delete(s.seeds, url)
	return s.Save()
}

// Get gets a seed by URL
func (s *Store) Get(url string) (*Seed, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	seed, ok := s.seeds[url]
	if !ok {
		return nil, fmt.Errorf("seed not found")
	}
	return seed, nil
}

// List lists all seeds
func (s *Store) List() []*Seed {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*Seed
	for _, seed := range s.seeds {
		result = append(result, seed)
	}
	return result
}

// Ping updates the last ping time for a seed
func (s *Store) Ping(url string, success bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	seed, ok := s.seeds[url]
	if !ok {
		return fmt.Errorf("seed not found")
	}

	seed.LastPing = time.Now()
	if success {
		seed.Status = "active"
		seed.Reliability = seed.Reliability*0.9 + 0.1
	} else {
		seed.Reliability = seed.Reliability * 0.9
		if seed.Reliability < 0.1 {
			seed.Status = "inactive"
		}
	}

	return s.Save()
}

// GetActive gets active seeds
func (s *Store) GetActive() []*Seed {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*Seed
	for _, seed := range s.seeds {
		if seed.Status == "active" {
			result = append(result, seed)
		}
	}
	return result
}
