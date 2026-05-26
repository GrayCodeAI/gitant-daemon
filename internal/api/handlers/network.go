package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/lakshmanpatel/gitant/internal/network"
)

// NetworkStatus returns libp2p peer information when P2P is enabled.
func NetworkStatus(node *network.Node) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if node == nil {
			writeJSON(w, http.StatusOK, map[string]interface{}{
				"enabled": false,
				"peers":   0,
			})
			return
		}

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"enabled":  true,
			"peer_id":  node.Host.ID().String(),
			"addrs":    node.AdvertisedAddrs(),
			"peers":    node.PeerCount(),
			"connected": node.PeerSummaries(),
		})
	}
}

// DiscoverFederation announces this node and returns federation records.
func DiscoverFederation(node *network.Node) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if node == nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]interface{}{
				"error": "P2P networking is disabled. Start the daemon with --p2p.",
			})
			return
		}

		queryDID := r.URL.Query().Get("did")
		records, err := node.DiscoverFederation(r.Context(), queryDID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]interface{}{
				"error": err.Error(),
			})
			return
		}

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"nodes":  records,
			"events": node.RemoteEvents(25),
		})
	}
}

// BootstrapPeers returns configured federation bootstrap multiaddrs.
func BootstrapPeers() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"peers": network.MergeBootstrapPeers(nil),
		})
	}
}

func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
