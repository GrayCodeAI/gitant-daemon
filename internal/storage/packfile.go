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

// ReadPackfile reads a packfile and returns the objects it contains.
// Uses go-git's Parser with memory storage to handle zlib decompression
// and delta resolution automatically.
func (r *PackfileReader) ReadPackfile(data []byte) ([]*GitObject, error) {
	// Each call uses a fresh storage to avoid cross-contamination
	ms := memory.NewStorage()

	reader := bytes.NewReader(data)
	parser := packfile.NewParser(reader, packfile.WithStorage(ms))

	_, err := parser.Parse()
	if err != nil {
		return nil, fmt.Errorf("parsing packfile: %w", err)
	}

	// Extract all objects stored by the parser
	var objects []*GitObject
	err = ms.ForEachObjectHash(func(hash plumbing.Hash) error {
		obj, err := ms.EncodedObject(plumbing.AnyObject, hash)
		if err != nil {
			return nil // skip objects we can't retrieve
		}

		rdr, err := obj.Reader()
		if err != nil {
			return nil
		}
		defer rdr.Close()

		content := make([]byte, obj.Size())
		n, err := rdr.Read(content)
		if err != nil && n == 0 {
			return nil
		}

		objects = append(objects, &GitObject{
			Type:    obj.Type(),
			Content: content[:n],
			Hash:    hash,
		})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("iterating objects: %w", err)
	}

	return objects, nil
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

// WritePackfile writes a set of git objects to a packfile using go-git's
// encoder with delta compression (packWindow=10).
func (w *PackfileWriter) WritePackfile(objects []*GitObject) ([]byte, error) {
	// Use a fresh storage for each write
	ms := memory.NewStorage()

	// Store objects in memory storage
	var hashes []plumbing.Hash
	for _, obj := range objects {
		enc := ms.NewEncodedObject()
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

		hash, err := ms.SetEncodedObject(enc)
		if err != nil {
			return nil, fmt.Errorf("storing object: %w", err)
		}

		hashes = append(hashes, hash)
	}

	// Create packfile encoder with delta compression (window=10)
	var buf bytes.Buffer
	encoder := packfile.NewEncoder(&buf, ms, false)

	_, err := encoder.Encode(hashes, 10)
	if err != nil {
		return nil, fmt.Errorf("encoding packfile: %w", err)
	}

	return buf.Bytes(), nil
}

// ExtractObjects extracts git objects from a packfile without storing them.
// This is a convenience wrapper around PackfileReader.ReadPackfile.
func ExtractObjects(data []byte) ([]*GitObject, error) {
	reader := NewPackfileReader()
	return reader.ReadPackfile(data)
}
