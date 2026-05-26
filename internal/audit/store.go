package audit

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// EventType represents the type of audit event
type EventType string

const (
	EventUserCreated     EventType = "user.created"
	EventUserUpdated     EventType = "user.updated"
	EventUserDeleted     EventType = "user.deleted"
	EventRepoCreated     EventType = "repo.created"
	EventRepoDeleted     EventType = "repo.deleted"
	EventRepoUpdated     EventType = "repo.updated"
	EventIssueCreated    EventType = "issue.created"
	EventIssueClosed     EventType = "issue.closed"
	EventIssueCommented  EventType = "issue.commented"
	EventPROpened        EventType = "pr.opened"
	EventPRMerged        EventType = "pr.merged"
	EventPRReviewed      EventType = "pr.reviewed"
	EventPush            EventType = "push"
	EventReleaseCreated  EventType = "release.created"
	EventWebhookCreated  EventType = "webhook.created"
	EventWebhookDeleted  EventType = "webhook.deleted"
	EventSettingsChanged EventType = "settings.changed"
)

// Event represents an audit event
type Event struct {
	ID        string                 `json:"id"`
	Type      EventType              `json:"type"`
	Actor     string                 `json:"actor"`
	RepoID    string                 `json:"repo_id,omitempty"`
	Resource  string                 `json:"resource"`
	Action    string                 `json:"action"`
	Details   map[string]interface{} `json:"details,omitempty"`
	IP        string                 `json:"ip,omitempty"`
	UserAgent string                 `json:"user_agent,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}

// Store manages audit events
type Store struct {
	mu     sync.RWMutex
	path   string
	events []Event
}

// NewStore creates a new audit store
func NewStore(baseDir string) *Store {
	return &Store{
		path:   filepath.Join(baseDir, "audit.json"),
		events: []Event{},
	}
}

// Load loads audit events from disk
func (s *Store) Load() error {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return json.Unmarshal(data, &s.events)
}

// Save saves audit events to disk
func (s *Store) Save() error {
	data, err := json.MarshalIndent(s.events, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0644)
}

// Record records an audit event
func (s *Store) Record(eventType EventType, actor, resource, action string, details map[string]interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	event := Event{
		ID:        fmt.Sprintf("audit-%d", time.Now().UnixNano()),
		Type:      eventType,
		Actor:     actor,
		Resource:  resource,
		Action:    action,
		Details:   details,
		Timestamp: time.Now(),
	}

	s.events = append(s.events, event)

	// Keep only last 10000 events
	if len(s.events) > 10000 {
		s.events = s.events[len(s.events)-10000:]
	}

	return s.Save()
}

// RecordWithRequest records an audit event with request info
func (s *Store) RecordWithRequest(eventType EventType, actor, resource, action string, details map[string]interface{}, ip, userAgent string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	event := Event{
		ID:        fmt.Sprintf("audit-%d", time.Now().UnixNano()),
		Type:      eventType,
		Actor:     actor,
		Resource:  resource,
		Action:    action,
		Details:   details,
		IP:        ip,
		UserAgent: userAgent,
		Timestamp: time.Now(),
	}

	s.events = append(s.events, event)

	if len(s.events) > 10000 {
		s.events = s.events[len(s.events)-10000:]
	}

	return s.Save()
}

// List lists audit events
func (s *Store) List(eventType EventType, actor string, limit int) []Event {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if limit <= 0 {
		limit = 100
	}

	var result []Event
	for i := len(s.events) - 1; i >= 0; i-- {
		event := s.events[i]
		if eventType != "" && event.Type != eventType {
			continue
		}
		if actor != "" && event.Actor != actor {
			continue
		}
		result = append(result, event)
		if len(result) >= limit {
			break
		}
	}

	if result == nil {
		result = []Event{}
	}

	return result
}

// ListByRepo lists audit events for a repository
func (s *Store) ListByRepo(repoID string, limit int) []Event {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if limit <= 0 {
		limit = 100
	}

	var result []Event
	for i := len(s.events) - 1; i >= 0; i-- {
		if s.events[i].RepoID == repoID {
			result = append(result, s.events[i])
			if len(result) >= limit {
				break
			}
		}
	}

	if result == nil {
		result = []Event{}
	}

	return result
}

// Count returns the total number of events
func (s *Store) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.events)
}
