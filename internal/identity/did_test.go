package identity

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewIdentity(t *testing.T) {
	id, err := NewIdentity()
	if err != nil {
		t.Fatal(err)
	}

	if id.DID == "" {
		t.Fatal("expected non-empty DID")
	}

	if len(id.PublicKey) == 0 {
		t.Fatal("expected non-empty public key")
	}

	if len(id.PrivateKey) == 0 {
		t.Fatal("expected non-empty private key")
	}
}

func TestSignVerify(t *testing.T) {
	id, err := NewIdentity()
	if err != nil {
		t.Fatal(err)
	}

	message := []byte("Hello, gitant!")

	// Sign
	signature := id.Sign(message)

	// Verify
	if !id.Verify(message, signature) {
		t.Fatal("expected signature to be valid")
	}

	// Verify with wrong message
	if id.Verify([]byte("wrong message"), signature) {
		t.Fatal("expected signature to be invalid")
	}
}

func TestSaveLoad(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "gitant-identity-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create identity
	id, err := NewIdentity()
	if err != nil {
		t.Fatal(err)
	}

	// Save
	path := filepath.Join(tmpDir, "identity.key")
	err = id.Save(path)
	if err != nil {
		t.Fatal(err)
	}

	// Load
	loaded, err := LoadIdentity(path)
	if err != nil {
		t.Fatal(err)
	}

	// Verify
	if loaded.DID != id.DID {
		t.Fatalf("expected %s, got %s", id.DID, loaded.DID)
	}

	// Test sign/verify with loaded identity
	message := []byte("test message")
	signature := loaded.Sign(message)
	if !id.Verify(message, signature) {
		t.Fatal("expected signature to be valid")
	}
}

func TestDIDDocument(t *testing.T) {
	id, err := NewIdentity()
	if err != nil {
		t.Fatal(err)
	}

	doc := id.DIDDocument()

	if doc["id"] != id.DID {
		t.Fatal("expected DID to match")
	}

	vm, ok := doc["verificationMethod"].([]map[string]interface{})
	if !ok {
		t.Fatal("expected verificationMethod to be a slice")
	}

	if len(vm) == 0 {
		t.Fatal("expected at least one verification method")
	}
}

func TestUCAN(t *testing.T) {
	issuer, err := NewIdentity()
	if err != nil {
		t.Fatal(err)
	}

	audience, err := NewIdentity()
	if err != nil {
		t.Fatal(err)
	}

	caps := []Capability{
		{
			Resource: "repo:test",
			Actions:  []string{"read", "write"},
		},
	}

	ucan := NewUCAN(issuer.DID, audience.DID, caps, 0)

	if ucan.Issuer != issuer.DID {
		t.Fatal("expected issuer to match")
	}

	if ucan.Audience != audience.DID {
		t.Fatal("expected audience to match")
	}

	if !ucan.HasCapability("repo:test", "read") {
		t.Fatal("expected to have read capability")
	}

	if !ucan.HasCapability("repo:test", "write") {
		t.Fatal("expected to have write capability")
	}

	if ucan.HasCapability("repo:test", "admin") {
		t.Fatal("expected not to have admin capability")
	}
}

func TestKeyRotation(t *testing.T) {
	id, err := NewIdentity()
	if err != nil {
		t.Fatal(err)
	}

	originalDID := id.DID
	message := []byte("test message")
	originalSig := id.Sign(message)

	// Rotate the key
	if err := id.Rotate(); err != nil {
		t.Fatal(err)
	}

	// DID should change
	if id.DID == originalDID {
		t.Fatal("expected DID to change after rotation")
	}

	// Old signature should still verify with history
	if !id.VerifyWithHistory(message, originalSig) {
		t.Fatal("expected old signature to verify with key history")
	}

	// New signature should verify
	newSig := id.Sign(message)
	if !id.VerifyWithHistory(message, newSig) {
		t.Fatal("expected new signature to verify")
	}

	// AllKnownDIDs should include both
	dids := id.AllKnownDIDs()
	if len(dids) != 2 {
		t.Fatalf("expected 2 DIDs, got %d", len(dids))
	}
	if dids[0] != originalDID {
		t.Fatalf("expected first DID to be original, got %s", dids[0])
	}
	if dids[1] != id.DID {
		t.Fatalf("expected second DID to be current, got %s", dids[1])
	}
}

func TestKeyRotationSaveLoad(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "gitant-identity-rotate-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	id, err := NewIdentity()
	if err != nil {
		t.Fatal(err)
	}

	originalDID := id.DID
	message := []byte("test message")
	originalSig := id.Sign(message)

	// Rotate and save
	if err := id.Rotate(); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(tmpDir, "identity.json")
	if err := id.Save(path); err != nil {
		t.Fatal(err)
	}

	// Load and verify key history is preserved
	loaded, err := LoadIdentity(path)
	if err != nil {
		t.Fatal(err)
	}

	if loaded.DID != id.DID {
		t.Fatalf("expected DID %s, got %s", id.DID, loaded.DID)
	}

	if len(loaded.PreviousKeys) != 1 {
		t.Fatalf("expected 1 previous key, got %d", len(loaded.PreviousKeys))
	}

	if loaded.PreviousKeys[0].DID != originalDID {
		t.Fatalf("expected previous DID %s, got %s", originalDID, loaded.PreviousKeys[0].DID)
	}

	// Old signature should still verify
	if !loaded.VerifyWithHistory(message, originalSig) {
		t.Fatal("expected old signature to verify after reload")
	}
}
