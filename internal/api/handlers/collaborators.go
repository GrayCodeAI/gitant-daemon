package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/lakshmanpatel/gitant/internal/store"
)

type collaboratorResponse struct {
	RepoID    string `json:"repo_id"`
	UserID    string `json:"user_id"`
	Role      string `json:"role"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type CollaboratorUserResolver func(ctx context.Context, userID, username string) (*store.User, error)

func newCollaboratorResponse(collaborator *store.RepoCollaborator) collaboratorResponse {
	return collaboratorResponse{
		RepoID:    collaborator.RepoID,
		UserID:    collaborator.UserID,
		Role:      collaborator.Role,
		CreatedAt: collaborator.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt: collaborator.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}
}

// ListCollaborators lists repository owners and collaborators.
func ListCollaborators(collaborators store.RepoCollaboratorStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if collaborators == nil {
			http.Error(w, "Repository collaborator store unavailable", http.StatusInternalServerError)
			return
		}

		repoID := chi.URLParam(r, "id")
		members, err := collaborators.ListByRepo(r.Context(), repoID)
		if err != nil {
			http.Error(w, SanitizeError(err, "failed to list collaborators"), http.StatusInternalServerError)
			return
		}

		result := make([]collaboratorResponse, 0, len(members))
		for _, member := range members {
			result = append(result, newCollaboratorResponse(member))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"collaborators": result,
			"total":         len(result),
		})
	}
}

// AddCollaborator adds or updates a repository collaborator membership.
func AddCollaborator(collaborators store.RepoCollaboratorStore, resolvers ...CollaboratorUserResolver) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if collaborators == nil {
			http.Error(w, "Repository collaborator store unavailable", http.StatusInternalServerError)
			return
		}

		var req struct {
			UserID   string `json:"user_id"`
			Username string `json:"username"`
			User     string `json:"user"`
			Role     string `json:"role"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		userID := strings.TrimSpace(req.UserID)
		username := strings.TrimSpace(req.Username)
		if userID == "" && username == "" {
			userID = strings.TrimSpace(req.User)
		}
		if len(resolvers) > 0 && resolvers[0] != nil {
			resolved, err := resolvers[0](r.Context(), userID, username)
			if err != nil {
				http.Error(w, SanitizeError(err, "collaborator user not found"), http.StatusNotFound)
				return
			}
			userID = resolved.ID
		}
		if userID == "" {
			http.Error(w, "user_id is required", http.StatusBadRequest)
			return
		}

		role := strings.TrimSpace(req.Role)
		if role == "" {
			role = store.RepoRoleCollaborator
		}
		if role != store.RepoRoleCollaborator {
			http.Error(w, "role must be collaborator", http.StatusBadRequest)
			return
		}

		collaborator := &store.RepoCollaborator{
			RepoID: chi.URLParam(r, "id"),
			UserID: userID,
			Role:   role,
		}
		if err := collaborators.Add(r.Context(), collaborator); err != nil {
			http.Error(w, SanitizeError(err, "failed to add collaborator"), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(newCollaboratorResponse(collaborator))
	}
}

// RemoveCollaborator removes a repository collaborator membership.
func RemoveCollaborator(collaborators store.RepoCollaboratorStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if collaborators == nil {
			http.Error(w, "Repository collaborator store unavailable", http.StatusInternalServerError)
			return
		}

		repoID := chi.URLParam(r, "id")
		userID := strings.TrimSpace(chi.URLParam(r, "user"))
		if userID == "" {
			http.Error(w, "user is required", http.StatusBadRequest)
			return
		}

		member, err := collaborators.Get(r.Context(), repoID, userID)
		if err == nil && member.Role == store.RepoRoleOwner {
			http.Error(w, "repository owner membership cannot be removed through collaborators API", http.StatusBadRequest)
			return
		}

		if err := collaborators.Remove(r.Context(), repoID, userID); err != nil {
			status := http.StatusInternalServerError
			if strings.Contains(err.Error(), "not found") {
				status = http.StatusNotFound
			}
			http.Error(w, SanitizeError(err, "collaborator not found"), status)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"removed": true,
			"repo_id": repoID,
			"user_id": userID,
		})
	}
}
