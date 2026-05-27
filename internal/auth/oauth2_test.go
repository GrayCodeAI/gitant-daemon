package auth

import (
	"testing"
	"time"
)

func TestAuthManager_GenerateAPIKey(t *testing.T) {
	mgr := NewAuthManager()

	key, rawKey, err := mgr.GenerateAPIKey("did:key:test", "test-key", []string{"read", "write"}, nil)
	if err != nil {
		t.Fatalf("GenerateAPIKey failed: %v", err)
	}

	if key.ID == "" {
		t.Error("Expected non-empty key ID")
	}
	if key.DID != "did:key:test" {
		t.Errorf("Expected DID 'did:key:test', got '%s'", key.DID)
	}
	if !key.Active {
		t.Error("Expected key to be active")
	}
	if rawKey == "" {
		t.Error("Expected non-empty raw key")
	}
}

func TestAuthManager_ValidateAPIKey(t *testing.T) {
	mgr := NewAuthManager()

	_, rawKey, err := mgr.GenerateAPIKey("did:key:test", "test-key", []string{"read"}, nil)
	if err != nil {
		t.Fatalf("GenerateAPIKey failed: %v", err)
	}

	validated, err := mgr.ValidateAPIKey(rawKey)
	if err != nil {
		t.Fatalf("ValidateAPIKey failed: %v", err)
	}

	if validated.DID != "did:key:test" {
		t.Errorf("Expected DID 'did:key:test', got '%s'", validated.DID)
	}
}

func TestAuthManager_ValidateAPIKey_Invalid(t *testing.T) {
	mgr := NewAuthManager()

	_, err := mgr.ValidateAPIKey("invalid-key")
	if err == nil {
		t.Error("Expected error for invalid key")
	}
}

func TestAuthManager_RevokeAPIKey(t *testing.T) {
	mgr := NewAuthManager()

	key, _, err := mgr.GenerateAPIKey("did:key:test", "test-key", []string{"read"}, nil)
	if err != nil {
		t.Fatalf("GenerateAPIKey failed: %v", err)
	}

	if err := mgr.RevokeAPIKey(key.ID); err != nil {
		t.Fatalf("RevokeAPIKey failed: %v", err)
	}

	keys := mgr.ListAPIKeys("did:key:test")
	if len(keys) != 1 {
		t.Fatalf("Expected 1 key, got %d", len(keys))
	}
	if keys[0].Active {
		t.Error("Expected key to be inactive after revocation")
	}
}

func TestAuthManager_ListAPIKeys(t *testing.T) {
	mgr := NewAuthManager()

	mgr.GenerateAPIKey("did:key:user1", "key1", []string{"read"}, nil)
	mgr.GenerateAPIKey("did:key:user1", "key2", []string{"write"}, nil)
	mgr.GenerateAPIKey("did:key:user2", "key3", []string{"read"}, nil)

	keys := mgr.ListAPIKeys("did:key:user1")
	if len(keys) != 2 {
		t.Errorf("Expected 2 keys for user1, got %d", len(keys))
	}

	keys = mgr.ListAPIKeys("did:key:user2")
	if len(keys) != 1 {
		t.Errorf("Expected 1 key for user2, got %d", len(keys))
	}
}

func TestAuthManager_CreateSession(t *testing.T) {
	mgr := NewAuthManager()

	session := mgr.CreateSession("did:key:test", "127.0.0.1", "test-agent", 24*time.Hour)
	if session.ID == "" {
		t.Error("Expected non-empty session ID")
	}
	if session.DID != "did:key:test" {
		t.Errorf("Expected DID 'did:key:test', got '%s'", session.DID)
	}
}

func TestAuthManager_ValidateSession(t *testing.T) {
	mgr := NewAuthManager()

	session := mgr.CreateSession("did:key:test", "127.0.0.1", "test-agent", 24*time.Hour)

	validated, err := mgr.ValidateSession(session.ID)
	if err != nil {
		t.Fatalf("ValidateSession failed: %v", err)
	}
	if validated.DID != "did:key:test" {
		t.Errorf("Expected DID 'did:key:test', got '%s'", validated.DID)
	}
}

func TestAuthManager_ValidateSession_Expired(t *testing.T) {
	mgr := NewAuthManager()

	session := mgr.CreateSession("did:key:test", "127.0.0.1", "test-agent", -1*time.Hour)

	_, err := mgr.ValidateSession(session.ID)
	if err == nil {
		t.Error("Expected error for expired session")
	}
}

func TestAuthManager_CreateOAuth2State(t *testing.T) {
	mgr := NewAuthManager()

	state := mgr.CreateOAuth2State("github", "did:key:test")
	if state == "" {
		t.Error("Expected non-empty state")
	}
}

func TestAuthManager_ValidateOAuth2State(t *testing.T) {
	mgr := NewAuthManager()

	state := mgr.CreateOAuth2State("github", "did:key:test")

	validated, err := mgr.ValidateOAuth2State(state)
	if err != nil {
		t.Fatalf("ValidateOAuth2State failed: %v", err)
	}
	if validated.Provider != "github" {
		t.Errorf("Expected provider 'github', got '%s'", validated.Provider)
	}
}

func TestAuthManager_ValidateOAuth2State_Used(t *testing.T) {
	mgr := NewAuthManager()

	state := mgr.CreateOAuth2State("github", "did:key:test")

	_, err := mgr.ValidateOAuth2State(state)
	if err != nil {
		t.Fatalf("ValidateOAuth2State failed: %v", err)
	}

	_, err = mgr.ValidateOAuth2State(state)
	if err == nil {
		t.Error("Expected error for used state")
	}
}

func TestAuthManager_RegisterOAuth2Provider(t *testing.T) {
	mgr := NewAuthManager()

	mgr.RegisterOAuth2Provider(&OAuth2Provider{
		Name:    "github",
		Enabled: true,
	})

	providers := mgr.GetOAuth2Providers()
	if len(providers) != 1 {
		t.Errorf("Expected 1 provider, got %d", len(providers))
	}
	if providers[0].Name != "github" {
		t.Errorf("Expected provider 'github', got '%s'", providers[0].Name)
	}
}
