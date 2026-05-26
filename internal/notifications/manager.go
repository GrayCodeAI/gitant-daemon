package notifications

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// NotificationType represents the type of notification
type NotificationType string

const (
	TypeIssueCreated    NotificationType = "issue.created"
	TypeIssueCommented  NotificationType = "issue.commented"
	TypeIssueClosed     NotificationType = "issue.closed"
	TypePROpened        NotificationType = "pr.opened"
	TypePRMerged        NotificationType = "pr.merged"
	TypePRReviewed      NotificationType = "pr.reviewed"
	TypeReviewComment   NotificationType = "review.comment"
	TypeReleasePublished NotificationType = "release.published"
	TypeMention         NotificationType = "mention"
)

// Notification represents a notification
type Notification struct {
	ID        string           `json:"id"`
	UserID    string           `json:"user_id"`
	Type      NotificationType `json:"type"`
	Title     string           `json:"title"`
	Body      string           `json:"body"`
	Read      bool             `json:"read"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt time.Time        `json:"created_at"`
}

// Manager manages notifications
type Manager struct {
	mu            sync.RWMutex
	baseDir       string
	notifications map[string][]*Notification // userID -> notifications
}

// NewManager creates a new notification manager
func NewManager(baseDir string) *Manager {
	return &Manager{
		baseDir:       baseDir,
		notifications: make(map[string][]*Notification),
	}
}

// Load loads notifications from disk
func (m *Manager) Load() error {
	path := filepath.Join(m.baseDir, "notifications.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return json.Unmarshal(data, &m.notifications)
}

// Save saves notifications to disk
func (m *Manager) Save() error {
	path := filepath.Join(m.baseDir, "notifications.json")
	data, err := json.MarshalIndent(m.notifications, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// Create creates a notification for a user
func (m *Manager) Create(userID string, notifType NotificationType, title, body string, metadata map[string]interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	notif := &Notification{
		ID:        fmt.Sprintf("notif-%d", time.Now().UnixNano()),
		UserID:    userID,
		Type:      notifType,
		Title:     title,
		Body:      body,
		Read:      false,
		Metadata:  metadata,
		CreatedAt: time.Now(),
	}

	m.notifications[userID] = append(m.notifications[userID], notif)
	return m.Save()
}

// List lists notifications for a user
func (m *Manager) List(userID string, unreadOnly bool) []*Notification {
	m.mu.RLock()
	defer m.mu.RUnlock()

	notifs := m.notifications[userID]
	if notifs == nil {
		return []*Notification{}
	}

	if !unreadOnly {
		return notifs
	}

	var filtered []*Notification
	for _, n := range notifs {
		if !n.Read {
			filtered = append(filtered, n)
		}
	}
	return filtered
}

// MarkAsRead marks a notification as read
func (m *Manager) MarkAsRead(userID, notifID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, n := range m.notifications[userID] {
		if n.ID == notifID {
			n.Read = true
			return m.Save()
		}
	}

	return fmt.Errorf("notification not found")
}

// MarkAllAsRead marks all notifications as read for a user
func (m *Manager) MarkAllAsRead(userID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, n := range m.notifications[userID] {
		n.Read = true
	}

	return m.Save()
}

// UnreadCount returns the unread notification count for a user
func (m *Manager) UnreadCount(userID string) int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	count := 0
	for _, n := range m.notifications[userID] {
		if !n.Read {
			count++
		}
	}
	return count
}
