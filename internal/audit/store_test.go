package audit

import (
	"testing"
	"time"
)

func TestStore_Record(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)

	if err := store.Record(EventRepoCreated, "alice", "repo:myrepo", "created", nil); err != nil {
		t.Fatalf("Record failed: %v", err)
	}

	if store.Count() != 1 {
		t.Fatalf("expected 1 event, got %d", store.Count())
	}
}

func TestStore_List(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)

	store.Record(EventRepoCreated, "alice", "repo:r1", "created", nil)
	store.Record(EventIssueCreated, "alice", "repo:r1", "created", nil)
	store.Record(EventRepoCreated, "bob", "repo:r2", "created", nil)

	all := store.List("", "", 0)
	if len(all) != 3 {
		t.Fatalf("expected 3 events, got %d", len(all))
	}

	alice := store.List("", "alice", 0)
	if len(alice) != 2 {
		t.Fatalf("expected 2 alice events, got %d", len(alice))
	}

	repos := store.List(EventRepoCreated, "", 0)
	if len(repos) != 2 {
		t.Fatalf("expected 2 repo events, got %d", len(repos))
	}
}

func TestStore_ListByRepo(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)

	// Record events with RepoID set directly
	store.events = append(store.events, Event{
		ID:        "1",
		Type:      EventRepoCreated,
		Actor:     "alice",
		RepoID:    "r1",
		Resource:  "repo:r1",
		Action:    "created",
		Timestamp: timeNow(),
	})
	store.events = append(store.events, Event{
		ID:        "2",
		Type:      EventIssueCreated,
		Actor:     "alice",
		RepoID:    "r1",
		Resource:  "issue:i1",
		Action:    "created",
		Timestamp: timeNow(),
	})
	store.events = append(store.events, Event{
		ID:        "3",
		Type:      EventRepoCreated,
		Actor:     "bob",
		RepoID:    "r2",
		Resource:  "repo:r2",
		Action:    "created",
		Timestamp: timeNow(),
	})

	events := store.ListByRepo("r1", 0)
	if len(events) != 2 {
		t.Fatalf("expected 2 r1 events, got %d", len(events))
	}
}

func TestStore_Limit(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)

	for i := 0; i < 10; i++ {
		store.Record(EventRepoCreated, "alice", "repo:r1", "created", nil)
	}

	events := store.List("", "", 5)
	if len(events) != 5 {
		t.Fatalf("expected 5 events, got %d", len(events))
	}
}

func TestStore_Persistence(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)

	store.Record(EventRepoCreated, "alice", "repo:r1", "created", nil)

	store2 := NewStore(dir)
	store2.Load()

	if store2.Count() != 1 {
		t.Fatalf("expected 1 event after reload, got %d", store2.Count())
	}
}

func TestStore_RecordWithRequest(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)

	if err := store.RecordWithRequest(
		EventUserCreated,
		"admin",
		"user:alice",
		"created",
		map[string]interface{}{"username": "alice"},
		"127.0.0.1",
		"curl/7.0",
	); err != nil {
		t.Fatalf("RecordWithRequest failed: %v", err)
	}

	events := store.List("", "", 0)
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	event := events[0]
	if event.IP != "127.0.0.1" {
		t.Errorf("expected IP=127.0.0.1, got %s", event.IP)
	}
	if event.UserAgent != "curl/7.0" {
		t.Errorf("expected UserAgent=curl/7.0, got %s", event.UserAgent)
	}
}

func timeNow() time.Time {
	return time.Now()
}
