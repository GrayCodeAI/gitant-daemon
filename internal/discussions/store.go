package discussions

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Discussion represents a discussion thread
type Discussion struct {
	ID        string    `json:"id"`
	RepoID    string    `json:"repo_id"`
	Title     string    `json:"title"`
	Body      string    `json:"body"`
	Author    string    `json:"author"`
	Category  string    `json:"category"`
	Status    string    `json:"status"`
	Tags      []string  `json:"tags"`
	Answers   []Answer  `json:"answers"`
	Upvotes   int       `json:"upvotes"`
	Views     int       `json:"views"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Answer represents a discussion answer
type Answer struct {
	ID         string    `json:"id"`
	Body       string    `json:"body"`
	Author     string    `json:"author"`
	IsAccepted bool      `json:"is_accepted"`
	Upvotes    int       `json:"upvotes"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// Store manages discussions
type Store struct {
	mu          sync.RWMutex
	baseDir     string
	discussions map[string][]*Discussion
}

// NewStore creates a new discussion store
func NewStore(baseDir string) *Store {
	return &Store{
		baseDir:     baseDir,
		discussions: make(map[string][]*Discussion),
	}
}

// Load loads discussions from disk
func (s *Store) Load() error {
	path := filepath.Join(s.baseDir, "discussions.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return json.Unmarshal(data, &s.discussions)
}

// Save saves discussions to disk
func (s *Store) Save() error {
	path := filepath.Join(s.baseDir, "discussions.json")
	data, err := json.MarshalIndent(s.discussions, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// Create creates a new discussion
func (s *Store) Create(discussion *Discussion) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	discussion.ID = fmt.Sprintf("disc-%d", time.Now().UnixNano())
	discussion.CreatedAt = time.Now()
	discussion.UpdatedAt = time.Now()
	if discussion.Status == "" {
		discussion.Status = "open"
	}
	if discussion.Answers == nil {
		discussion.Answers = []Answer{}
	}
	if discussion.Tags == nil {
		discussion.Tags = []string{}
	}

	s.discussions[discussion.RepoID] = append(s.discussions[discussion.RepoID], discussion)
	return s.Save()
}

// Get gets a discussion by ID
func (s *Store) Get(repoID, discussionID string) (*Discussion, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, d := range s.discussions[repoID] {
		if d.ID == discussionID {
			d.Views++
			return d, nil
		}
	}
	return nil, fmt.Errorf("discussion not found")
}

// List lists discussions for a repository
func (s *Store) List(repoID string, category string, status string) []*Discussion {
	s.mu.RLock()
	defer s.mu.RUnlock()

	discussions := s.discussions[repoID]
	if discussions == nil {
		return []*Discussion{}
	}

	var result []*Discussion
	for _, d := range discussions {
		if category != "" && d.Category != category {
			continue
		}
		if status != "" && d.Status != status {
			continue
		}
		result = append(result, d)
	}
	return result
}

// Update updates a discussion
func (s *Store) Update(repoID, discussionID string, fn func(*Discussion) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, d := range s.discussions[repoID] {
		if d.ID == discussionID {
			if err := fn(d); err != nil {
				return err
			}
			d.UpdatedAt = time.Now()
			return s.Save()
		}
	}
	return fmt.Errorf("discussion not found")
}

// AddAnswer adds an answer to a discussion
func (s *Store) AddAnswer(repoID, discussionID string, answer *Answer) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, d := range s.discussions[repoID] {
		if d.ID == discussionID {
			answer.ID = fmt.Sprintf("ans-%d", time.Now().UnixNano())
			answer.CreatedAt = time.Now()
			answer.UpdatedAt = time.Now()
			d.Answers = append(d.Answers, *answer)
			d.UpdatedAt = time.Now()
			return s.Save()
		}
	}
	return fmt.Errorf("discussion not found")
}

// AcceptAnswer marks an answer as accepted
func (s *Store) AcceptAnswer(repoID, discussionID, answerID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, d := range s.discussions[repoID] {
		if d.ID == discussionID {
			for i, a := range d.Answers {
				if a.ID == answerID {
					d.Answers[i].IsAccepted = true
					d.Status = "answered"
					d.UpdatedAt = time.Now()
					return s.Save()
				}
			}
			return fmt.Errorf("answer not found")
		}
	}
	return fmt.Errorf("discussion not found")
}

// Upvote upvotes a discussion
func (s *Store) Upvote(repoID, discussionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, d := range s.discussions[repoID] {
		if d.ID == discussionID {
			d.Upvotes++
			d.UpdatedAt = time.Now()
			return s.Save()
		}
	}
	return fmt.Errorf("discussion not found")
}

// Delete deletes a discussion
func (s *Store) Delete(repoID, discussionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	discussions := s.discussions[repoID]
	for i, d := range discussions {
		if d.ID == discussionID {
			s.discussions[repoID] = append(discussions[:i], discussions[i+1:]...)
			return s.Save()
		}
	}
	return fmt.Errorf("discussion not found")
}

// Search searches discussions
func (s *Store) Search(repoID, query string) []*Discussion {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var results []*Discussion
	for _, d := range s.discussions[repoID] {
		if contains(d.Title, query) || contains(d.Body, query) {
			results = append(results, d)
		}
	}
	return results
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && len(substr) > 0 && containsIgnoreCase(s, substr))
}

func containsIgnoreCase(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		match := true
		for j := 0; j < len(substr); j++ {
			if toLower(s[i+j]) != toLower(substr[j]) {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

func toLower(b byte) byte {
	if b >= 'A' && b <= 'Z' {
		return b + 32
	}
	return b
}
