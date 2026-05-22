package persistence

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// SaveJSON atomically writes v as JSON to path.
// It writes to a temp file first, then renames to prevent corruption on crash.
func SaveJSON(path string, v interface{}) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling JSON: %w", err)
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating directory: %w", err)
	}

	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return fmt.Errorf("writing temp file: %w", err)
	}

	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("renaming file: %w", err)
	}

	return nil
}

// LoadJSON reads JSON from path into v.
// If the file does not exist, it returns nil (no error) and leaves v unchanged.
func LoadJSON(path string, v interface{}) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("reading file: %w", err)
	}

	if err := json.Unmarshal(data, v); err != nil {
		return fmt.Errorf("unmarshaling JSON: %w", err)
	}

	return nil
}
