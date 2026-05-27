package network

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"time"

	"github.com/ipfs/go-cid"
	"github.com/multiformats/go-multihash"
	p2pnetwork "github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
)

const BlockProtocol = protocol.ID("/gitant/block/1.0.0")
const BatchBlockProtocol = protocol.ID("/gitant/blocks/1.0.0")

// ObjectStore reads and writes git objects for P2P replication.
type ObjectStore interface {
	HasObject(repoID, hash string) bool
	GetObject(repoID, hash string) (objType string, data []byte, err error)
	PutObject(repoID, hash, objType string, data []byte) error
}

type blockRequest struct {
	Repo string `json:"repo"`
	Hash string `json:"hash"`
}

type blockResponse struct {
	OK      bool   `json:"ok"`
	Error   string `json:"error,omitempty"`
	Type    string `json:"type,omitempty"`
	Content string `json:"content,omitempty"`
}

func registerBlockHandler(node *Node, store ObjectStore) {
	if node == nil || node.Host == nil || store == nil {
		return
	}

	registerBatchBlockHandler(node, store)

	node.Host.SetStreamHandler(BlockProtocol, func(stream p2pnetwork.Stream) {
		defer stream.Close()

		var req blockRequest
		decoder := json.NewDecoder(stream)
		if err := decoder.Decode(&req); err != nil {
			writeBlockResponse(stream, blockResponse{OK: false, Error: "invalid request"})
			return
		}

		objType, data, err := store.GetObject(req.Repo, req.Hash)
		if err != nil {
			writeBlockResponse(stream, blockResponse{OK: false, Error: err.Error()})
			return
		}

		writeBlockResponse(stream, blockResponse{
			OK:      true,
			Type:    objType,
			Content: base64.StdEncoding.EncodeToString(data),
		})
	})
}

func writeBlockResponse(w io.Writer, resp blockResponse) {
	_ = json.NewEncoder(w).Encode(resp)
}

type batchBlockRequest struct {
	Repo   string   `json:"repo"`
	Hashes []string `json:"hashes"`
}

type batchObject struct {
	Hash    string `json:"hash"`
	OK      bool   `json:"ok"`
	Error   string `json:"error,omitempty"`
	Type    string `json:"type,omitempty"`
	Content string `json:"content,omitempty"`
}

type batchBlockResponse struct {
	Objects []batchObject `json:"objects"`
}

func registerBatchBlockHandler(node *Node, store ObjectStore) {
	if node == nil || node.Host == nil || store == nil {
		return
	}

	node.Host.SetStreamHandler(BatchBlockProtocol, func(stream p2pnetwork.Stream) {
		defer stream.Close()

		var req batchBlockRequest
		if err := json.NewDecoder(stream).Decode(&req); err != nil {
			_ = json.NewEncoder(stream).Encode(batchBlockResponse{})
			return
		}

		resp := batchBlockResponse{
			Objects: make([]batchObject, 0, len(req.Hashes)),
		}
		for _, hash := range req.Hashes {
			objType, data, err := store.GetObject(req.Repo, hash)
			if err != nil {
				resp.Objects = append(resp.Objects, batchObject{Hash: hash, OK: false, Error: err.Error()})
			} else {
				resp.Objects = append(resp.Objects, batchObject{
					Hash:    hash,
					OK:      true,
					Type:    objType,
					Content: base64.StdEncoding.EncodeToString(data),
				})
			}
		}
		_ = json.NewEncoder(stream).Encode(resp)
	})
}

// FetchObject retrieves a git object from a connected peer.
func (n *Node) FetchObject(ctx context.Context, peerID peer.ID, repoID, hash string) (objType string, data []byte, err error) {
	if n == nil || n.Host == nil {
		return "", nil, fmt.Errorf("P2P not enabled")
	}

	reqCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	stream, err := n.Host.NewStream(reqCtx, peerID, BlockProtocol)
	if err != nil {
		return "", nil, fmt.Errorf("opening block stream: %w", err)
	}
	defer stream.Close()

	if err := json.NewEncoder(stream).Encode(blockRequest{Repo: repoID, Hash: hash}); err != nil {
		return "", nil, fmt.Errorf("sending block request: %w", err)
	}

	var resp blockResponse
	if err := json.NewDecoder(stream).Decode(&resp); err != nil {
		return "", nil, fmt.Errorf("reading block response: %w", err)
	}
	if !resp.OK {
		return "", nil, fmt.Errorf("peer block fetch failed: %s", resp.Error)
	}

	data, err = base64.StdEncoding.DecodeString(resp.Content)
	if err != nil {
		return "", nil, fmt.Errorf("decoding block content: %w", err)
	}
	return resp.Type, data, nil
}

// FetchResult holds the result of fetching a single object.
type FetchResult struct {
	Hash string
	Type string
	Data []byte
	Err  error
}

// FetchObjects retrieves multiple git objects from a peer in a single batch request.
// Falls back to sequential single-object fetch if the batch protocol is unsupported.
func (n *Node) FetchObjects(ctx context.Context, peerID peer.ID, repoID string, hashes []string) []FetchResult {
	if n == nil || n.Host == nil || len(hashes) == 0 {
		return nil
	}

	reqCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	stream, err := n.Host.NewStream(reqCtx, peerID, BatchBlockProtocol)
	if err != nil {
		// Peer doesn't support batch protocol — fall back to single-object fetch.
		return n.fetchObjectsFallback(ctx, peerID, repoID, hashes)
	}
	defer stream.Close()

	if err := json.NewEncoder(stream).Encode(batchBlockRequest{Repo: repoID, Hashes: hashes}); err != nil {
		return n.fetchObjectsFallback(ctx, peerID, repoID, hashes)
	}

	var resp batchBlockResponse
	if err := json.NewDecoder(stream).Decode(&resp); err != nil {
		return n.fetchObjectsFallback(ctx, peerID, repoID, hashes)
	}

	results := make([]FetchResult, 0, len(resp.Objects))
	for _, obj := range resp.Objects {
		r := FetchResult{Hash: obj.Hash}
		if !obj.OK {
			r.Err = fmt.Errorf("%s", obj.Error)
		} else {
			r.Type = obj.Type
			r.Data, r.Err = base64.StdEncoding.DecodeString(obj.Content)
		}
		results = append(results, r)
	}
	return results
}

func (n *Node) fetchObjectsFallback(ctx context.Context, peerID peer.ID, repoID string, hashes []string) []FetchResult {
	results := make([]FetchResult, 0, len(hashes))
	for _, hash := range hashes {
		objType, data, err := n.FetchObject(ctx, peerID, repoID, hash)
		results = append(results, FetchResult{Hash: hash, Type: objType, Data: data, Err: err})
	}
	return results
}

// CIDFromGitHash creates a CID from a repo ID and git object hash for DHT content routing.
func CIDFromGitHash(repoID, hash string) (cid.Cid, error) {
	// Combine repo+hash into a deterministic content key
	key := fmt.Sprintf("gitant:%s:%s", repoID, hash)
	mh, err := multihash.Sum([]byte(key), multihash.SHA2_256, -1)
	if err != nil {
		return cid.Undef, err
	}
	return cid.NewCidV1(cid.Raw, mh), nil
}

// AnnounceObject announces a git object in the DHT using CID-based content routing.
func (n *Node) AnnounceObject(ctx context.Context, repoID, hash string) {
	if n == nil || n.DHT == nil {
		return
	}
	c, err := CIDFromGitHash(repoID, hash)
	if err != nil {
		slog.Debug("CID creation failed", "repo", repoID, "hash", hash, "error", err)
		return
	}
	if err := n.DHT.Provide(ctx, c); err != nil {
		slog.Debug("DHT provide failed", "repo", repoID, "hash", hash, "error", err)
	}
}

// FindObjectProviders finds peers that have a specific git object.
func (n *Node) FindObjectProviders(ctx context.Context, repoID, hash string) ([]peer.AddrInfo, error) {
	if n == nil || n.DHT == nil {
		return nil, fmt.Errorf("P2P not enabled")
	}
	c, err := CIDFromGitHash(repoID, hash)
	if err != nil {
		return nil, fmt.Errorf("CID creation: %w", err)
	}
	return n.DHT.FindProviders(ctx, c)
}

// SyncObjects pulls missing objects from a peer and stores them locally.
func (n *Node) SyncObjects(ctx context.Context, store ObjectStore, peerID peer.ID, repoID string, hashes []string) error {
	if n == nil || store == nil {
		return nil
	}

	// Filter to only hashes we don't already have.
	needed := make([]string, 0, len(hashes))
	for _, hash := range hashes {
		if hash != "" && !store.HasObject(repoID, hash) {
			needed = append(needed, hash)
		}
	}
	if len(needed) == 0 {
		return nil
	}

	results := n.FetchObjects(ctx, peerID, repoID, needed)
	var firstErr error
	for _, r := range results {
		if r.Err != nil {
			slog.Warn("failed to fetch object from peer", "repo", repoID, "hash", r.Hash, "peer", peerID, "error", r.Err)
			if firstErr == nil {
				firstErr = r.Err
			}
			continue
		}
		objType := r.Type
		if objType == "" {
			objType = "blob"
		}
		if err := store.PutObject(repoID, r.Hash, objType, r.Data); err != nil {
			slog.Warn("failed to store replicated object", "repo", repoID, "hash", r.Hash, "error", err)
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		n.AnnounceObject(ctx, repoID, r.Hash)
		slog.Info("replicated git object", "repo", repoID, "hash", r.Hash, "from", peerID.String())
	}
	return firstErr
}
