package packages

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Package represents a package
type Package struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	RepoID      string          `json:"repo_id"`
	Versions    map[string]*Version `json:"versions"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

// Version represents a package version
type Version struct {
	Version     string            `json:"version"`
	Description string            `json:"description"`
	Author      string            `json:"author"`
	License     string            `json:"license"`
	Dependencies map[string]string `json:"dependencies"`
	Dist        Dist              `json:"dist"`
	CreatedAt   time.Time         `json:"created_at"`
}

// Dist contains distribution info
type Dist struct {
	Tarball string `json:"tarball"`
	Shasum  string `json:"shasum"`
}

// Registry manages packages
type Registry struct {
	mu       sync.RWMutex
	baseDir  string
	packages map[string]*Package
}

// NewRegistry creates a new package registry
func NewRegistry(baseDir string) *Registry {
	return &Registry{
		baseDir:  baseDir,
		packages: make(map[string]*Package),
	}
}

// Load loads packages from disk
func (r *Registry) Load() error {
	path := filepath.Join(r.baseDir, "packages.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("reading packages: %w", err)
	}
	return json.Unmarshal(data, &r.packages)
}

// Save saves packages to disk
func (r *Registry) Save() error {
	path := filepath.Join(r.baseDir, "packages.json")
	data, err := json.MarshalIndent(r.packages, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling packages: %w", err)
	}
	return os.WriteFile(path, data, 0644)
}

// Publish publishes a new package version
func (r *Registry) Publish(pkg *Package) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if existing, ok := r.packages[pkg.Name]; ok {
		for version, v := range pkg.Versions {
			existing.Versions[version] = v
		}
		existing.UpdatedAt = time.Now()
	} else {
		r.packages[pkg.Name] = pkg
	}

	return r.Save()
}

// Get gets a package by name
func (r *Registry) Get(name string) (*Package, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	pkg, ok := r.packages[name]
	if !ok {
		return nil, fmt.Errorf("package not found: %s", name)
	}
	return pkg, nil
}

// GetVersion gets a specific version of a package
func (r *Registry) GetVersion(name, version string) (*Version, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	pkg, ok := r.packages[name]
	if !ok {
		return nil, fmt.Errorf("package not found: %s", name)
	}

	v, ok := pkg.Versions[version]
	if !ok {
		return nil, fmt.Errorf("version not found: %s@%s", name, version)
	}
	return v, nil
}

// List lists all packages
func (r *Registry) List() []*Package {
	r.mu.RLock()
	defer r.mu.RUnlock()

	packages := make([]*Package, 0, len(r.packages))
	for _, pkg := range r.packages {
		packages = append(packages, pkg)
	}
	return packages
}

// Search searches for packages
func (r *Registry) Search(query string) []*Package {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var results []*Package
	for _, pkg := range r.packages {
		if contains(pkg.Name, query) || contains(pkg.Description, query) {
			results = append(results, pkg)
		}
	}
	return results
}

// Delete deletes a package
func (r *Registry) Delete(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.packages[name]; !ok {
		return fmt.Errorf("package not found: %s", name)
	}

	delete(r.packages, name)
	return r.Save()
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && len(substr) > 0 && containsIgnoreCase(s, substr))
}

func containsIgnoreCase(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		match := true
		for j := 0; j < len(substr); j++ {
			if toLower(s[i+j]) != toLower(substr[j]) {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

func toLower(b byte) byte {
	if b >= 'A' && b <= 'Z' {
		return b + 32
	}
	return b
}
