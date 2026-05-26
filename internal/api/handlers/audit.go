package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/lakshmanpatel/gitant/internal/audit"
)

// AuditHandler handles audit log endpoints
type AuditHandler struct {
	store *audit.Store
}

// NewAuditHandler creates a new audit handler
func NewAuditHandler(store *audit.Store) *AuditHandler {
	return &AuditHandler{store: store}
}

// ListEvents lists audit events
func (h *AuditHandler) ListEvents(w http.ResponseWriter, r *http.Request) {
	eventType := audit.EventType(r.URL.Query().Get("type"))
	actor := r.URL.Query().Get("actor")
	limitStr := r.URL.Query().Get("limit")

	limit := 100
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	events := h.store.List(eventType, actor, limit)

	result := make([]map[string]interface{}, len(events))
	for i, e := range events {
		result[i] = map[string]interface{}{
			"id":         e.ID,
			"type":       e.Type,
			"actor":      e.Actor,
			"repo_id":    e.RepoID,
			"resource":   e.Resource,
			"action":     e.Action,
			"details":    e.Details,
			"ip":         e.IP,
			"user_agent": e.UserAgent,
			"timestamp":  e.Timestamp,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"events": result,
		"total":  len(result),
	})
}

// ListRepoEvents lists audit events for a repository
func (h *AuditHandler) ListRepoEvents(w http.ResponseWriter, r *http.Request) {
	repoID := chi.URLParam(r, "id")
	limitStr := r.URL.Query().Get("limit")

	limit := 100
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	events := h.store.ListByRepo(repoID, limit)

	result := make([]map[string]interface{}, len(events))
	for i, e := range events {
		result[i] = map[string]interface{}{
			"id":         e.ID,
			"type":       e.Type,
			"actor":      e.Actor,
			"repo_id":    e.RepoID,
			"resource":   e.Resource,
			"action":     e.Action,
			"details":    e.Details,
			"ip":         e.IP,
			"user_agent": e.UserAgent,
			"timestamp":  e.Timestamp,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"events": result,
		"total":  len(result),
	})
}

// GetEvent gets an audit event by ID
func (h *AuditHandler) GetEvent(w http.ResponseWriter, r *http.Request) {
	eventID := chi.URLParam(r, "eventId")

	events := h.store.List("", "", 10000)
	for _, e := range events {
		if e.ID == eventID {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(e)
			return
		}
	}

	http.Error(w, "event not found", http.StatusNotFound)
}
