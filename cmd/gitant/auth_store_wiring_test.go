package main

import (
	"context"
	"testing"

	"github.com/lakshmanpatel/gitant/internal/store"
)

func TestNewServeAuthServiceUsesDurableSQLiteStores(t *testing.T) {
	dataDir := t.TempDir()
	ctx := context.Background()

	auth, closeAuth, err := newServeAuthService(dataDir)
	if err != nil {
		t.Fatalf("newServeAuthService failed: %v", err)
	}

	user, err := auth.Register(ctx, store.RegisterInput{
		Username:    "durable-user",
		Email:       "durable@example.com",
		Password:    "password123",
		DisplayName: "Durable User",
	})
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}

	session, err := auth.Login(ctx, store.LoginInput{Username: "durable-user", Password: "password123"})
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}
	if session.Token == "" {
		t.Fatal("expected session token")
	}
	if closeAuth != nil {
		closeAuth()
	}

	reopenedAuth, closeReopened, err := newServeAuthService(dataDir)
	if err != nil {
		t.Fatalf("reopen newServeAuthService failed: %v", err)
	}
	defer closeReopened()

	loggedIn, err := reopenedAuth.Login(ctx, store.LoginInput{Username: "durable-user", Password: "password123"})
	if err != nil {
		t.Fatalf("login after reopen failed: %v", err)
	}
	if loggedIn.Token == "" || loggedIn.Token == session.Token {
		t.Fatalf("expected a new non-empty token after reopen, got %q (old %q)", loggedIn.Token, session.Token)
	}

	profile, err := reopenedAuth.ValidateSession(ctx, session.Token)
	if err != nil {
		t.Fatalf("pre-reopen session should validate after reopen: %v", err)
	}
	if profile.ID != user.ID || profile.Username != "durable-user" {
		t.Fatalf("validated wrong user: got id=%q username=%q, want id=%q username=%q", profile.ID, profile.Username, user.ID, "durable-user")
	}
}
