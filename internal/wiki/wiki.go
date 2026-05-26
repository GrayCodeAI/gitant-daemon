package wiki

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// Page represents a wiki page
type Page struct {
	Title     string    `json:"title"`
	Slug      string    `json:"slug"`
	Content   string    `json:"content"`
	Author    string    `json:"author"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Wiki manages wiki pages for a repository
type Wiki struct {
	mu      sync.RWMutex
	repoID  string
	baseDir string
}

// NewWiki creates a new wiki for a repository
func NewWiki(baseDir, repoID string) *Wiki {
	return &Wiki{
		repoID:  repoID,
		baseDir: filepath.Join(baseDir, repoID, "wiki"),
	}
}

// GetPage gets a wiki page by slug
func (w *Wiki) GetPage(slug string) (*Page, error) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	path := w.pagePath(slug)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("page not found: %s", slug)
		}
		return nil, fmt.Errorf("reading page: %w", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	return &Page{
		Title:     extractTitle(string(data), slug),
		Slug:      slug,
		Content:   string(data),
		UpdatedAt: info.ModTime(),
	}, nil
}

// ListPages lists all wiki pages
func (w *Wiki) ListPages() ([]*Page, error) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	if err := os.MkdirAll(w.baseDir, 0755); err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(w.baseDir)
	if err != nil {
		return nil, err
	}

	var pages []*Page
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}

		slug := strings.TrimSuffix(entry.Name(), ".md")
		path := filepath.Join(w.baseDir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		pages = append(pages, &Page{
			Title:     extractTitle(string(data), slug),
			Slug:      slug,
			Content:   string(data),
			UpdatedAt: info.ModTime(),
		})
	}

	sort.Slice(pages, func(i, j int) bool {
		return pages[i].Title < pages[j].Title
	})

	return pages, nil
}

// CreatePage creates a new wiki page
func (w *Wiki) CreatePage(slug, content, author string) (*Page, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if err := os.MkdirAll(w.baseDir, 0755); err != nil {
		return nil, err
	}

	path := w.pagePath(slug)
	if _, err := os.Stat(path); err == nil {
		return nil, fmt.Errorf("page already exists: %s", slug)
	}

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return nil, err
	}

	now := time.Now()
	return &Page{
		Title:     extractTitle(content, slug),
		Slug:      slug,
		Content:   content,
		Author:    author,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

// UpdatePage updates an existing wiki page
func (w *Wiki) UpdatePage(slug, content, author string) (*Page, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	path := w.pagePath(slug)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, fmt.Errorf("page not found: %s", slug)
	}

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return nil, err
	}

	return &Page{
		Title:     extractTitle(content, slug),
		Slug:      slug,
		Content:   content,
		Author:    author,
		UpdatedAt: time.Now(),
	}, nil
}

// DeletePage deletes a wiki page
func (w *Wiki) DeletePage(slug string) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	path := w.pagePath(slug)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("page not found: %s", slug)
	}

	return os.Remove(path)
}

// Search searches wiki pages
func (w *Wiki) Search(query string) ([]*Page, error) {
	pages, err := w.ListPages()
	if err != nil {
		return nil, err
	}

	var results []*Page
	query = strings.ToLower(query)
	for _, page := range pages {
		if strings.Contains(strings.ToLower(page.Title), query) ||
			strings.Contains(strings.ToLower(page.Content), query) {
			results = append(results, page)
		}
	}

	return results, nil
}

func (w *Wiki) pagePath(slug string) string {
	return filepath.Join(w.baseDir, slug+".md")
}

func extractTitle(content, fallback string) string {
	lines := strings.SplitN(content, "\n", 5)
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "# ") {
			return strings.TrimPrefix(line, "# ")
		}
	}
	return fallback
}
