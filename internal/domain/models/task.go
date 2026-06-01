package models

import "time"

// Task represents a work item in a repository.
type Task struct {
    ID          string    `json:"id"`
    RepoID      string    `json:"repo_id"`
    Title       string    `json:"title"`
    Description string    `json:"description"`
    Author      string    `json:"author"`
    Assignee    string    `json:"assignee"`
    Status      string    `json:"status"` // "open", "in_progress", "completed"
    Priority    int       `json:"priority"`
    CreatedAt   time.Time `json:"created_at"`
    UpdatedAt   time.Time `json:"updated_at"`
}
