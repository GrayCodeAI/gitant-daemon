package models

import "time"

// Release represents a repository release.
type Release struct {
	ID         string    `json:"id"`
	RepoID     string    `json:"repo_id"`
	TagName    string    `json:"tag_name"`
	Name       string    `json:"name"`
	Body       string    `json:"body"`
	Author     string    `json:"author"`
	Draft      bool      `json:"draft"`
	Prerelease bool      `json:"prerelease"`
	CreatedAt  time.Time `json:"created_at"`
}
