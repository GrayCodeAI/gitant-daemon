package models

import "time"

// Label represents a label that can be applied to issues/PRs.
type Label struct {
	ID        string    `json:"id"`
	RepoID    string    `json:"repo_id"`
	Name      string    `json:"name"`
	Color     string    `json:"color"`
	CreatedAt time.Time `json:"created_at"`
}
