package persistence

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaveLoadJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.json")

	type sample struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}

	// Save
	err := SaveJSON(path, sample{Name: "hello", Value:42})
	if err != nil {
		t.Fatalf("SaveJSON: %v", err)
	}

	// Load
	var loaded sample
	err = LoadJSON(path, &loaded)
	if err != nil {
		t.Fatalf("LoadJSON: %v", err)
	}

	if loaded.Name != "hello" || loaded.Value != 42 {
		t.Fatalf("expected {hello 42}, got %+v", loaded)
	}
}

func TestLoadJSONMissingFile(t *testing.T) {
	var loaded map[string]string
	err := LoadJSON("/nonexistent/path.json", &loaded)
	if err != nil {
		t.Fatalf("expected nil error for missing file, got: %v", err)
	}
}

func TestSaveJSONAtomic(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "atomic.json")

	// Save initial data
	err := SaveJSON(path, map[string]int{"a": 1})
	if err != nil {
		t.Fatal(err)
	}

	// Verify .tmp file does not remain
	tmpPath := path + ".tmp"
	if _, err := os.Stat(tmpPath); !os.IsNotExist(err) {
		t.Fatal("temp file should not exist after successful save")
	}

	// Verify file is valid JSON
	var loaded map[string]int
	err = LoadJSON(path, &loaded)
	if err != nil {
		t.Fatal(err)
	}
	if loaded["a"] != 1 {
		t.Fatalf("expected a=1, got %+v", loaded)
	}
}
