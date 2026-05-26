package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/lakshmanpatel/gitant/internal/packages"
)

// PackageHandler handles package registry endpoints
type PackageHandler struct {
	registry *packages.Registry
}

// NewPackageHandler creates a new package handler
func NewPackageHandler(registry *packages.Registry) *PackageHandler {
	return &PackageHandler{registry: registry}
}

// ListPackages lists all packages
func (h *PackageHandler) ListPackages(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")

	var pkgs []*packages.Package
	if query != "" {
		pkgs = h.registry.Search(query)
	} else {
		pkgs = h.registry.List()
	}

	result := make([]map[string]interface{}, len(pkgs))
	for i, pkg := range pkgs {
		versions := make([]string, 0, len(pkg.Versions))
		for v := range pkg.Versions {
			versions = append(versions, v)
		}
		result[i] = map[string]interface{}{
			"name":        pkg.Name,
			"description": pkg.Description,
			"versions":    versions,
			"created_at":  pkg.CreatedAt,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"packages": result,
	})
}

// GetPackage gets a package by name
func (h *PackageHandler) GetPackage(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")

	pkg, err := h.registry.Get(name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	versions := make(map[string]interface{})
	for v, ver := range pkg.Versions {
		versions[v] = map[string]interface{}{
			"version":      ver.Version,
			"description":  ver.Description,
			"author":       ver.Author,
			"license":      ver.License,
			"dependencies": ver.Dependencies,
			"dist":         ver.Dist,
			"created_at":   ver.CreatedAt,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"name":        pkg.Name,
		"description": pkg.Description,
		"versions":    versions,
		"created_at":  pkg.CreatedAt,
		"updated_at":  pkg.UpdatedAt,
	})
}

// GetPackageVersion gets a specific version
func (h *PackageHandler) GetPackageVersion(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	version := chi.URLParam(r, "version")

	ver, err := h.registry.GetVersion(name, version)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"version":      ver.Version,
		"description":  ver.Description,
		"author":       ver.Author,
		"license":      ver.License,
		"dependencies": ver.Dependencies,
		"dist":         ver.Dist,
		"created_at":   ver.CreatedAt,
	})
}

// PublishPackage publishes a new package
func (h *PackageHandler) PublishPackage(w http.ResponseWriter, r *http.Request) {
	var pkg packages.Package
	if err := json.NewDecoder(r.Body).Decode(&pkg); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if pkg.Name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}

	if pkg.Versions == nil {
		pkg.Versions = make(map[string]*packages.Version)
	}

	if err := h.registry.Publish(&pkg); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"name":    pkg.Name,
	})
}

// DeletePackage deletes a package
func (h *PackageHandler) DeletePackage(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")

	if err := h.registry.Delete(name); err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
	})
}
