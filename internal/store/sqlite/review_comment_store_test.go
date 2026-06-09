package sqlite

import (
	"context"
	"testing"
	"time"

	"github.com/lakshmanpatel/gitant/internal/store"
)

func TestReviewCommentStoreLifecyclePersistsAcrossReopen(t *testing.T) {
	ctx := context.Background()
	dataDir := t.TempDir()

	s, err := NewStore(dataDir)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	reviews := s.NewReviewCommentStore()

	comment := &store.ReviewComment{
		ID:         "rc-test",
		PRID:       "pr-1",
		FilePath:   "main.go",
		LineNumber: 42,
		AuthorID:   "user-1",
		Body:       "please fix this",
		Status:     "open",
	}
	if err := reviews.Create(ctx, comment); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if comment.CreatedAt.IsZero() || comment.UpdatedAt.IsZero() {
		t.Fatalf("Create should fill timestamps, got created=%v updated=%v", comment.CreatedAt, comment.UpdatedAt)
	}
	if err := s.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	s, err = NewStore(dataDir)
	if err != nil {
		t.Fatalf("reopen NewStore: %v", err)
	}
	defer s.Close()
	reviews = s.NewReviewCommentStore()

	comments, err := reviews.ListByPR(ctx, "pr-1")
	if err != nil {
		t.Fatalf("ListByPR after reopen: %v", err)
	}
	if len(comments) != 1 || comments[0].ID != comment.ID {
		t.Fatalf("expected persisted comment %q, got %#v", comment.ID, comments)
	}
	if comments[0].Status != "open" {
		t.Fatalf("expected open status, got %q", comments[0].Status)
	}

	if err := reviews.Resolve(ctx, comment.ID); err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	resolved, err := reviews.Get(ctx, comment.ID)
	if err != nil {
		t.Fatalf("Get resolved: %v", err)
	}
	if resolved.Status != "resolved" {
		t.Fatalf("expected resolved status, got %q", resolved.Status)
	}
	if !resolved.UpdatedAt.After(comments[0].UpdatedAt) && !resolved.UpdatedAt.Equal(comments[0].UpdatedAt.Truncate(time.Second)) {
		t.Fatalf("expected updated timestamp to advance, before=%v after=%v", comments[0].UpdatedAt, resolved.UpdatedAt)
	}

	if err := reviews.Delete(ctx, comment.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	comments, err = reviews.ListByPR(ctx, "pr-1")
	if err != nil {
		t.Fatalf("ListByPR after delete: %v", err)
	}
	if len(comments) != 0 {
		t.Fatalf("expected no comments after delete, got %#v", comments)
	}
}
