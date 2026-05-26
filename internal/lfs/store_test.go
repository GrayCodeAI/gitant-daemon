package lfs

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"testing"
)

func TestStore_Upload(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)
	store.Init()

	content := []byte("Hello, LFS!")
	hash := sha256.Sum256(content)
	oid := hex.EncodeToString(hash[:])

	reader := bytes.NewReader(content)
	obj, err := store.Upload("repo1", oid, reader)
	if err != nil {
		t.Fatalf("Upload failed: %v", err)
	}

	if obj.OID != oid {
		t.Errorf("expected OID=%s, got %s", oid, obj.OID)
	}
	if obj.Size != int64(len(content)) {
		t.Errorf("expected Size=%d, got %d", len(content), obj.Size)
	}
}

func TestStore_Download(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)
	store.Init()

	content := []byte("Test content")
	hash := sha256.Sum256(content)
	oid := hex.EncodeToString(hash[:])

	store.Upload("repo1", oid, bytes.NewReader(content))

	reader, obj, err := store.Download(oid)
	if err != nil {
		t.Fatalf("Download failed: %v", err)
	}
	defer reader.Close()

	downloaded, _ := io.ReadAll(reader)
	if !bytes.Equal(downloaded, content) {
		t.Error("downloaded content mismatch")
	}

	if obj.Size != int64(len(content)) {
		t.Errorf("expected Size=%d, got %d", len(content), obj.Size)
	}
}

func TestStore_Exists(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)
	store.Init()

	content := []byte("Test")
	hash := sha256.Sum256(content)
	oid := hex.EncodeToString(hash[:])

	if store.Exists(oid) {
		t.Error("expected object to not exist")
	}

	store.Upload("repo1", oid, bytes.NewReader(content))

	if !store.Exists(oid) {
		t.Error("expected object to exist")
	}
}

func TestStore_Delete(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)
	store.Init()

	content := []byte("Delete me")
	hash := sha256.Sum256(content)
	oid := hex.EncodeToString(hash[:])

	store.Upload("repo1", oid, bytes.NewReader(content))

	if err := store.Delete(oid); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	if store.Exists(oid) {
		t.Error("expected object to be deleted")
	}
}

func TestStore_Batch(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)
	store.Init()

	content1 := []byte("File 1")
	content2 := []byte("File 2")
	hash1 := sha256.Sum256(content1)
	hash2 := sha256.Sum256(content2)
	oid1 := hex.EncodeToString(hash1[:])
	oid2 := hex.EncodeToString(hash2[:])

	store.Upload("repo1", oid1, bytes.NewReader(content1))
	store.Upload("repo1", oid2, bytes.NewReader(content2))

	objects := store.Batch([]string{oid1, oid2, "nonexistent"})
	if len(objects) != 2 {
		t.Fatalf("expected 2 objects, got %d", len(objects))
	}
}

func TestStore_DuplicateUpload(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)
	store.Init()

	content := []byte("Same content")
	hash := sha256.Sum256(content)
	oid := hex.EncodeToString(hash[:])

	store.Upload("repo1", oid, bytes.NewReader(content))
	obj, err := store.Upload("repo1", oid, bytes.NewReader(content))
	if err != nil {
		t.Fatalf("Duplicate upload failed: %v", err)
	}

	if obj.Size != int64(len(content)) {
		t.Errorf("expected Size=%d, got %d", len(content), obj.Size)
	}
}
