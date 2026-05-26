package events

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// EventType represents the type of event
type EventType string

const (
	EventPush           EventType = "push"
	EventIssueCreated   EventType = "issue.created"
	EventIssueClosed    EventType = "issue.closed"
	EventIssueCommented EventType = "issue.commented"
	EventPROpened       EventType = "pr.opened"
	EventPRMerged       EventType = "pr.merged"
	EventPRReviewed     EventType = "pr.reviewed"
	EventPRCommented    EventType = "pr.commented"
	EventReleaseCreated EventType = "release.created"
	EventStarAdded      EventType = "star.added"
	EventStarRemoved    EventType = "star.removed"
	EventForkCreated    EventType = "fork.created"
	EventWebhookFired   EventType = "webhook.fired"
)

// Event represents an event
type Event struct {
	ID        string                 `json:"id"`
	Type      EventType              `json:"type"`
	RepoID    string                 `json:"repo_id"`
	Actor     string                 `json:"actor"`
	Timestamp time.Time              `json:"timestamp"`
	Data      map[string]interface{} `json:"data"`
}

// Store manages events
type Store struct {
	mu     sync.RWMutex
	events []Event
	maxSize int
}

// NewStore creates a new event store
func NewStore(maxSize int) *Store {
	return &Store{
		events:  []Event{},
		maxSize: maxSize,
	}
}

// Append appends an event
func (s *Store) Append(eventType EventType, repoID, actor string, data map[string]interface{}) *Event {
	s.mu.Lock()
	defer s.mu.Unlock()

	event := &Event{
		ID:        fmt.Sprintf("evt-%d", time.Now().UnixNano()),
		Type:      eventType,
		RepoID:    repoID,
		Actor:     actor,
		Timestamp: time.Now(),
		Data:      data,
	}

	s.events = append(s.events, *event)

	// Trim to max size
	if len(s.events) > s.maxSize {
		s.events = s.events[len(s.events)-s.maxSize:]
	}

	return event
}

// List lists events
func (s *Store) List(repoID string, eventType EventType, limit int) []Event {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if limit <= 0 {
		limit = 100
	}

	var result []Event
	for i := len(s.events) - 1; i >= 0; i-- {
		event := s.events[i]
		if repoID != "" && event.RepoID != repoID {
			continue
		}
		if eventType != "" && event.Type != eventType {
			continue
		}
		result = append(result, event)
		if len(result) >= limit {
			break
		}
	}

	return result
}

// Get gets an event by ID
func (s *Store) Get(id string) (*Event, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, event := range s.events {
		if event.ID == id {
			return &event, nil
		}
	}
	return nil, fmt.Errorf("event not found")
}

// Count returns the number of events
func (s *Store) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.events)
}

// MarshalJSON marshals an event to JSON
func (e *Event) MarshalJSON() ([]byte, error) {
	type Alias Event
	return json.Marshal(&struct {
		*Alias
		Timestamp string `json:"timestamp"`
	}{
		Alias:     (*Alias)(e),
		Timestamp: e.Timestamp.Format(time.RFC3339),
	})
}
