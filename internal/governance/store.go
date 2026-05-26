package governance

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Proposal represents a governance proposal
type Proposal struct {
	ID          string    `json:"id"`
	RepoID      string    `json:"repo_id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Type        string    `json:"type"` // "feature", "policy", "budget", "election"
	Status      string    `json:"status"` // "draft", "voting", "passed", "rejected", "executed"
	Proposer    string    `json:"proposer"`
	Votes       Votes     `json:"votes"`
	Quorum      int       `json:"quorum"` // Minimum votes required
	Deadline    time.Time `json:"deadline"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Votes represents votes on a proposal
type Votes struct {
	For     []string `json:"for"`
	Against []string `json:"against"`
	Abstain []string `json:"abstain"`
}

// Store manages proposals
type Store struct {
	mu        sync.RWMutex
	baseDir   string
	proposals map[string][]*Proposal
}

// NewStore creates a new governance store
func NewStore(baseDir string) *Store {
	return &Store{
		baseDir:   baseDir,
		proposals: make(map[string][]*Proposal),
	}
}

// Load loads proposals from disk
func (s *Store) Load() error {
	path := filepath.Join(s.baseDir, "proposals.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return json.Unmarshal(data, &s.proposals)
}

// Save saves proposals to disk
func (s *Store) Save() error {
	path := filepath.Join(s.baseDir, "proposals.json")
	data, err := json.MarshalIndent(s.proposals, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// Create creates a new proposal
func (s *Store) Create(proposal *Proposal) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	proposal.ID = fmt.Sprintf("prop-%d", time.Now().UnixNano())
	proposal.CreatedAt = time.Now()
	proposal.UpdatedAt = time.Now()
	proposal.Status = "draft"
	proposal.Votes = Votes{
		For:     []string{},
		Against: []string{},
		Abstain: []string{},
	}

	s.proposals[proposal.RepoID] = append(s.proposals[proposal.RepoID], proposal)
	return s.Save()
}

// Get gets a proposal by ID
func (s *Store) Get(repoID, proposalID string) (*Proposal, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, p := range s.proposals[repoID] {
		if p.ID == proposalID {
			return p, nil
		}
	}
	return nil, fmt.Errorf("proposal not found")
}

// List lists proposals for a repository
func (s *Store) List(repoID, status string) []*Proposal {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*Proposal
	for _, p := range s.proposals[repoID] {
		if status != "" && p.Status != status {
			continue
		}
		result = append(result, p)
	}
	return result
}

// Submit submits a proposal for voting
func (s *Store) Submit(repoID, proposalID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, p := range s.proposals[repoID] {
		if p.ID == proposalID {
			p.Status = "voting"
			p.UpdatedAt = time.Now()
			return s.Save()
		}
	}
	return fmt.Errorf("proposal not found")
}

// Vote votes on a proposal
func (s *Store) Vote(repoID, proposalID, voter, voteType string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, p := range s.proposals[repoID] {
		if p.ID == proposalID {
			// Remove existing vote
			p.Votes.For = removeString(p.Votes.For, voter)
			p.Votes.Against = removeString(p.Votes.Against, voter)
			p.Votes.Abstain = removeString(p.Votes.Abstain, voter)

			// Add new vote
			switch voteType {
			case "for":
				p.Votes.For = append(p.Votes.For, voter)
			case "against":
				p.Votes.Against = append(p.Votes.Against, voter)
			case "abstain":
				p.Votes.Abstain = append(p.Votes.Abstain, voter)
			default:
				return fmt.Errorf("invalid vote type: %s", voteType)
			}

			p.UpdatedAt = time.Now()
			return s.Save()
		}
	}
	return fmt.Errorf("proposal not found")
}

// Finalize finalizes a proposal
func (s *Store) Finalize(repoID, proposalID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, p := range s.proposals[repoID] {
		if p.ID == proposalID {
			totalVotes := len(p.Votes.For) + len(p.Votes.Against) + len(p.Votes.Abstain)
			if totalVotes < p.Quorum {
				p.Status = "rejected"
			} else if len(p.Votes.For) > len(p.Votes.Against) {
				p.Status = "passed"
			} else {
				p.Status = "rejected"
			}
			p.UpdatedAt = time.Now()
			return s.Save()
		}
	}
	return fmt.Errorf("proposal not found")
}

func removeString(slice []string, s string) []string {
	var result []string
	for _, item := range slice {
		if item != s {
			result = append(result, item)
		}
	}
	return result
}
