package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	authMiddleware "github.com/lakshmanpatel/gitant/internal/api/middleware"
	"github.com/lakshmanpatel/gitant/internal/storage"
)

// CanAccessRepo reports whether the caller may read a repository.
func CanAccessRepo(r *http.Request, entry *storage.RepoEntry, serverDID string) bool {
	return CanAccessRepoWithCollaborators(r, entry, serverDID, nil)
}

// CanAccessRepoWithCollaborators reports whether the caller may read a repository,
// including session users with server-side owner/collaborator membership.
func CanAccessRepoWithCollaborators(r *http.Request, entry *storage.RepoEntry, serverDID string, collaborators authMiddleware.RepoWriteAuthorizer) bool {
	if entry == nil || !entry.Private {
		return true
	}

	identity := authMiddleware.GetIdentity(r)
	if identity == "" {
		return false
	}
	if serverDID != "" && identity == serverDID {
		return true
	}

	ucan := authMiddleware.GetUCAN(r)
	if ucan != nil {
		resource := "repo:" + entry.ID
		return ucan.HasCapability(resource, "read") ||
			ucan.HasCapability(resource, "*") ||
			ucan.HasCapability("*", "read") ||
			ucan.HasCapability("*", "*")
	}

	user := authMiddleware.GetUser(r)
	if user == nil || collaborators == nil {
		return false
	}
	allowed, err := collaborators.IsWriter(r.Context(), entry.ID, user.ID)
	return err == nil && allowed
}

// RequireRepoReadAccess blocks unauthenticated access to private repositories.
func RequireRepoReadAccess(registry *storage.RepositoryRegistry, serverDID string) func(http.Handler) http.Handler {
	return RequireRepoReadAccessWithCollaborators(registry, serverDID, nil)
}

// RequireRepoReadAccessWithCollaborators blocks unauthenticated access to private
// repositories while allowing session owners/collaborators to read their repos.
func RequireRepoReadAccessWithCollaborators(registry *storage.RepositoryRegistry, serverDID string, collaborators authMiddleware.RepoWriteAuthorizer) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			repoID := chi.URLParam(r, "id")
			if repoID == "" {
				next.ServeHTTP(w, r)
				return
			}

			entry, err := registry.GetEntry(repoID)
			if err != nil {
				http.Error(w, "Repository not found", http.StatusNotFound)
				return
			}
			if !CanAccessRepoWithCollaborators(r, entry, serverDID, collaborators) {
				http.Error(w, "Repository not found", http.StatusNotFound)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
