// Package webhooks provides event notification for gitant.
package webhooks

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"path/filepath"
	"sync"
	"time"

	"github.com/lakshmanpatel/gitant/internal/persistence"
)

// EventType represents the type of event
type EventType string

const (
	EventRepoCreated    EventType = "repo.created"
	EventRepoDeleted    EventType = "repo.deleted"
	EventIssueCreated   EventType = "issue.created"
	EventIssueClosed    EventType = "issue.closed"
	EventIssueCommented EventType = "issue.commented"
	EventPROpened       EventType = "pr.opened"
	EventPRMerged       EventType = "pr.merged"
	EventPRReviewed     EventType = "pr.reviewed"
	EventPush           EventType = "push"
)

// Event represents a webhook event
type Event struct {
	Type      EventType              `json:"type"`
	Repo      string                 `json:"repo"`
	Timestamp time.Time              `json:"timestamp"`
	Data      map[string]interface{} `json:"data"`
}

// Webhook represents a registered webhook
type Webhook struct {
	ID     string      `json:"id"`
	URL    string      `json:"url"`
	Events []EventType `json:"events"`
	Secret string      `json:"secret,omitempty"`
}

// Manager manages webhooks and dispatches events
type Manager struct {
	mu        sync.RWMutex
	webhooks  map[string]*Webhook
	client    *http.Client
	dataDir   string
	eventHook func(Event)
}

// NewManager creates a new webhook manager
func NewManager() *Manager {
	return &Manager{
		webhooks: make(map[string]*Webhook),
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// ValidateWebhookURL checks that a webhook URL is safe (not targeting internal/private IPs).
func ValidateWebhookURL(rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("URL scheme must be http or https")
	}
	if u.Hostname() == "" {
		return fmt.Errorf("URL must have a hostname")
	}

	// Resolve hostname to IP and check for private/internal ranges
	hostname := u.Hostname()
	ips, err := net.LookupIP(hostname)
	if err != nil {
		return nil // DNS failure means the URL won't work anyway; let it fail at dispatch
	}
	for _, ip := range ips {
		if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
			return fmt.Errorf("webhook URL resolves to a private/internal IP address (%s -> %s)", hostname, ip)
		}
		// Block AWS metadata endpoint range
		if ip.To4() != nil && ip.To4()[0] == 169 && ip.To4()[1] == 254 {
			return fmt.Errorf("webhook URL resolves to link-local metadata range (%s -> %s)", hostname, ip)
		}
	}
	return nil
}

// Register registers a new webhook and persists it.
func (m *Manager) Register(id, url string, events []EventType, secret string) *Webhook {
	m.mu.Lock()
	wh := &Webhook{
		ID:     id,
		URL:    url,
		Events: events,
		Secret: secret,
	}
	m.webhooks[id] = wh
	m.mu.Unlock()
	m.Save()
	return wh
}

// Remove removes a webhook and persists the change.
func (m *Manager) Remove(id string) {
	m.mu.Lock()
	delete(m.webhooks, id)
	m.mu.Unlock()
	m.Save()
}

// List returns all webhooks
func (m *Manager) List() []*Webhook {
	m.mu.RLock()
	defer m.mu.RUnlock()
	whs := make([]*Webhook, 0, len(m.webhooks))
	for _, wh := range m.webhooks {
		whs = append(whs, wh)
	}
	return whs
}

// SetEventHook registers a callback invoked for every dispatched event.
func (m *Manager) SetEventHook(fn func(Event)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.eventHook = fn
}

// Dispatch sends an event to all matching webhooks
func (m *Manager) Dispatch(event Event) {
	m.mu.RLock()
	hook := m.eventHook
	m.mu.RUnlock()

	event.Timestamp = time.Now()

	if hook != nil {
		hook(event)
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, wh := range m.webhooks {
		if m.matches(wh, event.Type) {
			go m.send(wh, event)
		}
	}
}

// matches checks if a webhook subscribes to an event type
func (m *Manager) matches(wh *Webhook, eventType EventType) bool {
	for _, e := range wh.Events {
		if e == eventType || e == "*" {
			return true
		}
	}
	return false
}

// send sends an event to a webhook endpoint
func (m *Manager) send(wh *Webhook, event Event) {
	body, err := json.Marshal(event)
	if err != nil {
		slog.Error("webhook marshal error", "error", err)
		return
	}

	req, err := http.NewRequest("POST", wh.URL, bytes.NewReader(body))
	if err != nil {
		slog.Error("webhook request error", "error", err)
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Gitant-Event", string(event.Type))
	req.Header.Set("X-Gitant-Delivery", fmt.Sprintf("%d", time.Now().UnixNano()))
	if wh.Secret != "" {
		sig := HMACSHA256(body, wh.Secret)
		req.Header.Set("X-Gitant-Signature-256", "sha256="+hex.EncodeToString(sig))
	}

	resp, err := m.client.Do(req)
	if err != nil {
		slog.Error("webhook delivery error", "url", wh.URL, "error", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		slog.Warn("webhook returned error status", "url", wh.URL, "status", resp.StatusCode)
	}
}

// HMACSHA256 computes the HMAC-SHA256 of msg using the given secret key.
func HMACSHA256(msg []byte, secret string) []byte {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(msg)
	return mac.Sum(nil)
}

// Save persists all webhooks to a JSON file in the data directory.
func (m *Manager) Save() error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.dataDir == "" {
		return nil
	}
	return persistence.SaveJSON(filepath.Join(m.dataDir, "webhooks.json"), m.webhooks)
}

// Load loads webhooks from a JSON file in dataDir.
// If the file does not exist, it is a no-op.
func (m *Manager) Load(dataDir string) error {
	m.dataDir = dataDir
	m.mu.Lock()
	defer m.mu.Unlock()
	loaded := make(map[string]*Webhook)
	if err := persistence.LoadJSON(filepath.Join(dataDir, "webhooks.json"), &loaded); err != nil {
		return err
	}
	m.webhooks = loaded
	return nil
}
