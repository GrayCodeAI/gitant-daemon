package network

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"time"

	p2pnetwork "github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
)

const BlockProtocol = protocol.ID("/gitant/block/1.0.0")

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

// AnnounceObject stores a DHT pointer for a git object hash.
func (n *Node) AnnounceObject(ctx context.Context, repoID, hash string) {
	if n == nil || n.DHT == nil {
		return
	}
	key := fmt.Sprintf("/gitant/repo/%s/object/%s", repoID, hash)
	value := []byte(n.Host.ID().String())
	if err := n.DHT.PutValue(ctx, key, value); err != nil {
		slog.Debug("DHT object announce failed", "repo", repoID, "hash", hash, "error", err)
	}
}

// SyncObjects pulls missing objects from a peer and stores them locally.
func (n *Node) SyncObjects(ctx context.Context, store ObjectStore, peerID peer.ID, repoID string, hashes []string) error {
	if n == nil || store == nil {
		return nil
	}

	var firstErr error
	for _, hash := range hashes {
		if hash == "" || store.HasObject(repoID, hash) {
			continue
		}

		objType, data, err := n.FetchObject(ctx, peerID, repoID, hash)
		if err != nil {
			slog.Warn("failed to fetch object from peer", "repo", repoID, "hash", hash, "peer", peerID, "error", err)
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		if objType == "" {
			objType = "blob"
		}
		if err := store.PutObject(repoID, hash, objType, data); err != nil {
			slog.Warn("failed to store replicated object", "repo", repoID, "hash", hash, "error", err)
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		n.AnnounceObject(ctx, repoID, hash)
		slog.Info("replicated git object", "repo", repoID, "hash", hash, "from", peerID.String())
	}
	return firstErr
}
