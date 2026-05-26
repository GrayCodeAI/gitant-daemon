package network

import (
	"encoding/json"
	"log/slog"
	"time"
)

const attestationTopic = "gitant/attestations"

// AttestationMessage propagates agent trust scores across the federation mesh.
type AttestationMessage struct {
	SourceDID  string    `json:"source_did"`
	TargetDID  string    `json:"target_did"`
	Score      float64   `json:"score"`
	Reason     string    `json:"reason,omitempty"`
	Timestamp  time.Time `json:"timestamp"`
	SourcePeer string    `json:"source_peer,omitempty"`
}

// TrustStore applies remote agent trust attestations.
type TrustStore interface {
	ApplyAttestation(sourceDID, targetDID string, score float64) error
}

// PublishAttestation gossips an agent trust attestation to peers.
func (c *SyncCoordinator) PublishAttestation(targetDID string, score float64, reason string) error {
	if c == nil || c.node == nil || c.trust == nil {
		return nil
	}

	msg := AttestationMessage{
		SourceDID:  c.node.cfg.ServerDID,
		TargetDID:  targetDID,
		Score:      score,
		Reason:     reason,
		Timestamp:  time.Now().UTC(),
		SourcePeer: c.node.Host.ID().String(),
	}
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return c.node.Gossip.Publish(attestationTopic, data)
}

func (c *SyncCoordinator) startAttestationSubscriber() error {
	if c.node == nil || c.trust == nil {
		return nil
	}

	sub, err := c.node.Gossip.Subscribe(attestationTopic)
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

			var att AttestationMessage
			if err := json.Unmarshal(msg.Data, &att); err != nil {
				continue
			}
			if att.SourcePeer == c.node.Host.ID().String() {
				continue
			}
			if err := c.trust.ApplyAttestation(att.SourceDID, att.TargetDID, att.Score); err != nil {
				slog.Warn("failed to apply remote attestation", "target", att.TargetDID, "error", err)
			}
		}
	}()
	return nil
}
