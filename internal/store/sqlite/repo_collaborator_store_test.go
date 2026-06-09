package sqlite

import (
	"context"
	"testing"

	"github.com/lakshmanpatel/gitant/internal/store"
)

func TestRepoCollaboratorStore_AddAndAuthorize(t *testing.T) {
	sqliteStore, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer sqliteStore.Close()

	acl := sqliteStore.NewRepoCollaboratorStore()
	ctx := context.Background()
	if err := acl.Add(ctx, &store.RepoCollaborator{RepoID: "repo-1", UserID: "user-1", Role: store.RepoRoleOwner}); err != nil {
		t.Fatal(err)
	}

	allowed, err := acl.IsWriter(ctx, "repo-1", "user-1")
	if err != nil {
		t.Fatal(err)
	}
	if !allowed {
		t.Fatal("expected owner to be allowed to write")
	}

	allowed, err = acl.IsWriter(ctx, "repo-1", "stranger")
	if err != nil {
		t.Fatal(err)
	}
	if allowed {
		t.Fatal("expected stranger to be denied")
	}
}

func TestRepoCollaboratorStore_PersistsAcrossReopen(t *testing.T) {
	dataDir := t.TempDir()
	ctx := context.Background()

	sqliteStore, err := NewStore(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	acl := sqliteStore.NewRepoCollaboratorStore()
	if err := acl.Add(ctx, &store.RepoCollaborator{RepoID: "repo-1", UserID: "user-1", Role: store.RepoRoleCollaborator}); err != nil {
		t.Fatal(err)
	}
	if err := sqliteStore.Close(); err != nil {
		t.Fatal(err)
	}

	sqliteStore, err = NewStore(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	defer sqliteStore.Close()
	acl = sqliteStore.NewRepoCollaboratorStore()

	allowed, err := acl.IsWriter(ctx, "repo-1", "user-1")
	if err != nil {
		t.Fatal(err)
	}
	if !allowed {
		t.Fatal("expected collaborator membership to persist")
	}
}
