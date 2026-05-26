package bounties

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// BountyStatus represents the status of a bounty
type BountyStatus string

const (
	StatusOpen     BountyStatus = "open"
	StatusClaimed  BountyStatus = "claimed"
	StatusSubmitted BountyStatus = "submitted"
	StatusApproved BountyStatus = "approved"
	StatusCanceled BountyStatus = "canceled"
	StatusDisputed BountyStatus = "disputed"
)

// Bounty represents a bounty
type Bounty struct {
	ID          string       `json:"id"`
	RepoID      string       `json:"repo_id"`
	Title       string       `json:"title"`
	Description string       `json:"description"`
	Amount      float64      `json:"amount"`
	Currency    string       `json:"currency"`
	Status      BountyStatus `json:"status"`
	Creator     string       `json:"creator"`
	ClaimedBy   string       `json:"claimed_by,omitempty"`
	SubmittedBy string       `json:"submitted_by,omitempty"`
	ApprovedBy  string       `json:"approved_by,omitempty"`
	Submission  string       `json:"submission,omitempty"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
	Deadline    *time.Time   `json:"deadline,omitempty"`
	Tags        []string     `json:"tags"`
}

// Store manages bounties
type Store struct {
	mu       sync.RWMutex
	baseDir  string
	bounties map[string][]*Bounty // repoID -> bounties
}

// NewStore creates a new bounty store
func NewStore(baseDir string) *Store {
	return &Store{
		baseDir:  baseDir,
		bounties: make(map[string][]*Bounty),
	}
}

// Load loads bounties from disk
func (s *Store) Load() error {
	path := filepath.Join(s.baseDir, "bounties.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return json.Unmarshal(data, &s.bounties)
}

// Save saves bounties to disk
func (s *Store) Save() error {
	path := filepath.Join(s.baseDir, "bounties.json")
	data, err := json.MarshalIndent(s.bounties, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// Create creates a new bounty
func (s *Store) Create(bounty *Bounty) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	bounty.ID = fmt.Sprintf("bounty-%d", time.Now().UnixNano())
	bounty.CreatedAt = time.Now()
	bounty.UpdatedAt = time.Now()
	if bounty.Status == "" {
		bounty.Status = StatusOpen
	}
	if bounty.Tags == nil {
		bounty.Tags = []string{}
	}

	s.bounties[bounty.RepoID] = append(s.bounties[bounty.RepoID], bounty)
	return s.Save()
}

// Get gets a bounty by ID
func (s *Store) Get(bountyID string) (*Bounty, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, bounties := range s.bounties {
		for _, b := range bounties {
			if b.ID == bountyID {
				return b, nil
			}
		}
	}
	return nil, fmt.Errorf("bounty not found")
}

// ListRepo lists bounties for a repository
func (s *Store) ListRepo(repoID string, status BountyStatus) []*Bounty {
	s.mu.RLock()
	defer s.mu.RUnlock()

	bounties := s.bounties[repoID]
	if bounties == nil {
		return []*Bounty{}
	}

	var result []*Bounty
	for _, b := range bounties {
		if status != "" && b.Status != status {
			continue
		}
		result = append(result, b)
	}
	return result
}

// ListAll lists all bounties
func (s *Store) ListAll(status BountyStatus) []*Bounty {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*Bounty
	for _, bounties := range s.bounties {
		for _, b := range bounties {
			if status != "" && b.Status != status {
				continue
			}
			result = append(result, b)
		}
	}
	return result
}

// Claim claims a bounty
func (s *Store) Claim(bountyID, agentDID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, bounties := range s.bounties {
		for _, b := range bounties {
			if b.ID == bountyID {
				if b.Status != StatusOpen {
					return fmt.Errorf("bounty is not open")
				}
				b.Status = StatusClaimed
				b.ClaimedBy = agentDID
				b.UpdatedAt = time.Now()
				return s.Save()
			}
		}
	}
	return fmt.Errorf("bounty not found")
}

// Submit submits work for a bounty
func (s *Store) Submit(bountyID, agentDID, submission string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, bounties := range s.bounties {
		for _, b := range bounties {
			if b.ID == bountyID {
				if b.Status != StatusClaimed || b.ClaimedBy != agentDID {
					return fmt.Errorf("bounty not claimed by this agent")
				}
				b.Status = StatusSubmitted
				b.SubmittedBy = agentDID
				b.Submission = submission
				b.UpdatedAt = time.Now()
				return s.Save()
			}
		}
	}
	return fmt.Errorf("bounty not found")
}

// Approve approves a bounty submission
func (s *Store) Approve(bountyID, approverDID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, bounties := range s.bounties {
		for _, b := range bounties {
			if b.ID == bountyID {
				if b.Status != StatusSubmitted {
					return fmt.Errorf("bounty not submitted")
				}
				b.Status = StatusApproved
				b.ApprovedBy = approverDID
				b.UpdatedAt = time.Now()
				return s.Save()
			}
		}
	}
	return fmt.Errorf("bounty not found")
}

// Cancel cancels a bounty
func (s *Store) Cancel(bountyID, creatorDID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, bounties := range s.bounties {
		for _, b := range bounties {
			if b.ID == bountyID {
				if b.Creator != creatorDID {
					return fmt.Errorf("only creator can cancel")
				}
				b.Status = StatusCanceled
				b.UpdatedAt = time.Now()
				return s.Save()
			}
		}
	}
	return fmt.Errorf("bounty not found")
}

// Dispute disputes a bounty
func (s *Store) Dispute(bountyID, agentDID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, bounties := range s.bounties {
		for _, b := range bounties {
			if b.ID == bountyID {
				b.Status = StatusDisputed
				b.UpdatedAt = time.Now()
				return s.Save()
			}
		}
	}
	return fmt.Errorf("bounty not found")
}
