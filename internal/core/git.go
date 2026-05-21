package core

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/go-git/go-git/v6/plumbing"
)

// GitObject represents a git object with its type and content
type GitObject struct {
	Type    plumbing.ObjectType
	Content []byte
	Hash    plumbing.Hash
}

// GitToCID converts a git hash to a CID
// Git hashes are SHA-1, we convert to SHA-256 for IPFS compatibility
func GitToCID(hash plumbing.Hash) *CID {
	// Git hash is already content-addressed
	// For IPFS, we'll use the git hash directly as the CID
	return &CID{
		Hash: hash.String(),
	}
}

// CIDToGit converts a CID back to a git hash
func CIDToGit(cid *CID) (plumbing.Hash, error) {
	if len(cid.Hash) != 40 {
		return plumbing.ZeroHash, fmt.Errorf("invalid git hash length: %d", len(cid.Hash))
	}
	return plumbing.NewHash(cid.Hash), nil
}

// ComputeGitHash computes the git hash for a blob
func ComputeGitHash(content []byte) plumbing.Hash {
	header := fmt.Sprintf("blob %d\x00", len(content))
	data := append([]byte(header), content...)
	hash := sha256.Sum256(data)
	return plumbing.NewHash(hex.EncodeToString(hash[:]))
}
