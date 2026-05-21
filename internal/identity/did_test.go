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
