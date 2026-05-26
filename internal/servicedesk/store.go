package servicedesk

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Ticket represents a service desk ticket
type Ticket struct {
	ID          string    `json:"id"`
	RepoID      string    `json:"repo_id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Status      string    `json:"status"` // "open", "in_progress", "waiting", "resolved", "closed"
	Priority    string    `json:"priority"` // "low", "medium", "high", "critical"
	Category    string    `json:"category"` // "bug", "feature", "support", "question"
	Reporter    string    `json:"reporter"`
	Assignee    string    `json:"assignee"`
	Comments    []Comment `json:"comments"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	ResolvedAt  *time.Time `json:"resolved_at,omitempty"`
}

// Comment represents a ticket comment
type Comment struct {
	ID        string    `json:"id"`
	Author    string    `json:"author"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
}

// Store manages service desk tickets
type Store struct {
	mu      sync.RWMutex
	baseDir string
	tickets map[string][]*Ticket
}

// NewStore creates a new service desk store
func NewStore(baseDir string) *Store {
	return &Store{
		baseDir: baseDir,
		tickets: make(map[string][]*Ticket),
	}
}

// Load loads tickets from disk
func (s *Store) Load() error {
	path := filepath.Join(s.baseDir, "tickets.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return json.Unmarshal(data, &s.tickets)
}

// Save saves tickets to disk
func (s *Store) Save() error {
	path := filepath.Join(s.baseDir, "tickets.json")
	data, err := json.MarshalIndent(s.tickets, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// Create creates a new ticket
func (s *Store) Create(ticket *Ticket) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	ticket.ID = fmt.Sprintf("ticket-%d", time.Now().UnixNano())
	ticket.CreatedAt = time.Now()
	ticket.UpdatedAt = time.Now()
	if ticket.Comments == nil {
		ticket.Comments = []Comment{}
	}

	s.tickets[ticket.RepoID] = append(s.tickets[ticket.RepoID], ticket)
	return s.Save()
}

// Get gets a ticket by ID
func (s *Store) Get(repoID, ticketID string) (*Ticket, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, t := range s.tickets[repoID] {
		if t.ID == ticketID {
			return t, nil
		}
	}
	return nil, fmt.Errorf("ticket not found")
}

// List lists tickets for a repository
func (s *Store) List(repoID, status, priority string) []*Ticket {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*Ticket
	for _, t := range s.tickets[repoID] {
		if status != "" && t.Status != status {
			continue
		}
		if priority != "" && t.Priority != priority {
			continue
		}
		result = append(result, t)
	}
	return result
}

// UpdateStatus updates ticket status
func (s *Store) UpdateStatus(repoID, ticketID, status string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, t := range s.tickets[repoID] {
		if t.ID == ticketID {
			t.Status = status
			t.UpdatedAt = time.Now()
			if status == "resolved" || status == "closed" {
				now := time.Now()
				t.ResolvedAt = &now
			}
			return s.Save()
		}
	}
	return fmt.Errorf("ticket not found")
}

// Assign assigns a ticket
func (s *Store) Assign(repoID, ticketID, assignee string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, t := range s.tickets[repoID] {
		if t.ID == ticketID {
			t.Assignee = assignee
			t.UpdatedAt = time.Now()
			return s.Save()
		}
	}
	return fmt.Errorf("ticket not found")
}

// AddComment adds a comment to a ticket
func (s *Store) AddComment(repoID, ticketID, author, body string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, t := range s.tickets[repoID] {
		if t.ID == ticketID {
			comment := Comment{
				ID:        fmt.Sprintf("comment-%d", time.Now().UnixNano()),
				Author:    author,
				Body:      body,
				CreatedAt: time.Now(),
			}
			t.Comments = append(t.Comments, comment)
			t.UpdatedAt = time.Now()
			return s.Save()
		}
	}
	return fmt.Errorf("ticket not found")
}
