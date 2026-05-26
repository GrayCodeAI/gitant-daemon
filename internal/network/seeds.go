package network

import (
	_ "embed"
	"encoding/json"
	"log/slog"
	"os"
	"strings"
)

//go:embed bootstrap_peers.json
var embeddedBootstrapPeers []byte

// DefaultBootstrapPeers returns well-known Gitant seed nodes for WAN federation.
// Override with GITANT_SEED_PEERS, --bootstrap-peers, or bootstrap_peers.json at deploy time.
func DefaultBootstrapPeers() []string {
	if seeds := os.Getenv("GITANT_SEED_PEERS"); seeds != "" {
		return splitPeers(seeds)
	}
	return loadEmbeddedBootstrapPeers()
}

func loadEmbeddedBootstrapPeers() []string {
	var peers []string
	if err := json.Unmarshal(embeddedBootstrapPeers, &peers); err != nil {
		slog.Warn("failed to parse embedded bootstrap peers", "error", err)
		return nil
	}
	return peers
}

func splitPeers(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

// MergeBootstrapPeers combines built-in and user-provided bootstrap peers.
func MergeBootstrapPeers(user []string) []string {
	merged := append(DefaultBootstrapPeers(), user...)
	seen := make(map[string]struct{}, len(merged))
	out := make([]string, 0, len(merged))
	for _, peer := range merged {
		if peer == "" {
			continue
		}
		if _, ok := seen[peer]; ok {
			continue
		}
		seen[peer] = struct{}{}
		out = append(out, peer)
	}
	return out
}
