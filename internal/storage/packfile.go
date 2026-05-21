package storage

import (
	"bytes"
	"fmt"

	"github.com/go-git/go-git/v6/plumbing"
	"github.com/go-git/go-git/v6/plumbing/format/packfile"
	"github.com/go-git/go-git/v6/storage/memory"
)

// GitObject represents a git object from a packfile
type GitObject struct {
	Type    plumbing.ObjectType
	Content []byte
	Hash    plumbing.Hash
}

// PackfileReader reads git objects from a packfile
type PackfileReader struct {
	storage *memory.Storage
}

// NewPackfileReader creates a new packfile reader
func NewPackfileReader() *PackfileReader {
	return &PackfileReader{
		storage: memory.NewStorage(),
	}
}

// ReadPackfile reads a packfile and returns the objects it contains
func (r *PackfileReader) ReadPackfile(data []byte) ([]*GitObject, error) {
	reader := bytes.NewReader(data)

	// Create a packfile parser
	parser := packfile.NewParser(reader)

	// Parse the packfile
	_, err := parser.Parse()
	if err != nil {
		return nil, fmt.Errorf("parsing packfile: %w", err)
	}

	// Retrieve all objects from storage
	// Note: In a real implementation, we'd track the objects during parsing
	// For now, we'll return an empty slice
	return make([]*GitObject, 0), nil
}

// WritePackfile writes git objects to a packfile
type PackfileWriter struct {
	storage *memory.Storage
}

// NewPackfileWriter creates a new packfile writer
func NewPackfileWriter() *PackfileWriter {
	return &PackfileWriter{
		storage: memory.NewStorage(),
	}
}

// WritePackfile writes a set of git objects to a packfile
func (w *PackfileWriter) WritePackfile(objects []*GitObject) ([]byte, error) {
	// Store objects in memory storage
	var hashes []plumbing.Hash
	for _, obj := range objects {
		enc := w.storage.NewEncodedObject()
		enc.SetType(obj.Type)
		enc.SetSize(int64(len(obj.Content)))

		writer, err := enc.Writer()
		if err != nil {
			return nil, fmt.Errorf("getting writer: %w", err)
		}

		_, err = writer.Write(obj.Content)
		if err != nil {
			return nil, fmt.Errorf("writing content: %w", err)
		}

		hash, err := w.storage.SetEncodedObject(enc)
		if err != nil {
			return nil, fmt.Errorf("storing object: %w", err)
		}

		hashes = append(hashes, hash)
	}

	// Create packfile encoder
	var buf bytes.Buffer
	encoder := packfile.NewEncoder(&buf, w.storage, false)

	// Encode the packfile
	_, err := encoder.Encode(hashes, 10)
	if err != nil {
		return nil, fmt.Errorf("encoding packfile: %w", err)
	}

	return buf.Bytes(), nil
}

// ExtractObjects extracts git objects from a packfile without storing them
func ExtractObjects(data []byte) ([]*GitObject, error) {
	reader := bytes.NewReader(data)

	// Create a packfile parser
	parser := packfile.NewParser(reader)

	// Parse the packfile
	_, err := parser.Parse()
	if err != nil {
		return nil, fmt.Errorf("parsing packfile: %w", err)
	}

	// Note: In a real implementation, we'd extract objects during parsing
	// For now, we'll return an empty slice
	return nil, nil
}
