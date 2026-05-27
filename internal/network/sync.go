package network

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/lakshmanpatel/gitant/internal/crdt"
)

const crdtGlobalTopic = "gitant/crdt"

// CRDTStore applies remote CRDT snapshots from peers.
type CRDTStore interface {
	MergeIssue(repoID string, issue *crdt.Issue) error
	MergePR(repoID string, pr *crdt.PullRequest) error
}

// CRDTMessage replicates issue/PR operation logs across peers.
type CRDTMessage struct {
	Repo       string          `json:"repo"`
	Entity     string          `json:"entity"`
	EntityID   string          `json:"entity_id"`
	Payload    json.RawMessage `json:"payload"`
	SourceDID  string          `json:"source_did,omitempty"`
	SourcePeer string          `json:"source_peer,omitempty"`
}

// RepoCRDTTopic returns the repo-scoped CRDT gossip topic.
func RepoCRDTTopic(repoID string) string {
	return fmt.Sprintf("gitant/repo/%s/crdt", repoID)
}

// SyncCoordinator wires block exchange and CRDT replication into a node.
type SyncCoordinator struct {
	node    *Node
	objects ObjectStore
	crdt    CRDTStore
	trust   TrustStore
	pinner  ObjectPinner
}

// ObjectPinner optionally pins replicated git objects (IPFS warm storage adapter).
type ObjectPinner interface {
	PinGitObject(ctx context.Context, repoID, hash string, data []byte) (string, error)
	IsPinned(repoID, hash string) bool
}

// NewSyncCoordinator registers replication handlers on a node.
func NewSyncCoordinator(node *Node, objects ObjectStore, crdtStore CRDTStore, trustStore TrustStore, pinner ObjectPinner) *SyncCoordinator {
	coord := &SyncCoordinator{
		node:    node,
		objects: objects,
		crdt:    crdtStore,
		trust:   trustStore,
		pinner:  pinner,
	}
	if node == nil {
		return coord
	}

	registerBlockHandler(node, objects)
	if err := coord.startCRDTSubscriber(); err != nil {
		slog.Warn("CRDT gossip subscription failed", "error", err)
	}
	if err := coord.startAttestationSubscriber(); err != nil {
		slog.Warn("attestation gossip subscription failed", "error", err)
	}

	node.SetFederatedEventHandler(coord.handleFederatedEvent)
	return coord
}

// PublishIssue broadcasts an issue snapshot to peers.
func (c *SyncCoordinator) PublishIssue(repoID string, issue *crdt.Issue) error {
	if c == nil || c.node == nil || issue == nil {
		return nil
	}
	payload, err := json.Marshal(issue)
	if err != nil {
		return err
	}
	return c.publishCRDT(CRDTMessage{
		Repo:     repoID,
		Entity:   "issue",
		EntityID: issue.ID,
		Payload:  payload,
	})
}

// PublishPR broadcasts a pull request snapshot to peers.
func (c *SyncCoordinator) PublishPR(repoID string, pr *crdt.PullRequest) error {
	if c == nil || c.node == nil || pr == nil {
		return nil
	}
	payload, err := json.Marshal(pr)
	if err != nil {
		return err
	}
	return c.publishCRDT(CRDTMessage{
		Repo:     repoID,
		Entity:   "pr",
		EntityID: pr.ID,
		Payload:  payload,
	})
}

// AnnouncePushObjects announces git object hashes in the DHT and optional pin store.
func (c *SyncCoordinator) AnnouncePushObjects(ctx context.Context, repoID string, hashes []string) {
	if c == nil || c.node == nil {
		return
	}
	for _, hash := range hashes {
		c.node.AnnounceObject(ctx, repoID, hash)
		c.pinLocalObject(ctx, repoID, hash)
	}
}

func (c *SyncCoordinator) pinLocalObject(ctx context.Context, repoID, hash string) {
	if c == nil || c.pinner == nil || c.objects == nil {
		return
	}
	if c.pinner.IsPinned(repoID, hash) {
		return
	}
	_, data, err := c.objects.GetObject(repoID, hash)
	if err != nil {
		return
	}
	if _, err := c.pinner.PinGitObject(ctx, repoID, hash, data); err != nil {
		slog.Debug("IPFS pin failed", "repo", repoID, "hash", hash, "error", err)
	}
}

func (c *SyncCoordinator) publishCRDT(msg CRDTMessage) error {
	msg.SourceDID = c.node.cfg.ServerDID
	msg.SourcePeer = c.node.Host.ID().String()

	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	if err := c.node.Gossip.Publish(crdtGlobalTopic, data); err != nil {
		slog.Warn("CRDT global publish failed", "error", err)
	}
	return c.node.Gossip.Publish(RepoCRDTTopic(msg.Repo), data)
}

func (c *SyncCoordinator) startCRDTSubscriber() error {
	sub, err := c.node.Gossip.Subscribe(crdtGlobalTopic)
	if err != nil {
		return err
	}

	go func() {
		for {
			msg, err := sub.Next(c.node.ctx)
			if err != nil {
				if c.node.ctx.Err() != nil {
					return
				}
				continue
			}
			if msg.From == c.node.Host.ID() {
				continue
			}

			var crdtMsg CRDTMessage
			if err := json.Unmarshal(msg.Data, &crdtMsg); err != nil {
				continue
			}
			if crdtMsg.SourcePeer == c.node.Host.ID().String() {
				continue
			}
			c.applyCRDT(crdtMsg)
		}
	}()
	return nil
}

func (c *SyncCoordinator) applyCRDT(msg CRDTMessage) {
	if c.crdt == nil {
		return
	}

	switch msg.Entity {
	case "issue":
		var issue crdt.Issue
		if err := json.Unmarshal(msg.Payload, &issue); err != nil {
			slog.Warn("invalid remote issue payload", "error", err)
			return
		}
		if err := c.crdt.MergeIssue(msg.Repo, &issue); err != nil {
			slog.Warn("failed to merge remote issue", "repo", msg.Repo, "issue", msg.EntityID, "error", err)
			return
		}
		slog.Info("merged remote issue", "repo", msg.Repo, "issue", msg.EntityID, "from", msg.SourcePeer)
	case "pr":
		var pr crdt.PullRequest
		if err := json.Unmarshal(msg.Payload, &pr); err != nil {
			slog.Warn("invalid remote PR payload", "error", err)
			return
		}
		if err := c.crdt.MergePR(msg.Repo, &pr); err != nil {
			slog.Warn("failed to merge remote PR", "repo", msg.Repo, "pr", msg.EntityID, "error", err)
			return
		}
		slog.Info("merged remote PR", "repo", msg.Repo, "pr", msg.EntityID, "from", msg.SourcePeer)
	}
}

func (c *SyncCoordinator) handleFederatedEvent(event FederatedEvent) {
	if c == nil || c.node == nil || c.objects == nil {
		return
	}
	if event.Type != "push" || event.SourcePeer == "" || event.SourcePeer == c.node.Host.ID().String() {
		return
	}

	hashes := ParseObjectHashes(event.Data)
	if len(hashes) == 0 {
		return
	}

	peerID, err := peer.Decode(event.SourcePeer)
	if err != nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := c.node.SyncObjects(ctx, c.objects, peerID, event.Repo, hashes); err != nil {
		slog.Debug("sync objects failed", "peer", peerID, "repo", event.Repo, "error", err)
	}
	for _, hash := range hashes {
		c.pinLocalObject(ctx, event.Repo, hash)
	}
}

// ParseObjectHashes extracts git object hashes from webhook event data.
func ParseObjectHashes(data map[string]interface{}) []string {
	return extractStringSlice(data, "object_hashes")
}

// ParseRefHeads extracts ref -> head hash mappings from webhook event data.
func ParseRefHeads(data map[string]interface{}) map[string]string {
	raw, ok := data["ref_heads"]
	if !ok {
		return nil
	}
	heads, ok := raw.(map[string]interface{})
	if !ok {
		return nil
	}
	out := make(map[string]string, len(heads))
	for ref, value := range heads {
		if hash, ok := value.(string); ok && hash != "" {
			out[ref] = hash
		}
	}
	return out
}

// PushEventData builds webhook/sync metadata for a push.
func PushEventData(objectHashes []string, refHeads map[string]string) map[string]interface{} {
	data := map[string]interface{}{
		"object_hashes": objectHashes,
		"objects":       len(objectHashes),
	}
	if len(refHeads) > 0 {
		data["ref_heads"] = refHeads
		refNames := make([]string, 0, len(refHeads))
		for ref := range refHeads {
			refNames = append(refNames, ref)
		}
		data["refs"] = refNames
		if head, ok := refHeads[refNames[0]]; ok {
			data["ref"] = head
		}
	}
	return data
}

// NormalizeObjectHash trims and validates a git object hash string.
func NormalizeObjectHash(hash string) string {
	hash = strings.TrimSpace(hash)
	if len(hash) != 40 {
		return ""
	}
	return hash
}

func extractStringSlice(data map[string]interface{}, key string) []string {
	raw, ok := data[key]
	if !ok {
		return nil
	}
	switch values := raw.(type) {
	case []string:
		return values
	case []interface{}:
		out := make([]string, 0, len(values))
		for _, value := range values {
			if s, ok := value.(string); ok && s != "" {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
}
