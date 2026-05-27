package packages

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// PackageFormat represents the package format
type PackageFormat string

const (
	FormatNPM    PackageFormat = "npm"
	FormatDocker PackageFormat = "docker"
	FormatPyPI   PackageFormat = "pypi"
	FormatMaven  PackageFormat = "maven"
	FormatGo     PackageFormat = "go"
	FormatCargo  PackageFormat = "cargo"
	FormatNuGet  PackageFormat = "nuget"
	FormatGeneric PackageFormat = "generic"
)

// EnhancedPackage represents a multi-format package
type EnhancedPackage struct {
	Name        string                 `json:"name"`
	Format      PackageFormat          `json:"format"`
	Description string                 `json:"description"`
	RepoID      string                 `json:"repo_id"`
	Author      string                 `json:"author"`
	License     string                 `json:"license"`
	Homepage    string                 `json:"homepage"`
	Keywords    []string               `json:"keywords"`
	Versions    map[string]*EnhancedVersion `json:"versions"`
	Metadata    map[string]interface{} `json:"metadata"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// EnhancedVersion represents a version with format-specific metadata
type EnhancedVersion struct {
	Version      string            `json:"version"`
	Description  string            `json:"description"`
	Author       string            `json:"author"`
	License      string            `json:"license"`
	Dependencies map[string]string `json:"dependencies"`
	Dist         DistInfo          `json:"dist"`
	FormatMeta   FormatMetadata    `json:"format_meta"`
	CreatedAt    time.Time         `json:"created_at"`
	PublishedBy  string            `json:"published_by"`
}

// DistInfo contains distribution info
type DistInfo struct {
	Tarball  string `json:"tarball"`
	Shasum   string `json:"shasum"`
	Size     int64  `json:"size"`
	Uploaded bool   `json:"uploaded"`
}

// FormatMetadata contains format-specific metadata
type FormatMetadata struct {
	// NPM specific
	Main       string            `json:"main,omitempty"`
	Scripts    map[string]string `json:"scripts,omitempty"`
	Bin        map[string]string `json:"bin,omitempty"`
	Engines    map[string]string `json:"engines,omitempty"`

	// Docker specific
	Registry   string `json:"registry,omitempty"`
	Repository string `json:"repository,omitempty"`
	Tag        string `json:"tag,omitempty"`
	Digest     string `json:"digest,omitempty"`
	Layers     []string `json:"layers,omitempty"`
	Arch       []string `json:"arch,omitempty"`

	// PyPI specific
	RequiresPython string   `json:"requires_python,omitempty"`
	Classifiers    []string `json:"classifiers,omitempty"`
	ProjectURLs    map[string]string `json:"project_urls,omitempty"`
	EntryPoint     string   `json:"entry_point,omitempty"`

	// Maven specific
	GroupID    string `json:"group_id,omitempty"`
	ArtifactID string `json:"artifact_id,omitempty"`
	Packaging  string `json:"packaging,omitempty"`
	JavaVersion string `json:"java_version,omitempty"`
}

// EnhancedRegistry manages multi-format packages
type EnhancedRegistry struct {
	mu       sync.RWMutex
	baseDir  string
	packages map[string]*EnhancedPackage
}

// NewEnhancedRegistry creates a new enhanced registry
func NewEnhancedRegistry(baseDir string) *EnhancedRegistry {
	return &EnhancedRegistry{
		baseDir:  baseDir,
		packages: make(map[string]*EnhancedPackage),
	}
}

// Publish publishes a package version
func (r *EnhancedRegistry) Publish(pkg *EnhancedPackage) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := fmt.Sprintf("%s/%s", pkg.Format, pkg.Name)

	if existing, ok := r.packages[key]; ok {
		for version, v := range pkg.Versions {
			existing.Versions[version] = v
		}
		existing.UpdatedAt = time.Now()
	} else {
		pkg.CreatedAt = time.Now()
		pkg.UpdatedAt = time.Now()
		r.packages[key] = pkg
	}

	return r.save()
}

// Get gets a package by format and name
func (r *EnhancedRegistry) Get(format PackageFormat, name string) (*EnhancedPackage, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	key := fmt.Sprintf("%s/%s", format, name)
	pkg, ok := r.packages[key]
	if !ok {
		return nil, fmt.Errorf("package not found: %s/%s", format, name)
	}
	return pkg, nil
}

// GetVersion gets a specific version
func (r *EnhancedRegistry) GetVersion(format PackageFormat, name, version string) (*EnhancedVersion, error) {
	pkg, err := r.Get(format, name)
	if err != nil {
		return nil, err
	}

	v, ok := pkg.Versions[version]
	if !ok {
		return nil, fmt.Errorf("version not found: %s@%s", name, version)
	}
	return v, nil
}

// List lists all packages of a format
func (r *EnhancedRegistry) List(format PackageFormat) []*EnhancedPackage {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var packages []*EnhancedPackage
	for _, pkg := range r.packages {
		if pkg.Format == format {
			packages = append(packages, pkg)
		}
	}
	return packages
}

// Search searches packages across all formats
func (r *EnhancedRegistry) Search(query string) []*EnhancedPackage {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var results []*EnhancedPackage
	q := strings.ToLower(query)
	for _, pkg := range r.packages {
		if strings.Contains(strings.ToLower(pkg.Name), q) ||
			strings.Contains(strings.ToLower(pkg.Description), q) {
			results = append(results, pkg)
		}
	}
	return results
}

// UploadAsset uploads a package asset (tarball, wheel, jar, etc.)
func (r *EnhancedRegistry) UploadAsset(format PackageFormat, name, version, filename string, data []byte) (string, error) {
	dir := filepath.Join(r.baseDir, string(format), name, version)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("creating directory: %w", err)
	}

	path := filepath.Join(dir, filename)
	if err := os.WriteFile(path, data, 0644); err != nil {
		return "", fmt.Errorf("writing file: %w", err)
	}

	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:]), nil
}

// GetAssetPath returns the path to a package asset
func (r *EnhancedRegistry) GetAssetPath(format PackageFormat, name, version, filename string) string {
	return filepath.Join(r.baseDir, string(format), name, version, filename)
}

// GenerateNPMRegistry generates npm-compatible registry response
func (r *EnhancedRegistry) GenerateNPMRegistry(name string) (map[string]interface{}, error) {
	pkg, err := r.Get(FormatNPM, name)
	if err != nil {
		return nil, err
	}

	versions := make(map[string]interface{})
	for ver, v := range pkg.Versions {
		versions[ver] = map[string]interface{}{
			"name":            pkg.Name,
			"version":         ver,
			"description":     v.Description,
			"main":            v.FormatMeta.Main,
			"scripts":         v.FormatMeta.Scripts,
			"bin":             v.FormatMeta.Bin,
			"engines":         v.FormatMeta.Engines,
			"dependencies":    v.Dependencies,
			"dist": map[string]interface{}{
				"tarball": v.Dist.Tarball,
				"shasum":  v.Dist.Shasum,
			},
		}
	}

	return map[string]interface{}{
		"name":        pkg.Name,
		"description": pkg.Description,
		"versions":    versions,
		"dist-tags":   map[string]string{"latest": latestVersion(pkg.Versions)},
	}, nil
}

// GeneratePyPIIndex generates PyPI-compatible Simple Repository API response
func (r *EnhancedRegistry) GeneratePyPIIndex(name string) (string, error) {
	pkg, err := r.Get(FormatPyPI, name)
	if err != nil {
		return "", err
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("<html>\n<head><title>Links for %s</title></head>\n<body>\n<h1>Links for %s</h1>\n", pkg.Name, pkg.Name))

	for ver, v := range pkg.Versions {
		filename := fmt.Sprintf("%s-%s.tar.gz", pkg.Name, ver)
		sb.WriteString(fmt.Sprintf("<a href=\"../../packages/%s/%s/%s#sha256=%s\">%s</a><br/>\n",
			pkg.Name, ver, filename, v.Dist.Shasum, filename))
	}

	sb.WriteString("</body>\n</html>")
	return sb.String(), nil
}

// GenerateMavenMetadata generates Maven-compatible metadata
func (r *EnhancedRegistry) GenerateMavenMetadata(groupID, artifactID string) (string, error) {
	pkg, err := r.Get(FormatMaven, fmt.Sprintf("%s:%s", groupID, artifactID))
	if err != nil {
		return "", err
	}

	var versions []string
	for ver := range pkg.Versions {
		versions = append(versions, ver)
	}

	var sb strings.Builder
	sb.WriteString("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n")
	sb.WriteString("<metadata>\n")
	sb.WriteString(fmt.Sprintf("  <groupId>%s</groupId>\n", groupID))
	sb.WriteString(fmt.Sprintf("  <artifactId>%s</artifactId>\n", artifactID))
	sb.WriteString(fmt.Sprintf("  <versioning>\n    <latest>%s</latest>\n", latestVersion(pkg.Versions)))
	sb.WriteString("    <versions>\n")
	for _, v := range versions {
		sb.WriteString(fmt.Sprintf("      <version>%s</version>\n", v))
	}
	sb.WriteString("    </versions>\n  </versioning>\n</metadata>")

	return sb.String(), nil
}

// Delete deletes a package
func (r *EnhancedRegistry) Delete(format PackageFormat, name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := fmt.Sprintf("%s/%s", format, name)
	if _, ok := r.packages[key]; !ok {
		return fmt.Errorf("package not found")
	}

	delete(r.packages, key)
	return r.save()
}

func (r *EnhancedRegistry) save() error {
	path := filepath.Join(r.baseDir, "registry.json")
	data, err := json.MarshalIndent(r.packages, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func latestVersion(versions map[string]*EnhancedVersion) string {
	var latest string
	for ver := range versions {
		if latest == "" || ver > latest {
			latest = ver
		}
	}
	return latest
}
