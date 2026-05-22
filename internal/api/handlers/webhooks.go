package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

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

		if len(req.Events) == 0 {
			req.Events = []webhooks.EventType{"*"}
		}

		id := fmt.Sprintf("wh-%d", time.Now().UnixNano())
		wh := manager.Register(id, req.URL, req.Events, req.Secret)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(wh)
	}
}

// ListWebhooks lists all registered webhooks
func ListWebhooks(manager *webhooks.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		whs := manager.List()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"webhooks": whs,
			"total":    len(whs),
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
