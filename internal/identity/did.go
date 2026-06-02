package identity

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Identity represents a DID:key identity
type Identity struct {
	mu sync.RWMutex

	PublicKey  ed25519.PublicKey
	PrivateKey ed25519.PrivateKey
	DID        string
	path       string
	// KeyHistory stores previous public keys for verifying old signatures.
	// Entries are ordered oldest-first.
	PreviousKeys []KeyHistoryEntry `json:"previous_keys,omitempty"`
}

// KeyHistoryEntry records a previous key in the rotation chain.
type KeyHistoryEntry struct {
	PublicKey ed25519.PublicKey `json:"public_key"`
	DID       string            `json:"did"`
	RotatedAt time.Time         `json:"rotated_at"`
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

// LoadIdentity loads an identity from disk.
// Supports both the legacy format (raw base64 private key) and the new JSON
// format (includes key history for rotation support).
func LoadIdentity(path string) (*Identity, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading identity file: %w", err)
	}

	// Try JSON format first (new format with key history)
	var file identityFile
	if err := json.Unmarshal(data, &file); err == nil && file.PrivateKey != "" {
		privBytes, err := base64.RawURLEncoding.DecodeString(file.PrivateKey)
		if err != nil {
			return nil, fmt.Errorf("decoding private key: %w", err)
		}
		priv := ed25519.PrivateKey(privBytes)
		pub := priv.Public().(ed25519.PublicKey)
		did := fmt.Sprintf("did:key:z%s", base64.RawURLEncoding.EncodeToString(pub))

		prev := make([]KeyHistoryEntry, len(file.PreviousKeys))
		for idx, entry := range file.PreviousKeys {
			pubBytes, err := base64.RawURLEncoding.DecodeString(entry.PublicKey)
			if err != nil {
				continue // skip malformed entries
			}
			prev[idx] = KeyHistoryEntry{
				PublicKey: ed25519.PublicKey(pubBytes),
				DID:       entry.DID,
				RotatedAt: entry.RotatedAt,
			}
		}

		return &Identity{
			PublicKey:    pub,
			PrivateKey:   priv,
			DID:          did,
			path:         path,
			PreviousKeys: prev,
		}, nil
	}

	// Fall back to legacy format (raw base64 private key)
	privBytes, err := base64.RawURLEncoding.DecodeString(string(data))
	if err != nil {
		return nil, fmt.Errorf("decoding private key: %w", err)
	}

	priv := ed25519.PrivateKey(privBytes)
	pub := priv.Public().(ed25519.PublicKey)
	did := fmt.Sprintf("did:key:z%s", base64.RawURLEncoding.EncodeToString(pub))

	return &Identity{
		PublicKey:  pub,
		PrivateKey: priv,
		DID:        did,
		path:       path,
	}, nil
}

// identityFile is the JSON-serializable representation of an Identity on disk.
type identityFile struct {
	PrivateKey   string           `json:"private_key"`
	PublicKey    string           `json:"public_key"`
	DID          string           `json:"did"`
	PreviousKeys []keyHistoryJSON `json:"previous_keys,omitempty"`
}

type keyHistoryJSON struct {
	PublicKey string    `json:"public_key"`
	DID       string    `json:"did"`
	RotatedAt time.Time `json:"rotated_at"`
}

// Save saves the identity to disk as JSON (includes key history).
func (i *Identity) Save(path string) error {
	i.mu.Lock()
	defer i.mu.Unlock()

	prev := make([]keyHistoryJSON, len(i.PreviousKeys))
	for idx, entry := range i.PreviousKeys {
		prev[idx] = keyHistoryJSON{
			PublicKey: base64.RawURLEncoding.EncodeToString(entry.PublicKey),
			DID:       entry.DID,
			RotatedAt: entry.RotatedAt,
		}
	}

	file := identityFile{
		PrivateKey:   base64.RawURLEncoding.EncodeToString(i.PrivateKey),
		PublicKey:    base64.RawURLEncoding.EncodeToString(i.PublicKey),
		DID:          i.DID,
		PreviousKeys: prev,
	}

	data, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling identity: %w", err)
	}

	// Create directory if needed
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("creating directory: %w", err)
	}

	// Write to file with restrictive permissions
	if err := os.WriteFile(path, data, 0600); err != nil {
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

// Rotate generates a new Ed25519 keypair, archiving the current key in history.
// The old key is retained for verifying previously-signed tokens.
// Call Save() after Rotate() to persist the new identity.
func (i *Identity) Rotate() error {
	i.mu.Lock()
	defer i.mu.Unlock()

	// Archive current key
	i.PreviousKeys = append(i.PreviousKeys, KeyHistoryEntry{
		PublicKey: i.PublicKey,
		DID:       i.DID,
		RotatedAt: time.Now(),
	})

	// Generate new keypair
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return fmt.Errorf("generating new keypair: %w", err)
	}

	i.PublicKey = pub
	i.PrivateKey = priv
	i.DID = fmt.Sprintf("did:key:z%s", base64.RawURLEncoding.EncodeToString(pub))

	return nil
}

// VerifyWithHistory verifies a signature against the current key or any previous key.
// This is used to verify tokens that were signed before a key rotation.
func (i *Identity) VerifyWithHistory(message, signature []byte) bool {
	i.mu.RLock()
	defer i.mu.RUnlock()

	// Try current key first
	if ed25519.Verify(i.PublicKey, message, signature) {
		return true
	}

	// Try previous keys
	for _, entry := range i.PreviousKeys {
		if ed25519.Verify(entry.PublicKey, message, signature) {
			return true
		}
	}

	return false
}

// AllKnownDIDs returns the current DID plus all previous DIDs from key rotations.
// Useful for checking if a DID belongs to this identity.
func (i *Identity) AllKnownDIDs() []string {
	i.mu.RLock()
	defer i.mu.RUnlock()

	dids := make([]string, 0, len(i.PreviousKeys)+1)
	for _, entry := range i.PreviousKeys {
		dids = append(dids, entry.DID)
	}
	dids = append(dids, i.DID)
	return dids
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

// didWebClient is the HTTP client used for did:web resolution.
// Uses a short timeout to prevent hanging on unresponsive servers.
var didWebClient = &http.Client{Timeout: 10 * time.Second}

// ResolveDIDWeb resolves a did:web identifier by fetching the DID document
// from the corresponding HTTPS well-known endpoint.
//
// Resolution rules (per did:web spec):
//   - did:web:example.com → https://example.com/.well-known/did.json
//   - did:web:example.com:user:alice → https://example.com/user/alice/did.json
//
// Returns the raw DID document as a map, or an error if resolution fails.
func ResolveDIDWeb(did string) (map[string]interface{}, error) {
	if !strings.HasPrefix(did, "did:web:") {
		return nil, fmt.Errorf("not a did:web identifier: %s", did)
	}

	// Parse the method-specific identifier
	msi := strings.TrimPrefix(did, "did:web:")

	// Replace : with / for path construction, but the first segment is the host
	parts := strings.Split(msi, ":")
	if len(parts) == 0 {
		return nil, fmt.Errorf("empty did:web method-specific identifier")
	}

	host := parts[0]
	// did:web uses percent-encoding for port numbers (e.g., example.com%3A8080)
	host = strings.ReplaceAll(host, "%3A", ":")

	var url string
	if len(parts) == 1 {
		// did:web:example.com → https://example.com/.well-known/did.json
		url = fmt.Sprintf("https://%s/.well-known/did.json", host)
	} else {
		// did:web:example.com:user:alice → https://example.com/user/alice/did.json
		path := strings.Join(parts[1:], "/")
		url = fmt.Sprintf("https://%s/%s/did.json", host, path)
	}

	resp, err := didWebClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("fetching DID document from %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("DID document not found at %s (HTTP %d)", url, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading DID document response: %w", err)
	}

	var doc map[string]interface{}
	if err := json.Unmarshal(body, &doc); err != nil {
		return nil, fmt.Errorf("parsing DID document JSON: %w", err)
	}

	// Validate that the document's id matches the requested DID
	if docID, ok := doc["id"].(string); ok && docID != did {
		return nil, fmt.Errorf("DID document id mismatch: expected %s, got %s", did, docID)
	}

	return doc, nil
}

// ExtractPublicKeyFromDID extracts the Ed25519 public key from a did:key string
func ExtractPublicKeyFromDID(did string) (ed25519.PublicKey, error) {
	if len(did) < 10 || did[:8] != "did:key:" {
		return nil, fmt.Errorf("invalid DID format: %s", did)
	}

	encoded := did[8:] // remove "did:key:"
	if len(encoded) < 2 || encoded[0] != 'z' {
		return nil, fmt.Errorf("invalid did:key encoding")
	}
	encoded = encoded[1:] // remove 'z' prefix

	pubKey, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("decoding public key: %w", err)
	}

	if len(pubKey) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("invalid public key size: %d", len(pubKey))
	}

	return ed25519.PublicKey(pubKey), nil
}
