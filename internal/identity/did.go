package identity

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// Identity represents a DID:key identity
type Identity struct {
	mu sync.RWMutex

	PublicKey  ed25519.PublicKey
	PrivateKey ed25519.PrivateKey
	DID        string
	path       string
}

// NewIdentity creates a new DID:key identity
func NewIdentity() (*Identity, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generating keypair: %w", err)
	}

	did := fmt.Sprintf("did:key:z%s", base64.RawURLEncoding.EncodeToString(pub))

	return &Identity{
		PublicKey:  pub,
		PrivateKey: priv,
		DID:        did,
	}, nil
}

// LoadIdentity loads an identity from disk
func LoadIdentity(path string) (*Identity, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading identity file: %w", err)
	}

	// Decode the private key
	privBytes, err := base64.RawURLEncoding.DecodeString(string(data))
	if err != nil {
		return nil, fmt.Errorf("decoding private key: %w", err)
	}

	// Convert to ed25519 private key
	priv := ed25519.PrivateKey(privBytes)

	// Derive public key
	pub := priv.Public().(ed25519.PublicKey)

	did := fmt.Sprintf("did:key:z%s", base64.RawURLEncoding.EncodeToString(pub))

	return &Identity{
		PublicKey:  pub,
		PrivateKey: priv,
		DID:        did,
		path:       path,
	}, nil
}

// Save saves the identity to disk
func (i *Identity) Save(path string) error {
	i.mu.Lock()
	defer i.mu.Unlock()

	// Encode private key
	encoded := base64.RawURLEncoding.EncodeToString(i.PrivateKey)

	// Create directory if needed
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("creating directory: %w", err)
	}

	// Write to file with restrictive permissions
	if err := os.WriteFile(path, []byte(encoded), 0600); err != nil {
		return fmt.Errorf("writing identity file: %w", err)
	}

	i.path = path
	return nil
}

// Sign signs a message with the private key
func (i *Identity) Sign(message []byte) []byte {
	i.mu.RLock()
	defer i.mu.RUnlock()

	return ed25519.Sign(i.PrivateKey, message)
}

// Verify verifies a signature with the public key
func (i *Identity) Verify(message, signature []byte) bool {
	i.mu.RLock()
	defer i.mu.RUnlock()

	return ed25519.Verify(i.PublicKey, message, signature)
}

// DIDDocument returns the DID document for this identity
func (i *Identity) DIDDocument() map[string]interface{} {
	i.mu.RLock()
	defer i.mu.RUnlock()

	return map[string]interface{}{
		"@context": []string{
			"https://www.w3.org/ns/did/v1",
			"https://w3id.org/security/suites/ed25519-2020/v1",
		},
		"id": i.DID,
		"verificationMethod": []map[string]interface{}{
			{
				"id":                 i.DID + "#controller",
				"type":               "Ed25519VerificationKey2020",
				"controller":         i.DID,
				"publicKeyMultibase": base64.RawURLEncoding.EncodeToString(i.PublicKey),
			},
		},
		"authentication": []string{i.DID + "#controller"},
	}
}

// String returns the DID as a string
func (i *Identity) String() string {
	return i.DID
}
