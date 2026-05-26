package signatures

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Signature represents a cryptographic signature
type Signature struct {
	ID        string    `json:"id"`
	RepoID    string    `json:"repo_id"`
	ObjectID  string    `json:"object_id"` // Commit, tag, or artifact ID
	ObjectType string   `json:"object_type"` // "commit", "tag", "release"
	Signer    string    `json:"signer"` // DID of signer
	Signature string    `json:"signature"` // Base64-encoded signature
	PublicKey string    `json:"public_key"` // Base64-encoded public key
	CreatedAt time.Time `json:"created_at"`
}

// Store manages signatures
type Store struct {
	mu         sync.RWMutex
	baseDir    string
	signatures map[string][]*Signature
}

// NewStore creates a new signature store
func NewStore(baseDir string) *Store {
	return &Store{
		baseDir:    baseDir,
		signatures: make(map[string][]*Signature),
	}
}

// Load loads signatures from disk
func (s *Store) Load() error {
	path := filepath.Join(s.baseDir, "signatures.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return json.Unmarshal(data, &s.signatures)
}

// Save saves signatures to disk
func (s *Store) Save() error {
	path := filepath.Join(s.baseDir, "signatures.json")
	data, err := json.MarshalIndent(s.signatures, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// Sign signs an object
func (s *Store) Sign(sig *Signature, privateKey ed25519.PrivateKey) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Sign the object ID
	sigBytes := ed25519.Sign(privateKey, []byte(sig.ObjectID))
	sig.Signature = base64.StdEncoding.EncodeToString(sigBytes)
	sig.PublicKey = base64.StdEncoding.EncodeToString(privateKey.Public().(ed25519.PublicKey))
	sig.ID = fmt.Sprintf("sig-%d", time.Now().UnixNano())
	sig.CreatedAt = time.Now()

	s.signatures[sig.RepoID] = append(s.signatures[sig.RepoID], sig)
	return s.Save()
}

// Verify verifies a signature
func (s *Store) Verify(repoID, objectID string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, sig := range s.signatures[repoID] {
		if sig.ObjectID == objectID {
			sigBytes, err := base64.StdEncoding.DecodeString(sig.Signature)
			if err != nil {
				return false, err
			}
			pubKeyBytes, err := base64.StdEncoding.DecodeString(sig.PublicKey)
			if err != nil {
				return false, err
			}
			pubKey := ed25519.PublicKey(pubKeyBytes)
			return ed25519.Verify(pubKey, []byte(objectID), sigBytes), nil
		}
	}
	return false, fmt.Errorf("signature not found")
}

// GetSignatures gets signatures for an object
func (s *Store) GetSignatures(repoID, objectID string) []*Signature {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*Signature
	for _, sig := range s.signatures[repoID] {
		if sig.ObjectID == objectID {
			result = append(result, sig)
		}
	}
	return result
}

// ListSignatures lists all signatures for a repository
func (s *Store) ListSignatures(repoID string) []*Signature {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.signatures[repoID]
}
