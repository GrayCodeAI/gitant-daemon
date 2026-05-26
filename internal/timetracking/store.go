package timetracking

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// TimeEntry represents a time tracking entry
type TimeEntry struct {
	ID          string    `json:"id"`
	RepoID      string    `json:"repo_id"`
	IssueID     string    `json:"issue_id,omitempty"`
	PRID        string    `json:"pr_id,omitempty"`
	TaskID      string    `json:"task_id,omitempty"`
	User        string    `json:"user"`
	Description string    `json:"description"`
	Duration    int       `json:"duration"` // Duration in seconds
	StartedAt   time.Time `json:"started_at"`
	EndedAt     time.Time `json:"ended_at"`
	Billable    bool      `json:"billable"`
}

// Store manages time entries
type Store struct {
	mu     sync.RWMutex
	baseDir string
	entries map[string][]*TimeEntry
}

// NewStore creates a new time tracking store
func NewStore(baseDir string) *Store {
	return &Store{
		baseDir: baseDir,
		entries: make(map[string][]*TimeEntry),
	}
}

// Load loads time entries from disk
func (s *Store) Load() error {
	path := filepath.Join(s.baseDir, "time-entries.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return json.Unmarshal(data, &s.entries)
}

// Save saves time entries to disk
func (s *Store) Save() error {
	path := filepath.Join(s.baseDir, "time-entries.json")
	data, err := json.MarshalIndent(s.entries, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// Start starts a timer
func (s *Store) Start(entry *TimeEntry) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry.ID = fmt.Sprintf("time-%d", time.Now().UnixNano())
	entry.StartedAt = time.Now()

	s.entries[entry.RepoID] = append(s.entries[entry.RepoID], entry)
	return s.Save()
}

// Stop stops a timer
func (s *Store) Stop(repoID, entryID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, e := range s.entries[repoID] {
		if e.ID == entryID {
			e.EndedAt = time.Now()
			e.Duration = int(e.EndedAt.Sub(e.StartedAt).Seconds())
			return s.Save()
		}
	}
	return fmt.Errorf("entry not found")
}

// List lists time entries for a repository
func (s *Store) List(repoID, user string, startDate, endDate time.Time) []*TimeEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*TimeEntry
	for _, e := range s.entries[repoID] {
		if user != "" && e.User != user {
			continue
		}
		if !startDate.IsZero() && e.StartedAt.Before(startDate) {
			continue
		}
		if !endDate.IsZero() && e.StartedAt.After(endDate) {
			continue
		}
		result = append(result, e)
	}
	return result
}

// GetSummary gets time tracking summary
func (s *Store) GetSummary(repoID, user string, startDate, endDate time.Time) map[string]interface{} {
	entries := s.List(repoID, user, startDate, endDate)

	totalSeconds := 0
	billableSeconds := 0
	for _, e := range entries {
		totalSeconds += e.Duration
		if e.Billable {
			billableSeconds += e.Duration
		}
	}

	return map[string]interface{}{
		"total_seconds":    totalSeconds,
		"billable_seconds": billableSeconds,
		"entries_count":    len(entries),
		"total_hours":      float64(totalSeconds) / 3600,
		"billable_hours":   float64(billableSeconds) / 3600,
	}
}
