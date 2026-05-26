package notifications

import (
	"testing"
)

func TestManager_Create(t *testing.T) {
	dir := t.TempDir()
	m := NewManager(dir)

	if err := m.Create("user1", TypeIssueCreated, "New Issue", "Issue created", nil); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if m.UnreadCount("user1") != 1 {
		t.Fatalf("expected 1 unread, got %d", m.UnreadCount("user1"))
	}
}

func TestManager_List(t *testing.T) {
	dir := t.TempDir()
	m := NewManager(dir)

	m.Create("user1", TypeIssueCreated, "Issue 1", "", nil)
	m.Create("user1", TypePROpened, "PR 1", "", nil)
	m.Create("user2", TypeIssueCreated, "Issue 2", "", nil)

	all := m.List("user1", false)
	if len(all) != 2 {
		t.Fatalf("expected 2 notifications, got %d", len(all))
	}

	user2 := m.List("user2", false)
	if len(user2) != 1 {
		t.Fatalf("expected 1 notification, got %d", len(user2))
	}
}

func TestManager_MarkAsRead(t *testing.T) {
	dir := t.TempDir()
	m := NewManager(dir)

	m.Create("user1", TypeIssueCreated, "Issue", "", nil)

	notifs := m.List("user1", false)
	if len(notifs) == 0 {
		t.Fatal("expected notifications")
	}

	if err := m.MarkAsRead("user1", notifs[0].ID); err != nil {
		t.Fatalf("MarkAsRead failed: %v", err)
	}

	if m.UnreadCount("user1") != 0 {
		t.Fatalf("expected 0 unread, got %d", m.UnreadCount("user1"))
	}
}

func TestManager_MarkAllAsRead(t *testing.T) {
	dir := t.TempDir()
	m := NewManager(dir)

	m.Create("user1", TypeIssueCreated, "I1", "", nil)
	m.Create("user1", TypePROpened, "P1", "", nil)

	if err := m.MarkAllAsRead("user1"); err != nil {
		t.Fatalf("MarkAllAsRead failed: %v", err)
	}

	if m.UnreadCount("user1") != 0 {
		t.Fatalf("expected 0 unread, got %d", m.UnreadCount("user1"))
	}
}

func TestManager_UnreadOnly(t *testing.T) {
	dir := t.TempDir()
	m := NewManager(dir)

	m.Create("user1", TypeIssueCreated, "I1", "", nil)
	m.Create("user1", TypePROpened, "P1", "", nil)

	notifs := m.List("user1", true)
	if len(notifs) != 2 {
		t.Fatalf("expected 2 unread, got %d", len(notifs))
	}

	notifs[0].Read = true

	unreadOnly := m.List("user1", true)
	if len(unreadOnly) != 1 {
		t.Fatalf("expected 1 unread, got %d", len(unreadOnly))
	}
}

func TestManager_Persistence(t *testing.T) {
	dir := t.TempDir()
	m := NewManager(dir)

	m.Create("user1", TypeIssueCreated, "Persist", "", nil)

	m2 := NewManager(dir)
	m2.Load()

	if m2.UnreadCount("user1") != 1 {
		t.Fatalf("expected 1 unread after reload, got %d", m2.UnreadCount("user1"))
	}
}
