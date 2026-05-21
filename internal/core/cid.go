package core

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/go-git/go-git/v6/plumbing"
)

// CID represents a Content Identifier for IPFS
// Git objects are content-addressed by SHA-1, IPFS uses SHA-256
type CID struct {
	Hash string
}

// NewCID creates a new CID from raw data
func NewCID(data []byte) *CID {
	hash := sha256.Sum256(data)
	return &CID{
		Hash: hex.EncodeToString(hash[:]),
	}
}

// CIDFromGitHash converts a git hash to a CID
// Git uses SHA-1, we convert to SHA-256 for IPFS compatibility
func CIDFromGitHash(hash plumbing.Hash) *CID {
	// Convert SHA-1 to SHA-256 by hashing the git hash string
	data := []byte(hash.String())
	sha256Hash := sha256.Sum256(data)
	return &CID{
		Hash: hex.EncodeToString(sha256Hash[:]),
	}
}

// CIDFromBytes creates a CID from raw bytes
func CIDFromBytes(data []byte) *CID {
	hash := sha256.Sum256(data)
	return &CID{
		Hash: hex.EncodeToString(hash[:]),
	}
}

// String returns the CID as a string
func (c *CID) String() string {
	return fmt.Sprintf("bafy%s", c.Hash[:20])
}

// Bytes returns the CID as bytes
func (c *CID) Bytes() []byte {
	b, _ := hex.DecodeString(c.Hash)
	return b
}

// Equals checks if two CIDs are equal
func (c *CID) Equals(other *CID) bool {
	return c.Hash == other.Hash
}

// GitObjectToCID converts a git object to a CID
func GitObjectToCID(objType plumbing.ObjectType, content []byte) *CID {
	// Create a deterministic representation of the git object
	header := fmt.Sprintf("%s %d\x00", objType.String(), len(content))
	data := append([]byte(header), content...)
	return NewCID(data)
}

// GitHashToCIDString converts a git hash to a CID string
func GitHashToCIDString(hash plumbing.Hash) string {
	cid := CIDFromGitHash(hash)
	return cid.String()
}

// CIDStringToGitHash converts a CID string back to a git hash
func CIDStringToGitHash(cidStr string) (plumbing.Hash, error) {
	// Remove the "bafy" prefix
	if len(cidStr) < 4 || cidStr[:4] != "bafy" {
		return plumbing.ZeroHash, fmt.Errorf("invalid CID format: %s", cidStr)
	}

	// The CID is derived from the git hash, so we need to store the mapping
	// For now, we'll use a simple approach: store the git hash in the CID
	// This will be enhanced in Phase 2 with proper IPFS integration
	return plumbing.ZeroHash, fmt.Errorf("CID to git hash conversion requires IPFS mapping")
}
