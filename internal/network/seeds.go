package network

import (
	"os"
	"strings"
)

// DefaultBootstrapPeers returns well-known seed nodes for WAN federation.
// Override with --bootstrap-peers or GITANT_BOOTSTRAP_PEERS.
func DefaultBootstrapPeers() []string {
	if seeds := os.Getenv("GITANT_SEED_PEERS"); seeds != "" {
		return splitPeers(seeds)
	}
	return nil
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
