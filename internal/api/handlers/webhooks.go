package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/lakshmanpatel/gitant/internal/webhooks"
)

// RegisterWebhook registers a new webhook
func RegisterWebhook(manager *webhooks.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			URL    string             `json:"url"`
			Events []webhooks.EventType `json:"events"`
			Secret string             `json:"secret,omitempty"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if req.URL == "" {
			http.Error(w, "URL is required", http.StatusBadRequest)
			return
		}

		if err := webhooks.ValidateWebhookURL(req.URL); err != nil {
			http.Error(w, fmt.Sprintf("Invalid webhook URL: %s", err), http.StatusBadRequest)
			return
		}

		if len(req.Events) == 0 {
			req.Events = []webhooks.EventType{"*"}
		}

		id := generateID("wh")
		wh := manager.Register(id, req.URL, req.Events, req.Secret)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(wh)
	}
}

// ListWebhooks lists all registered webhooks (secrets masked)
func ListWebhooks(manager *webhooks.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		offset, limit := ParsePagination(r)
		whs := manager.List()

		// Mask secrets before sending to client
		masked := make([]map[string]interface{}, len(whs))
		for i, wh := range whs {
			entry := map[string]interface{}{
				"id":     wh.ID,
				"url":    wh.URL,
				"events": wh.Events,
			}
			if wh.Secret != "" {
				entry["secret"] = "***"
			}
			masked[i] = entry
		}

		paged, total := PaginateSlice(masked, offset, limit)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"webhooks": paged,
			"total":    total,
			"offset":   offset,
			"limit":    limit,
		})
	}
}

// DeleteWebhook deletes a webhook
func DeleteWebhook(manager *webhooks.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		manager.Remove(id)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]bool{"deleted": true})
	}
}
