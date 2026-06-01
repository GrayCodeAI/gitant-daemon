package handlers

import (
	"testing"

	"github.com/lakshmanpatel/gitant/internal/webhooks"
)

func TestDispatchPushEvent_CallsOnRepoChanged(t *testing.T) {
	orig := OnRepoChanged
	defer func() { OnRepoChanged = orig }()

	called := false
	var gotRepo string
	OnRepoChanged = func(repoID string) {
		called = true
		gotRepo = repoID
	}

	wm := webhooks.NewManager()
	dispatchPushEvent(wm, "my-repo", []string{"hash1"}, nil)

	if !called {
		t.Fatal("OnRepoChanged was not called")
	}
	if gotRepo != "my-repo" {
		t.Fatalf("expected repo 'my-repo', got %q", gotRepo)
	}
}

func TestDispatchPushEvent_NilOnRepoChanged(t *testing.T) {
	orig := OnRepoChanged
	defer func() { OnRepoChanged = orig }()
	OnRepoChanged = nil

	// Should not panic when OnRepoChanged is nil.
	wm := webhooks.NewManager()
	dispatchPushEvent(wm, "repo", []string{"h1"}, nil)
}
