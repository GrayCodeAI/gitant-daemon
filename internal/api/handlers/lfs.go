package handlers

import (
	"fmt"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/lakshmanpatel/gitant/internal/lfs"
)

// LFSHandler handles LFS endpoints
type LFSHandler struct {
	store *lfs.Store
}

// NewLFSHandler creates a new LFS handler
func NewLFSHandler(store *lfs.Store) *LFSHandler {
	return &LFSHandler{store: store}
}

// Download handles object download
func (h *LFSHandler) Download(w http.ResponseWriter, r *http.Request) {
	oid := chi.URLParam(r, "oid")

	reader, obj, err := h.store.Download(oid)
	if err != nil {
		http.Error(w, SanitizeError(err, "LFS object not found"), http.StatusNotFound)
		return
	}
	defer reader.Close()

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", obj.Size))
	seeker, ok := reader.(io.ReadSeeker)
	if !ok {
		reader.Close()
		http.Error(w, "LFS object not seekable", http.StatusInternalServerError)
		return
	}
	http.ServeContent(w, r, "", obj.CreatedAt, seeker)
}
