package discussions

import (
	"os"
	"testing"
)

func TestStore_Create(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)

	discussion := &Discussion{
		RepoID:   "repo1",
		Title:    "Test Discussion",
		Body:     "This is a test",
		Author:   "alice",
		Category: "general",
	}

	if err := store.Create(discussion); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if discussion.ID == "" {
		t.Fatal("expected non-empty ID")
	}
}

func TestStore_Get(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)

	discussion := &Discussion{
		RepoID: "repo1",
		Title:  "Test",
		Body:   "Body",
		Author: "alice",
	}
	store.Create(discussion)

	got, err := store.Get("repo1", discussion.ID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if got.Title != "Test" {
		t.Errorf("expected Title=Test, got %s", got.Title)
	}
}

func TestStore_List(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)

	store.Create(&Discussion{RepoID: "repo1", Title: "Q1", Category: "question", Author: "a"})
	store.Create(&Discussion{RepoID: "repo1", Title: "G1", Category: "general", Author: "b"})
	store.Create(&Discussion{RepoID: "repo2", Title: "Q2", Category: "question", Author: "c"})

	all := store.List("repo1", "", "")
	if len(all) != 2 {
		t.Fatalf("expected 2 discussions, got %d", len(all))
	}

	questions := store.List("repo1", "question", "")
	if len(questions) != 1 {
		t.Fatalf("expected 1 question, got %d", len(questions))
	}
}

func TestStore_AddAnswer(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)

	discussion := &Discussion{RepoID: "repo1", Title: "Q", Author: "a"}
	store.Create(discussion)

	answer := &Answer{Body: "Answer", Author: "b"}
	if err := store.AddAnswer("repo1", discussion.ID, answer); err != nil {
		t.Fatalf("AddAnswer failed: %v", err)
	}

	got, _ := store.Get("repo1", discussion.ID)
	if len(got.Answers) != 1 {
		t.Fatalf("expected 1 answer, got %d", len(got.Answers))
	}
}

func TestStore_AcceptAnswer(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)

	discussion := &Discussion{RepoID: "repo1", Title: "Q", Author: "a"}
	store.Create(discussion)

	answer := &Answer{Body: "A", Author: "b"}
	store.AddAnswer("repo1", discussion.ID, answer)

	if err := store.AcceptAnswer("repo1", discussion.ID, answer.ID); err != nil {
		t.Fatalf("AcceptAnswer failed: %v", err)
	}

	got, _ := store.Get("repo1", discussion.ID)
	if got.Status != "answered" {
		t.Errorf("expected status=answered, got %s", got.Status)
	}
}

func TestStore_Upvote(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)

	discussion := &Discussion{RepoID: "repo1", Title: "Q", Author: "a"}
	store.Create(discussion)

	store.Upvote("repo1", discussion.ID)
	store.Upvote("repo1", discussion.ID)

	got, _ := store.Get("repo1", discussion.ID)
	if got.Upvotes != 2 {
		t.Errorf("expected upvotes=2, got %d", got.Upvotes)
	}
}

func TestStore_Delete(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)

	discussion := &Discussion{RepoID: "repo1", Title: "Q", Author: "a"}
	store.Create(discussion)

	if err := store.Delete("repo1", discussion.ID); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	_, err := store.Get("repo1", discussion.ID)
	if err == nil {
		t.Fatal("expected error after delete")
	}
}

func TestStore_Persistence(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)

	store.Create(&Discussion{RepoID: "repo1", Title: "Persist", Author: "a"})

	// Create new store from same directory
	store2 := NewStore(dir)
	store2.Load()

	discussions := store2.List("repo1", "", "")
	if len(discussions) != 1 {
		t.Fatalf("expected 1 discussion after reload, got %d", len(discussions))
	}
}

func TestStore_Search(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)

	store.Create(&Discussion{RepoID: "repo1", Title: "How to use Git?", Author: "a"})
	store.Create(&Discussion{RepoID: "repo1", Title: "Best practices", Author: "b"})

	results := store.Search("repo1", "git")
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
}

func init() {
	// Ensure temp dirs are cleaned up
	os.MkdirAll("/tmp/discussions-test", 0755)
}
