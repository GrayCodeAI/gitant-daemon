package packages

import (
	"testing"
)

func TestRegistry_Publish(t *testing.T) {
	dir := t.TempDir()
	registry := NewRegistry(dir)

	pkg := &Package{
		Name:        "test-package",
		Description: "A test package",
		RepoID:      "repo1",
		Versions: map[string]*Version{
			"1.0.0": {
				Version:     "1.0.0",
				Description: "Initial release",
				Author:      "alice",
			},
		},
	}

	if err := registry.Publish(pkg); err != nil {
		t.Fatalf("Publish failed: %v", err)
	}
}

func TestRegistry_Get(t *testing.T) {
	dir := t.TempDir()
	registry := NewRegistry(dir)

	pkg := &Package{
		Name:    "test-pkg",
		Versions: map[string]*Version{
			"1.0.0": {Version: "1.0.0"},
		},
	}
	registry.Publish(pkg)

	got, err := registry.Get("test-pkg")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if got.Name != "test-pkg" {
		t.Errorf("expected Name=test-pkg, got %s", got.Name)
	}
}

func TestRegistry_GetVersion(t *testing.T) {
	dir := t.TempDir()
	registry := NewRegistry(dir)

	pkg := &Package{
		Name: "test-pkg",
		Versions: map[string]*Version{
			"1.0.0": {Version: "1.0.0", Author: "alice"},
			"2.0.0": {Version: "2.0.0", Author: "bob"},
		},
	}
	registry.Publish(pkg)

	ver, err := registry.GetVersion("test-pkg", "1.0.0")
	if err != nil {
		t.Fatalf("GetVersion failed: %v", err)
	}

	if ver.Author != "alice" {
		t.Errorf("expected Author=alice, got %s", ver.Author)
	}
}

func TestRegistry_List(t *testing.T) {
	dir := t.TempDir()
	registry := NewRegistry(dir)

	registry.Publish(&Package{Name: "pkg1", Versions: map[string]*Version{}})
	registry.Publish(&Package{Name: "pkg2", Versions: map[string]*Version{}})

	pkgs := registry.List()
	if len(pkgs) != 2 {
		t.Fatalf("expected 2 packages, got %d", len(pkgs))
	}
}

func TestRegistry_Search(t *testing.T) {
	dir := t.TempDir()
	registry := NewRegistry(dir)

	registry.Publish(&Package{Name: "git-helper", Description: "Git utilities", Versions: map[string]*Version{}})
	registry.Publish(&Package{Name: "web-framework", Description: "Web framework", Versions: map[string]*Version{}})

	results := registry.Search("git")
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
}

func TestRegistry_Delete(t *testing.T) {
	dir := t.TempDir()
	registry := NewRegistry(dir)

	registry.Publish(&Package{Name: "test-pkg", Versions: map[string]*Version{}})

	if err := registry.Delete("test-pkg"); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	_, err := registry.Get("test-pkg")
	if err == nil {
		t.Fatal("expected error after delete")
	}
}
