package handlers

import (
	"encoding/base64"
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-git/go-git/v6/plumbing"
	"github.com/lakshmanpatel/gitant/internal/storage"
	"github.com/lakshmanpatel/gitant/internal/webhooks"
)

// CreateRepo creates a new repository
func CreateRepo(registry *storage.RepositoryRegistry, wm *webhooks.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Name        string `json:"name"`
			Description string `json:"description"`
			Private     bool   `json:"private"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if req.Name == "" {
			http.Error(w, "Name is required", http.StatusBadRequest)
			return
		}

		if !ValidateID(req.Name) {
			http.Error(w, "Invalid repository name: must be alphanumeric with hyphens/dots/underscores, max 64 chars", http.StatusBadRequest)
			return
		}
		if len(req.Description) > 10000 {
			http.Error(w, "Description too long (max 10000 characters)", http.StatusBadRequest)
			return
		}

		entry, err := registry.Create(req.Name, req.Name, req.Description, req.Private)
		if err != nil {
			http.Error(w, SanitizeError(err, "failed to create repository"), http.StatusConflict)
			return
		}

		wm.Dispatch(webhooks.Event{
			Type: webhooks.EventRepoCreated,
			Repo: entry.Name,
			Data: map[string]interface{}{
				"id":          entry.ID,
				"description": entry.Description,
				"private":     entry.Private,
			},
		})

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":          entry.ID,
			"name":        entry.Name,
			"description": entry.Description,
			"private":     entry.Private,
			"created_at":  entry.CreatedAt,
		})
	}
}

// ListRepos lists all repositories visible to the caller.
func ListRepos(registry *storage.RepositoryRegistry, serverDID string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		offset, limit := ParsePagination(r)
		repos := registry.List()

		result := make([]map[string]interface{}, 0, len(repos))
		for _, entry := range repos {
			if !CanAccessRepo(r, entry, serverDID) {
				continue
			}
			result = append(result, map[string]interface{}{
				"id":          entry.ID,
				"name":        entry.Name,
				"description": entry.Description,
				"private":     entry.Private,
				"created_at":  entry.CreatedAt,
			})
		}

		paged, total := PaginateSlice(result, offset, limit)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"repos":  paged,
			"total":  total,
			"offset": offset,
			"limit":  limit,
		})
	}
}

// GetRepo gets a repository by ID
func GetRepo(registry *storage.RepositoryRegistry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")

		entry, err := registry.GetEntry(id)
		if err != nil {
			http.Error(w, SanitizeError(err, "repository not found"), http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":          entry.ID,
			"name":        entry.Name,
			"description": entry.Description,
			"private":     entry.Private,
			"created_at":  entry.CreatedAt,
		})
	}
}

// DeleteRepo deletes a repository
func DeleteRepo(registry *storage.RepositoryRegistry, wm *webhooks.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")

		if err := registry.Delete(id); err != nil {
			if strings.Contains(err.Error(), "not found") {
				http.Error(w, SanitizeError(err, "repository not found"), http.StatusNotFound)
			} else {
				http.Error(w, SanitizeError(err, "failed to delete repository"), http.StatusInternalServerError)
			}
			return
		}

		wm.Dispatch(webhooks.Event{
			Type: webhooks.EventRepoDeleted,
			Repo: id,
			Data: map[string]interface{}{
				"id": id,
			},
		})

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"deleted": true,
			"id":      id,
		})
	}
}

// PushObjects pushes objects to a repository
func PushObjects(registry *storage.RepositoryRegistry, protectionStore *storage.ProtectionStore, wm *webhooks.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")

		repo, err := registry.Open(id)
		if err != nil {
			http.Error(w, SanitizeError(err, "repository not found"), http.StatusNotFound)
			return
		}

		// Parse the incoming objects and ref updates
		var req struct {
			Objects []struct {
				Hash    string `json:"hash"`
				Type    string `json:"type"`
				Content string `json:"content"` // base64
			} `json:"objects"`
			RefUpdates []struct {
				Name      string `json:"name"`
				OldHash   string `json:"old_hash"`
				NewHash   string `json:"new_hash"`
			} `json:"ref_updates"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Check branch protection rules before allowing ref updates
		for _, update := range req.RefUpdates {
			branch := update.Name
			if len(branch) > 11 && branch[:11] == "refs/heads/" {
				branch = branch[11:]
			}
			protection := protectionStore.Get(id, branch)
			if protection != nil {
				if protection.NoForcePush {
					// Check if this is a force push (old hash doesn't match current ref)
					if update.OldHash != "" && update.OldHash != "0000000000000000000000000000000000000000" {
						// Verify old hash matches current ref to detect force push
						currentRef, err := repo.GetBranch(branch)
						if err == nil && currentRef.String() != update.OldHash {
							http.Error(w, "force push rejected: branch '"+branch+"' is protected", http.StatusForbidden)
							return
						}
					}
				}
			}
		}

		// Store objects
		var errors []string
		objectHashes := make([]string, 0, len(req.Objects))
		for _, obj := range req.Objects {
			objectHashes = append(objectHashes, obj.Hash)
			hash := plumbing.NewHash(obj.Hash)
			content, err := base64.StdEncoding.DecodeString(obj.Content)
			if err != nil {
				errors = append(errors, "invalid base64 for "+obj.Hash)
				continue
			}
			objType := plumbing.AnyObject
			switch obj.Type {
			case "blob":
				objType = plumbing.BlobObject
			case "tree":
				objType = plumbing.TreeObject
			case "commit":
				objType = plumbing.CommitObject
			case "tag":
				objType = plumbing.TagObject
			}
			if err := repo.StoreObject(hash, objType, content); err != nil {
				errors = append(errors, "storing "+obj.Hash+": "+err.Error())
			}
		}

		// Update refs
		for _, update := range req.RefUpdates {
			if update.NewHash != "" && update.NewHash != "0000000000000000000000000000000000000000" {
				hash := plumbing.NewHash(update.NewHash)
				if err := repo.CreateBranch(update.Name, hash); err != nil {
					log.Printf("warning: failed to create branch %s: %v", update.Name, err)
					errors = append(errors, err.Error())
				}
			}
		}

		w.Header().Set("Content-Type", "application/json")
		if len(errors) > 0 {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"repo":    id,
				"errors":  errors,
			})
		} else {
			refHeads := make(map[string]string)
			for _, update := range req.RefUpdates {
				if update.NewHash != "" && update.NewHash != "0000000000000000000000000000000000000000" {
					refHeads[update.Name] = update.NewHash
				}
			}
			dispatchPushEvent(wm, id, objectHashes, refHeads)

			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": true,
				"repo":    id,
			})
		}
	}
}

// PushPackfile accepts a packfile and ref updates for more efficient pushing
func PushPackfile(registry *storage.RepositoryRegistry, protectionStore *storage.ProtectionStore, wm *webhooks.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")

		repo, err := registry.Open(id)
		if err != nil {
			http.Error(w, SanitizeError(err, "repository not found"), http.StatusNotFound)
			return
		}

		var req struct {
			Packfile   string `json:"packfile"` // base64-encoded packfile
			RefUpdates []struct {
				Name    string `json:"name"`
				OldHash string `json:"old_hash"`
				NewHash string `json:"new_hash"`
			} `json:"ref_updates"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Check branch protection rules
		for _, update := range req.RefUpdates {
			branch := update.Name
			if len(branch) > 11 && branch[:11] == "refs/heads/" {
				branch = branch[11:]
			}
			protection := protectionStore.Get(id, branch)
			if protection != nil && protection.NoForcePush {
				if update.OldHash != "" && update.OldHash != "0000000000000000000000000000000000000000" {
					currentRef, err := repo.GetBranch(branch)
					if err == nil && currentRef.String() != update.OldHash {
						http.Error(w, "force push rejected: branch '"+branch+"' is protected", http.StatusForbidden)
						return
					}
				}
			}
		}

		// Decode and ingest packfile
		objectHashes := make([]string, 0)
		if req.Packfile != "" {
			packData, err := base64.StdEncoding.DecodeString(req.Packfile)
			if err != nil {
				http.Error(w, "Invalid base64 packfile", http.StatusBadRequest)
				return
			}

			objects, err := storage.ExtractObjects(packData)
			if err != nil {
				http.Error(w, "Failed to parse packfile: "+err.Error(), http.StatusBadRequest)
				return
			}

			for _, obj := range objects {
				objectHashes = append(objectHashes, obj.Hash.String())
				if err := repo.StoreObject(obj.Hash, obj.Type, obj.Content); err != nil {
					log.Printf("warning: failed to store object %s: %v", obj.Hash, err)
				}
			}
		}

		// Update refs
		var errors []string
		for _, update := range req.RefUpdates {
			if update.NewHash != "" && update.NewHash != "0000000000000000000000000000000000000000" {
				hash := plumbing.NewHash(update.NewHash)
				if err := repo.CreateBranch(update.Name, hash); err != nil {
					log.Printf("warning: failed to create branch %s: %v", update.Name, err)
					errors = append(errors, err.Error())
				}
			}
		}

		refHeads := make(map[string]string)
		for _, update := range req.RefUpdates {
			if update.NewHash != "" && update.NewHash != "0000000000000000000000000000000000000000" {
				refHeads[update.Name] = update.NewHash
			}
		}
		dispatchPushEvent(wm, id, objectHashes, refHeads)

		w.Header().Set("Content-Type", "application/json")
		if len(errors) > 0 {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"repo":    id,
				"errors":  errors,
			})
		} else {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": true,
				"repo":    id,
			})
		}
	}
}

// CloneRepo clones a repository
func CloneRepo(registry *storage.RepositoryRegistry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")

		entry, err := registry.GetEntry(id)
		if err != nil {
			http.Error(w, SanitizeError(err, "repository not found"), http.StatusNotFound)
			return
		}

		repo, err := registry.Open(id)
		if err != nil {
			http.Error(w, SanitizeError(err, "failed to open repository"), http.StatusInternalServerError)
			return
		}

		refs, err := repo.ListAllRefs()
		if err != nil {
			http.Error(w, SanitizeError(err, "failed to list refs"), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":   entry.ID,
			"name": entry.Name,
			"refs": refs,
		})
	}
}

// GetObject returns a raw git object by hash
func GetObject(registry *storage.RepositoryRegistry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		hashStr := chi.URLParam(r, "hash")

		repo, err := registry.Open(id)
		if err != nil {
			http.Error(w, SanitizeError(err, "repository not found"), http.StatusNotFound)
			return
		}

		hash := plumbing.NewHash(hashStr)
		objType, content, err := repo.GetObject(hash)
		if err != nil {
			http.Error(w, SanitizeError(err, "object not found"), http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"hash":    hashStr,
			"type":    objType.String(),
			"content": content,
			"size":    len(content),
		})
	}
}

// ListRefs lists all references in a repository
func ListRefs(registry *storage.RepositoryRegistry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		offset, limit := ParsePagination(r)
		id := chi.URLParam(r, "id")

		repo, err := registry.Open(id)
		if err != nil {
			http.Error(w, SanitizeError(err, "repository not found"), http.StatusNotFound)
			return
		}

		refs, err := repo.ListAllRefs()
		if err != nil {
			http.Error(w, SanitizeError(err, "failed to list refs"), http.StatusInternalServerError)
			return
		}

		paged, total := PaginateSlice(refs, offset, limit)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"refs":   paged,
			"total":  total,
			"offset": offset,
			"limit":  limit,
		})
	}
}

// ForkRepo forks a repository
func ForkRepo(registry *storage.RepositoryRegistry, wm *webhooks.Manager, serverDID string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sourceID := chi.URLParam(r, "id")

		sourceEntry, err := registry.GetEntry(sourceID)
		if err != nil {
			http.Error(w, SanitizeError(err, "repository not found"), http.StatusNotFound)
			return
		}
		if sourceEntry.Private && !CanAccessRepo(r, sourceEntry, serverDID) {
			http.Error(w, "Repository not found", http.StatusNotFound)
			return
		}

		var req struct {
			Name string `json:"name"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if req.Name == "" {
			http.Error(w, "Name is required", http.StatusBadRequest)
			return
		}

		if !ValidateID(req.Name) {
			http.Error(w, "Invalid fork name: must be alphanumeric with hyphens/dots/underscores, max 64 chars", http.StatusBadRequest)
			return
		}

		entry, err := registry.Fork(sourceID, req.Name, "anonymous")
		if err != nil {
			if strings.Contains(err.Error(), "not found") {
				http.Error(w, SanitizeError(err, "repository not found"), http.StatusNotFound)
			} else if strings.Contains(err.Error(), "already exists") {
				http.Error(w, SanitizeError(err, "repository already exists"), http.StatusConflict)
			} else {
				http.Error(w, SanitizeError(err, "failed to fork repository"), http.StatusInternalServerError)
			}
			return
		}

		wm.Dispatch(webhooks.Event{
			Type: webhooks.EventRepoCreated,
			Repo: entry.Name,
			Data: map[string]interface{}{
				"id":          entry.ID,
				"forked_from": entry.ForkedFrom,
			},
		})

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":          entry.ID,
			"name":        entry.Name,
			"description": entry.Description,
			"private":     entry.Private,
			"created_at":  entry.CreatedAt,
			"forked_from": entry.ForkedFrom,
		})
	}
}

// CreateBranch creates a new branch
func CreateBranch(registry *storage.RepositoryRegistry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")

		var req struct {
			Name   string `json:"name"`
			Commit string `json:"commit"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		repo, err := registry.Open(id)
		if err != nil {
			http.Error(w, SanitizeError(err, "repository not found"), http.StatusNotFound)
			return
		}

		hash := plumbing.NewHash(req.Commit)
		if err := repo.CreateBranch(req.Name, hash); err != nil {
			http.Error(w, SanitizeError(err, "failed to create branch"), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"name":   req.Name,
			"commit": req.Commit,
		})
	}
}
