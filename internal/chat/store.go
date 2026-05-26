package chat

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Message represents a chat message
type Message struct {
	ID        string    `json:"id"`
	RepoID    string    `json:"repo_id"`
	Channel   string    `json:"channel"` // "general", "dev", "support"
	Author    string    `json:"author"`
	Body      string    `json:"body"`
	ReplyTo   string    `json:"reply_to,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// Channel represents a chat channel
type Channel struct {
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}

// Store manages chat messages
type Store struct {
	mu       sync.RWMutex
	baseDir  string
	messages map[string]map[string][]*Message // repoID -> channel -> messages
	channels map[string][]Channel             // repoID -> channels
}

// NewStore creates a new chat store
func NewStore(baseDir string) *Store {
	return &Store{
		baseDir:  baseDir,
		messages: make(map[string]map[string][]*Message),
		channels: make(map[string][]Channel),
	}
}

// Load loads chat data from disk
func (s *Store) Load() error {
	path := filepath.Join(s.baseDir, "chat.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return json.Unmarshal(data, &s.messages)
}

// Save saves chat data to disk
func (s *Store) Save() error {
	path := filepath.Join(s.baseDir, "chat.json")
	data, err := json.MarshalIndent(s.messages, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// SendMessage sends a message
func (s *Store) SendMessage(msg *Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	msg.ID = fmt.Sprintf("msg-%d", time.Now().UnixNano())
	msg.CreatedAt = time.Now()

	if s.messages[msg.RepoID] == nil {
		s.messages[msg.RepoID] = make(map[string][]*Message)
	}

	channel := msg.Channel
	if channel == "" {
		channel = "general"
	}

	s.messages[msg.RepoID][channel] = append(s.messages[msg.RepoID][channel], msg)
	return s.Save()
}

// GetMessages gets messages for a channel
func (s *Store) GetMessages(repoID, channel string, limit int) []*Message {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.messages[repoID] == nil {
		return []*Message{}
	}

	messages := s.messages[repoID][channel]
	if limit > 0 && len(messages) > limit {
		messages = messages[len(messages)-limit:]
	}
	return messages
}

// GetChannels gets channels for a repository
func (s *Store) GetChannels(repoID string) []Channel {
	s.mu.RLock()
	defer s.mu.RUnlock()

	channels := s.channels[repoID]
	if channels == nil {
		// Return default channels
		return []Channel{
			{Name: "general", Description: "General discussion"},
			{Name: "dev", Description: "Development discussion"},
			{Name: "support", Description: "Support and help"},
		}
	}
	return channels
}

// CreateChannel creates a new channel
func (s *Store) CreateChannel(repoID, name, description string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	channel := Channel{
		Name:        name,
		Description: description,
		CreatedAt:   time.Now(),
	}

	s.channels[repoID] = append(s.channels[repoID], channel)
	return s.Save()
}
