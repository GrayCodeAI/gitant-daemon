package dto

// Repository represents a repository response DTO.
type Repository struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Private     bool   `json:"private"`
	CreatedAt   string `json:"created_at"`
}

// RepositoryListResponse is the response for listing repositories.
type RepositoryListResponse struct {
	Repos  []Repository `json:"repos"`
	Total  int          `json:"total"`
	Offset int          `json:"offset"`
	Limit  int          `json:"limit"`
}

// CreateRepositoryRequest is the request for creating a repository.
type CreateRepositoryRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Private     bool   `json:"private"`
}
