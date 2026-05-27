package lfs

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sync"
	"time"
)

var validOID = regexp.MustCompile(`^[a-f0-9]{64}$`)

// Object represents an LFS object
type Object struct {
	OID          string    `json:"oid"`
	Size         int64     `json:"size"`
	RepoID       string    `json:"repo_id"`
	Path         string    `json:"path"`
	CreatedAt    time.Time `json:"created_at"`
}

// Store manages LFS objects
type Store struct {
	mu       sync.RWMutex
	baseDir  string
	objects  map[string]*Object // oid -> object
}

// NewStore creates a new LFS store
func NewStore(baseDir string) *Store {
	return &Store{
		baseDir: baseDir,
		objects: make(map[string]*Object),
	}
}

// Init initializes the LFS store
func (s *Store) Init() error {
	return os.MkdirAll(s.baseDir, 0755)
}

// Upload uploads an LFS object
func (s *Store) Upload(repoID, oid string, reader io.Reader) (*Object, error) {
	if !validOID.MatchString(oid) {
		return nil, fmt.Errorf("invalid OID: must be 64 lowercase hex characters")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if object already exists
	if obj, ok := s.objects[oid]; ok {
		return obj, nil
	}

	// Write to temp file first
	tmpPath := filepath.Join(s.baseDir, oid+".tmp")
	file, err := os.Create(tmpPath)
	if err != nil {
		return nil, fmt.Errorf("creating temp file: %w", err)
	}
	defer file.Close()

	hasher := sha256.New()
	writer := io.MultiWriter(file, hasher)

	size, err := io.Copy(writer, reader)
	if err != nil {
		os.Remove(tmpPath)
		return nil, fmt.Errorf("writing object: %w", err)
	}

	// Verify OID
	computedOID := hex.EncodeToString(hasher.Sum(nil))
	if computedOID != oid {
		os.Remove(tmpPath)
		return nil, fmt.Errorf("OID mismatch: expected %s, got %s", oid, computedOID)
	}

	// Move to final location
	finalPath := filepath.Join(s.baseDir, oid)
	if err := os.Rename(tmpPath, finalPath); err != nil {
		os.Remove(tmpPath)
		return nil, fmt.Errorf("moving object: %w", err)
	}

	obj := &Object{
		OID:       oid,
		Size:      size,
		RepoID:    repoID,
		Path:      finalPath,
		CreatedAt: time.Now(),
	}

	s.objects[oid] = obj
	return obj, nil
}

// Download downloads an LFS object
func (s *Store) Download(oid string) (io.ReadCloser, *Object, error) {
	if !validOID.MatchString(oid) {
		return nil, nil, fmt.Errorf("invalid OID: must be 64 lowercase hex characters")
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	obj, ok := s.objects[oid]
	if !ok {
		return nil, nil, fmt.Errorf("object not found: %s", oid)
	}

	file, err := os.Open(obj.Path)
	if err != nil {
		return nil, nil, fmt.Errorf("opening object: %w", err)
	}

	return file, obj, nil
}

// Get gets LFS object metadata
func (s *Store) Get(oid string) (*Object, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	obj, ok := s.objects[oid]
	if !ok {
		return nil, fmt.Errorf("object not found: %s", oid)
	}

	return obj, nil
}

// Exists checks if an LFS object exists
func (s *Store) Exists(oid string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	_, ok := s.objects[oid]
	return ok
}

// Delete deletes an LFS object
func (s *Store) Delete(oid string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	obj, ok := s.objects[oid]
	if !ok {
		return fmt.Errorf("object not found: %s", oid)
	}

	if err := os.Remove(obj.Path); err != nil && !os.IsNotExist(err) {
		return err
	}

	delete(s.objects, oid)
	return nil
}

// Batch processes a batch request
func (s *Store) Batch(oids []string) map[string]*Object {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[string]*Object)
	for _, oid := range oids {
		if obj, ok := s.objects[oid]; ok {
			result[oid] = obj
		}
	}
	return result
}
