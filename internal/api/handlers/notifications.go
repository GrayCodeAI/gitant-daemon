package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/lakshmanpatel/gitant/internal/api/middleware"
	"github.com/lakshmanpatel/gitant/internal/notifications"
)

// NotificationHandler handles notification endpoints
type NotificationHandler struct {
	manager *notifications.Manager
}

// NewNotificationHandler creates a new notification handler
func NewNotificationHandler(manager *notifications.Manager) *NotificationHandler {
	return &NotificationHandler{manager: manager}
}

// ListNotifications lists notifications for the current user
func (h *NotificationHandler) ListNotifications(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}

	unreadOnly := r.URL.Query().Get("unread") == "true"
	notifs := h.manager.List(user.ID, unreadOnly)

	result := make([]map[string]interface{}, len(notifs))
	for i, n := range notifs {
		result[i] = map[string]interface{}{
			"id":         n.ID,
			"type":       n.Type,
			"title":      n.Title,
			"body":       n.Body,
			"read":       n.Read,
			"metadata":   n.Metadata,
			"created_at": n.CreatedAt,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"notifications": result,
		"unread_count":  h.manager.UnreadCount(user.ID),
	})
}

// MarkAsRead marks a notification as read
func (h *NotificationHandler) MarkAsRead(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}

	notifID := chi.URLParam(r, "id")
	if err := h.manager.MarkAsRead(user.ID, notifID); err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
	})
}

// MarkAllAsRead marks all notifications as read
func (h *NotificationHandler) MarkAllAsRead(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}

	if err := h.manager.MarkAllAsRead(user.ID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
	})
}

// GetUnreadCount gets the unread notification count
func (h *NotificationHandler) GetUnreadCount(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"count": h.manager.UnreadCount(user.ID),
	})
}
