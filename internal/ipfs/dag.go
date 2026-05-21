package ipfs

import (
	"context"
	"fmt"

	"github.com/go-git/go-git/v6/plumbing"
	"github.com/ipfs/go-cid"
	"github.com/multiformats/go-multihash"
)

// GitDAG provides git object storage in IPFS
type GitDAG struct {
	node *Node
}

// NewGitDAG creates a new GitDAG
func NewGitDAG(node *Node) *GitDAG {
	return &GitDAG{
		node: node,
	}
}

// PutGitObject stores a git object in IPFS
func (g *GitDAG) PutGitObject(ctx context.Context, objType plumbing.ObjectType, content []byte) (cid.Cid, error) {
	return g.node.PutBlock(ctx, content)
}

// GetGitObject retrieves a git object from IPFS
func (g *GitDAG) GetGitObject(ctx context.Context, c cid.Cid) ([]byte, error) {
	return g.node.GetBlock(ctx, c)
}

// GitHashToCID converts a git hash to an IPFS CID
func GitHashToCID(hash plumbing.Hash) (cid.Cid, error) {
	// Create multihash from git hash
	mh, err := multihash.Sum([]byte(hash.String()), multihash.SHA2_256, -1)
	if err != nil {
		return cid.Undef, fmt.Errorf("creating multihash: %w", err)
	}

	// Create CID
	return cid.NewCidV1(cid.Raw, mh), nil
}

// CIDToGitHash converts an IPFS CID to a git hash
func CIDToGitHash(c cid.Cid) (plumbing.Hash, error) {
	// Get multihash
	mh := c.Hash()

	// Decode multihash
	dm, err := multihash.Decode(mh)
	if err != nil {
		return plumbing.ZeroHash, fmt.Errorf("decoding multihash: %w", err)
	}

	// Create git hash
	return plumbing.NewHash(string(dm.Digest)), nil
}
