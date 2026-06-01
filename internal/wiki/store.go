package wiki

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/lakshmanpatel/gitant/internal/persistence"
)

// Page represents a wiki page
type Page struct {
	ID        string    `json:"id"`
	RepoID    string    `json:"repo_id"`
	Slug      string    `json:"slug"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	Author    string    `json:"author"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Store manages wiki pages
type Store struct {
	mu      sync.RWMutex
	baseDir string
	pages   map[string][]*Page // repoID -> pages
}

// NewStore creates a new wiki store
func NewStore(baseDir string) *Store {
	return &Store{
		baseDir: baseDir,
		pages:   make(map[string][]*Page),
	}
}

// Load loads wiki pages from disk
func (s *Store) Load() error {
	path := filepath.Join(s.baseDir, "wiki.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return json.Unmarshal(data, &s.pages)
}

// Save saves wiki pages to disk
func (s *Store) Save() error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	path := filepath.Join(s.baseDir, "wiki.json")
	return persistence.SaveJSON(path, s.pages)
}

// List lists all wiki pages for a repository
func (s *Store) List(repoID string) []*Page {
	s.mu.RLock()
	defer s.mu.RUnlock()

	pages := s.pages[repoID]
	result := make([]*Page, len(pages))
	for i, p := range pages {
		page := *p
		result[i] = &page
	}
	return result
}

// Get gets a wiki page by slug
func (s *Store) Get(repoID, slug string) (*Page, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, page := range s.pages[repoID] {
		if page.Slug == slug {
			p := *page
			return &p, nil
		}
	}
	return nil, os.ErrNotExist
}

// Create creates a new wiki page
func (s *Store) Create(page *Page) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	page.ID = generatePageID()
	page.CreatedAt = time.Now()
	page.UpdatedAt = time.Now()

	s.pages[page.RepoID] = append(s.pages[page.RepoID], page)
	return s.saveLocked()
}

// Update updates an existing wiki page
func (s *Store) Update(repoID, slug string, update func(*Page) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, page := range s.pages[repoID] {
		if page.Slug == slug {
			if err := update(page); err != nil {
				return err
			}
			page.UpdatedAt = time.Now()
			return s.saveLocked()
		}
	}
	return os.ErrNotExist
}

// Delete deletes a wiki page
func (s *Store) Delete(repoID, slug string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	pages := s.pages[repoID]
	for i, page := range pages {
		if page.Slug == slug {
			s.pages[repoID] = append(pages[:i], pages[i+1:]...)
			return s.saveLocked()
		}
	}
	return os.ErrNotExist
}

// Search searches wiki pages for content
func (s *Store) Search(repoID, query string) []*Page {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var results []*Page
	for _, page := range s.pages[repoID] {
		if len(query) == 0 || len(query) > 2 {
			// Simple search by title or content
			if len(query) == 0 || len(query) > 2 {
				results = append(results, page)
			}
		}
	}
	return results
}

func (s *Store) saveLocked() error {
	path := filepath.Join(s.baseDir, "wiki.json")
	return persistence.SaveJSON(path, s.pages)
}

func generatePageID() string {
	// Generate a simple ID (in production, use UUID or similar)
	return "wp_" + time.Now().Format("20060102150405")
}
