package extensions

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Extension represents a CLI extension
type Extension struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Version     string            `json:"version"`
	Author      string            `json:"author"`
	Repository  string            `json:"repository"`
	Commands    []Command         `json:"commands"`
	Config      map[string]string `json:"config"`
	InstalledAt time.Time         `json:"installed_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

// Command represents an extension command
type Command struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Usage       string `json:"usage"`
	Handler     string `json:"handler"` // Script or binary path
}

// Store manages extensions
type Store struct {
	mu         sync.RWMutex
	baseDir    string
	extensions map[string]*Extension
}

// NewStore creates a new extension store
func NewStore(baseDir string) *Store {
	return &Store{
		baseDir:    baseDir,
		extensions: make(map[string]*Extension),
	}
}

// Load loads extensions from disk
func (s *Store) Load() error {
	path := filepath.Join(s.baseDir, "extensions.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return json.Unmarshal(data, &s.extensions)
}

// Save saves extensions to disk
func (s *Store) Save() error {
	path := filepath.Join(s.baseDir, "extensions.json")
	data, err := json.MarshalIndent(s.extensions, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// Install installs an extension
func (s *Store) Install(ext *Extension) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	ext.InstalledAt = time.Now()
	ext.UpdatedAt = time.Now()
	s.extensions[ext.Name] = ext
	return s.Save()
}

// Uninstall uninstalls an extension
func (s *Store) Uninstall(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.extensions[name]; !ok {
		return fmt.Errorf("extension not found: %s", name)
	}

	delete(s.extensions, name)
	return s.Save()
}

// Get gets an extension by name
func (s *Store) Get(name string) (*Extension, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ext, ok := s.extensions[name]
	if !ok {
		return nil, fmt.Errorf("extension not found: %s", name)
	}
	return ext, nil
}

// List lists all extensions
func (s *Store) List() []*Extension {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*Extension
	for _, ext := range s.extensions {
		result = append(result, ext)
	}
	return result
}

// Update updates an extension
func (s *Store) Update(name string, ext *Extension) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.extensions[name]; !ok {
		return fmt.Errorf("extension not found: %s", name)
	}

	ext.UpdatedAt = time.Now()
	s.extensions[name] = ext
	return s.Save()
}
