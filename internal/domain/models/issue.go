package models

import "time"

// Issue represents a repository issue.
type Issue struct {
    ID          string    `json:"id"`
    RepoID      string    `json:"repo_id"`
    Title       string    `json:"title"`
    Body        string    `json:"body"`
    Author      string    `json:"author"`
    Status      string    `json:"status"` // "open" or "closed"
    Labels      []string  `json:"labels"`
    CreatedAt   time.Time `json:"created_at"`
    UpdatedAt   time.Time `json:"updated_at"`
}

// IssueFilters defines filtering options for listing issues.
type IssueFilters struct {
    Status string
    Labels []string
    Offset int
    Limit  int
}
