package wiki

import (
	"testing"
)

func TestWiki_CreatePage(t *testing.T) {
	dir := t.TempDir()
	w := NewWiki(dir, "repo1")

	page, err := w.CreatePage("getting-started", "# Getting Started\n\nWelcome!", "alice")
	if err != nil {
		t.Fatalf("CreatePage failed: %v", err)
	}

	if page.Title != "Getting Started" {
		t.Errorf("expected Title=Getting Started, got %s", page.Title)
	}
}

func TestWiki_GetPage(t *testing.T) {
	dir := t.TempDir()
	w := NewWiki(dir, "repo1")

	w.CreatePage("test-page", "# Test\nContent", "alice")

	page, err := w.GetPage("test-page")
	if err != nil {
		t.Fatalf("GetPage failed: %v", err)
	}

	if page.Content != "# Test\nContent" {
		t.Errorf("content mismatch")
	}
}

func TestWiki_ListPages(t *testing.T) {
	dir := t.TempDir()
	w := NewWiki(dir, "repo1")

	w.CreatePage("page-a", "# Page A", "alice")
	w.CreatePage("page-b", "# Page B", "bob")

	pages, err := w.ListPages()
	if err != nil {
		t.Fatalf("ListPages failed: %v", err)
	}

	if len(pages) != 2 {
		t.Fatalf("expected 2 pages, got %d", len(pages))
	}
}

func TestWiki_UpdatePage(t *testing.T) {
	dir := t.TempDir()
	w := NewWiki(dir, "repo1")

	w.CreatePage("test", "# Original", "alice")

	page, err := w.UpdatePage("test", "# Updated\nNew content", "bob")
	if err != nil {
		t.Fatalf("UpdatePage failed: %v", err)
	}

	if page.Title != "Updated" {
		t.Errorf("expected Title=Updated, got %s", page.Title)
	}
}

func TestWiki_DeletePage(t *testing.T) {
	dir := t.TempDir()
	w := NewWiki(dir, "repo1")

	w.CreatePage("test", "# Test", "alice")

	if err := w.DeletePage("test"); err != nil {
		t.Fatalf("DeletePage failed: %v", err)
	}

	_, err := w.GetPage("test")
	if err == nil {
		t.Fatal("expected error after delete")
	}
}

func TestWiki_Search(t *testing.T) {
	dir := t.TempDir()
	w := NewWiki(dir, "repo1")

	w.CreatePage("getting-started", "# Getting Started\nLearn how to use", "alice")
	w.CreatePage("api-reference", "# API Reference\nEndpoints", "bob")

	results, err := w.Search("api")
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
}
