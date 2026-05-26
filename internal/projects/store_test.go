package projects

import (
	"testing"
)

func TestStore_Create(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)

	project := &Project{
		RepoID:      "repo1",
		Name:        "Test Project",
		Description: "A test project",
	}

	if err := store.Create(project); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if project.ID == "" {
		t.Fatal("expected non-empty ID")
	}

	if len(project.Columns) != 3 {
		t.Fatalf("expected 3 default columns, got %d", len(project.Columns))
	}
}

func TestStore_Get(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)

	project := &Project{RepoID: "repo1", Name: "Test"}
	store.Create(project)

	got, err := store.Get("repo1", project.ID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if got.Name != "Test" {
		t.Errorf("expected Name=Test, got %s", got.Name)
	}
}

func TestStore_List(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)

	store.Create(&Project{RepoID: "repo1", Name: "P1"})
	store.Create(&Project{RepoID: "repo1", Name: "P2"})
	store.Create(&Project{RepoID: "repo2", Name: "P3"})

	all := store.List("repo1", "")
	if len(all) != 2 {
		t.Fatalf("expected 2 projects, got %d", len(all))
	}
}

func TestStore_AddCard(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)

	project := &Project{RepoID: "repo1", Name: "Test"}
	store.Create(project)

	card := &Card{
		Title:       "Task 1",
		Description: "Do something",
		Assignee:    "alice",
	}

	if err := store.AddCard("repo1", project.ID, project.Columns[0].ID, card); err != nil {
		t.Fatalf("AddCard failed: %v", err)
	}

	got, _ := store.Get("repo1", project.ID)
	if len(got.Columns[0].Cards) != 1 {
		t.Fatalf("expected 1 card, got %d", len(got.Columns[0].Cards))
	}
}

func TestStore_MoveCard(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)

	project := &Project{RepoID: "repo1", Name: "Test"}
	store.Create(project)

	card := &Card{Title: "Task 1"}
	store.AddCard("repo1", project.ID, project.Columns[0].ID, card)

	if err := store.MoveCard("repo1", project.ID, card.ID, project.Columns[1].ID, 0); err != nil {
		t.Fatalf("MoveCard failed: %v", err)
	}

	got, _ := store.Get("repo1", project.ID)
	if len(got.Columns[0].Cards) != 0 {
		t.Error("expected 0 cards in source column")
	}
	if len(got.Columns[1].Cards) != 1 {
		t.Error("expected 1 card in target column")
	}
}

func TestStore_DeleteCard(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)

	project := &Project{RepoID: "repo1", Name: "Test"}
	store.Create(project)

	card := &Card{Title: "Task 1"}
	store.AddCard("repo1", project.ID, project.Columns[0].ID, card)

	if err := store.DeleteCard("repo1", project.ID, card.ID); err != nil {
		t.Fatalf("DeleteCard failed: %v", err)
	}

	got, _ := store.Get("repo1", project.ID)
	if len(got.Columns[0].Cards) != 0 {
		t.Error("expected 0 cards after delete")
	}
}

func TestStore_Delete(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)

	project := &Project{RepoID: "repo1", Name: "Test"}
	store.Create(project)

	if err := store.Delete("repo1", project.ID); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	_, err := store.Get("repo1", project.ID)
	if err == nil {
		t.Fatal("expected error after delete")
	}
}

func TestStore_Persistence(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)

	store.Create(&Project{RepoID: "repo1", Name: "Persist"})

	store2 := NewStore(dir)
	store2.Load()

	projects := store2.List("repo1", "")
	if len(projects) != 1 {
		t.Fatalf("expected 1 project after reload, got %d", len(projects))
	}
}
