package identity

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
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

func TestResolveDIDWebBlocksInternalHostsBeforeFetch(t *testing.T) {
	tests := []string{
		"did:web:localhost",
		"did:web:127.0.0.1%3A7777",
		"did:web:127.0.0.1%3a7777",
		"did:web:::1",
		"did:web:10.0.0.1",
		"did:web:172.16.0.1",
		"did:web:192.168.1.1",
		"did:web:169.254.169.254",
	}

	for _, did := range tests {
		t.Run(did, func(t *testing.T) {
			var called atomic.Bool
			oldClient := didWebClient
			didWebClient = &http.Client{
				Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
					called.Store(true)
					return nil, nil
				}),
			}
			t.Cleanup(func() { didWebClient = oldClient })

			if _, err := ResolveDIDWeb(did); err == nil {
				t.Fatal("expected internal did:web host to be rejected")
			}
			if called.Load() {
				t.Fatal("expected internal did:web host to be rejected before any outbound fetch")
			}
		})
	}
}

func TestResolveDIDWebRejectsRedirectsToInternalHosts(t *testing.T) {
	did := "did:web:example.com"
	var internalFetches atomic.Int32
	oldClient := didWebClient
	didWebClient = &http.Client{
		CheckRedirect: didWebRedirectPolicy,
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			if req.URL.Hostname() == "127.0.0.1" {
				internalFetches.Add(1)
			}
			return &http.Response{
				StatusCode: http.StatusFound,
				Header:     http.Header{"Location": []string{"https://127.0.0.1/.well-known/did.json"}},
				Body:       io.NopCloser(strings.NewReader("")),
				Request:    req,
			}, nil
		}),
	}
	t.Cleanup(func() { didWebClient = oldClient })

	if _, err := ResolveDIDWeb(did); err == nil {
		t.Fatal("expected redirect to internal host to be rejected")
	}
	if internalFetches.Load() != 0 {
		t.Fatal("expected redirect target not to be fetched")
	}
}

func TestResolveDIDWebRejectsHostOutsideAllowlist(t *testing.T) {
	t.Setenv("GITANT_DID_WEB_ALLOWLIST", "example.com")
	var called atomic.Bool
	oldClient := didWebClient
	didWebClient = &http.Client{
		Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			called.Store(true)
			return nil, nil
		}),
	}
	t.Cleanup(func() { didWebClient = oldClient })

	if _, err := ResolveDIDWeb("did:web:example.org"); err == nil {
		t.Fatal("expected non-allowlisted did:web host to be rejected")
	}
	if called.Load() {
		t.Fatal("expected non-allowlisted did:web host to be rejected before any outbound fetch")
	}
}

func TestResolveDIDWebAllowsSafePublicHost(t *testing.T) {
	t.Setenv("GITANT_DID_WEB_ALLOWLIST", "example.com")
	did := "did:web:example.com:user:alice"
	oldClient := didWebClient
	didWebClient = &http.Client{
		CheckRedirect: didWebRedirectPolicy,
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			if req.URL.String() != "https://example.com/user/alice/did.json" {
				t.Fatalf("unexpected resolve URL: %s", req.URL.String())
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body:       io.NopCloser(strings.NewReader(`{"id":"did:web:example.com:user:alice"}`)),
				Request:    req,
			}, nil
		}),
	}
	t.Cleanup(func() { didWebClient = oldClient })

	doc, err := ResolveDIDWeb(did)
	if err != nil {
		t.Fatalf("expected safe public did:web to resolve: %v", err)
	}
	if doc["id"] != did {
		t.Fatalf("expected document id %s, got %v", did, doc["id"])
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}
