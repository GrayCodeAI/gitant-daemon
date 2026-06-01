// Package models contains domain entities for the gitant platform.
// These are pure business objects with no infrastructure dependencies.
package models

import "time"

// Repository represents a git repository in the system.
type Repository struct {
    ID          string    `json:"id"`
    Name        string    `json:"name"`
    Description string    `json:"description"`
    Owner       string    `json:"owner"`
    Private     bool      `json:"private"`
    CreatedAt   time.Time `json:"created_at"`
    UpdatedAt   time.Time `json:"updated_at"`
}

// RepositoryFilters defines filtering options for listing repositories.
type RepositoryFilters struct {
    Owner    string
    Private  *bool
    Offset   int
    Limit    int
}
