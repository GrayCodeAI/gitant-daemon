package models

import "time"

// PullRequest represents a git pull request.
type PullRequest struct {
	ID           string    `json:"id"`
	RepoID       string    `json:"repo_id"`
	Title        string    `json:"title"`
	Body         string    `json:"body"`
	Author       string    `json:"author"`
	SourceBranch string    `json:"source_branch"`
	TargetBranch string    `json:"target_branch"`
	Status       string    `json:"status"` // "open", "merged", "closed"
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// PRFilters defines filtering options for listing pull requests.
type PRFilters struct {
	Status string
	Offset int
	Limit  int
}
