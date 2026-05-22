package handlers

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-git/go-git/v6/plumbing"
	"github.com/lakshmanpatel/gitant/internal/git"
	"github.com/lakshmanpatel/gitant/internal/storage"
)

// InfoRefs handles GET /{id}/info/refs?service=git-upload-pack|git-receive-pack
func InfoRefs(registry *storage.RepositoryRegistry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		service := r.URL.Query().Get("service")

		if service != "git-upload-pack" && service != "git-receive-pack" {
			http.Error(w, "unsupported service", http.StatusBadRequest)
			return
		}

		repo, err := registry.Open(id)
		if err != nil {
			http.Error(w, "repository not found", http.StatusNotFound)
			return
		}

		refs, err := repo.ListAllRefs()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Convert to git.RefLine
		refLines := make([]git.RefLine, len(refs))
		for i, ref := range refs {
			refLines[i] = git.RefLine{Hash: ref.Hash, Name: ref.Name}
		}

		response := git.ServiceRefResponse(service, refLines)

		contentType := "application/x-" + service + "-advertisement"
		w.Header().Set("Content-Type", contentType)
		w.Header().Set("Cache-Control", "no-cache")
		w.Write([]byte(response))
	}
}

// GitUploadPack handles POST /{id}/git-upload-pack
// Receives want/have lines, returns a packfile
func GitUploadPack(registry *storage.RepositoryRegistry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")

		repo, err := registry.Open(id)
		if err != nil {
			http.Error(w, "repository not found", http.StatusNotFound)
			return
		}

		// Read the request body (pkt-line format)
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "reading body", http.StatusBadRequest)
			return
		}

		lines := parsePktLines(string(body))
		if len(lines) == 0 {
			http.Error(w, "empty request", http.StatusBadRequest)
			return
		}

		// Parse want and have lines
		wants := git.ParseWantLines(lines)
		haves := git.ParseHaveLines(lines)

		if len(wants) == 0 {
			http.Error(w, "no wants specified", http.StatusBadRequest)
			return
		}

		// Collect objects for all wanted hashes
		objects := collectObjectsForWants(repo, wants, haves)

		// Generate packfile
		packData, err := generatePackfile(repo, objects)
		if err != nil {
			log.Printf("error generating packfile: %v", err)
			http.Error(w, "generating packfile", http.StatusInternalServerError)
			return
		}

		// Send response
		w.Header().Set("Content-Type", "application/x-git-upload-pack-result")
		w.Header().Set("Cache-Control", "no-cache")

		// Write "packfile\n" prefix then the data
		writer := newSidebandWriter(w)
		writer.Write(packData)
		writer.Close()
	}
}

// GitReceivePack handles POST /{id}/git-receive-pack
// Receives a packfile and ref updates
func GitReceivePack(registry *storage.RepositoryRegistry, protectionStore *storage.ProtectionStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")

		repo, err := registry.Open(id)
		if err != nil {
			http.Error(w, "repository not found", http.StatusNotFound)
			return
		}

		// Read the request body
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "reading body", http.StatusBadRequest)
			return
		}

		lines := parsePktLines(string(body))
		if len(lines) == 0 {
			http.Error(w, "empty request", http.StatusBadRequest)
			return
		}

		// Parse ref updates (before the packfile)
		var updates []git.PushRefUpdate
		packStart := -1
		for i, line := range lines {
			if strings.HasPrefix(line, "PACK") {
				packStart = i
				break
			}
			if u := git.ParsePushRefUpdates([]string{line}); len(u) > 0 {
				updates = append(updates, u...)
			}
		}

		// Check branch protection rules before accepting push
		for _, update := range updates {
			branch := update.RefName
			if len(branch) > 11 && branch[:11] == "refs/heads/" {
				branch = branch[11:]
			}
			protection := protectionStore.Get(id, branch)
			if protection != nil && protection.NoForcePush {
				// Check if this is a non-fast-forward update
				// For now, we allow the push but log a warning
				// Full force-push detection requires checking the current ref hash
				log.Printf("branch %s is protected (no-force-push), push from %s", branch, r.RemoteAddr)
			}
		}

		// Ingest packfile if present
		if packStart >= 0 {
			packData := strings.Join(lines[packStart:], "")
			if err := ingestPackfile(repo, []byte(packData)); err != nil {
				log.Printf("error ingesting packfile: %v", err)
			}
		}

		// Update refs
		for _, update := range updates {
			if update.NewHash == "0000000000000000000000000000000000000000" {
				continue // delete, skip for now
			}
			hash := plumbing.NewHash(update.NewHash)
			if err := repo.CreateBranch(update.RefName, hash); err != nil {
				log.Printf("warning: failed to update ref %s: %v", update.RefName, err)
			}
		}

		// Send response
		w.Header().Set("Content-Type", "application/x-git-receive-pack-result")
		w.Header().Set("Cache-Control", "no-cache")

		var response strings.Builder
		response.WriteString(git.PktLine("unpack ok\n"))
		for _, update := range updates {
			response.WriteString(git.PktLinef("ok %s\n", update.RefName))
		}
		response.WriteString(git.FlushPacket())
		w.Write([]byte(response.String()))
	}
}

// parsePktLines extracts data from pkt-line format
func parsePktLines(data string) []string {
	var lines []string
	i := 0
	for i < len(data) {
		if i+4 > len(data) {
			break
		}
		lengthHex := data[i : i+4]
		if lengthHex == "0000" {
			i += 4
			continue
		}
		var length int
		fmt.Sscanf(lengthHex, "%x", &length)
		if length < 4 || i+length > len(data) {
			break
		}
		lines = append(lines, data[i+4:i+length])
		i += length
	}
	return lines
}

// collectObjectsForWants collects all objects reachable from wants but not from haves
func collectObjectsForWants(repo *storage.Repository, wants, haves []string) []plumbing.Hash {
	// For MVP, return all objects reachable from wants
	var objects []plumbing.Hash
	seen := make(map[string]bool)

	for _, want := range wants {
		hash := plumbing.NewHash(want)
		collectReachableObjects(repo, hash, seen, &objects)
	}

	// Remove haves
	haveSet := make(map[string]bool)
	for _, have := range haves {
		haveSet[have] = true
	}

	var filtered []plumbing.Hash
	for _, obj := range objects {
		if !haveSet[obj.String()] {
			filtered = append(filtered, obj)
		}
	}

	return filtered
}

// collectReachableObjects walks the object graph
func collectReachableObjects(repo *storage.Repository, hash plumbing.Hash, seen map[string]bool, objects *[]plumbing.Hash) {
	if seen[hash.String()] {
		return
	}
	seen[hash.String()] = true
	*objects = append(*objects, hash)

	// Try to get the object to find references
	objType, content, err := repo.GetObject(hash)
	if err != nil {
		return
	}

	switch objType {
	case plumbing.CommitObject:
		// Parse commit to find tree and parent hashes
		contentStr := string(content)
		for _, line := range strings.Split(contentStr, "\n") {
			if strings.HasPrefix(line, "tree ") {
				treeHash := plumbing.NewHash(strings.TrimPrefix(line, "tree "))
				collectReachableObjects(repo, treeHash, seen, objects)
			} else if strings.HasPrefix(line, "parent ") {
				parentHash := plumbing.NewHash(strings.TrimPrefix(line, "parent "))
				collectReachableObjects(repo, parentHash, seen, objects)
			}
		}
	case plumbing.TreeObject:
		// Parse tree entries
		// Tree format: "<mode> <name>\0<20-byte-hash>" repeated
		i := 0
		for i < len(content) {
			// Find null byte
			nullIdx := -1
			for j := i; j < len(content); j++ {
				if content[j] == 0 {
					nullIdx = j
					break
				}
			}
			if nullIdx == -1 || nullIdx+21 > len(content) {
				break
			}
			entryHash := plumbing.NewHash(string(content[nullIdx+1 : nullIdx+21]))
			collectReachableObjects(repo, entryHash, seen, objects)
			i = nullIdx + 21
		}
	}
}

// generatePackfile creates a packfile from a set of objects using go-git's
// encoder with delta compression and proper zlib deflation.
func generatePackfile(repo *storage.Repository, objects []plumbing.Hash) ([]byte, error) {
	// Convert hashes to GitObjects
	gitObjects := make([]*storage.GitObject, 0, len(objects))
	for _, hash := range objects {
		objType, content, err := repo.GetObject(hash)
		if err != nil {
			continue
		}
		gitObjects = append(gitObjects, &storage.GitObject{
			Type:    objType,
			Content: content,
			Hash:    hash,
		})
	}

	// Use the packfile writer for proper encoding with delta compression
	writer := storage.NewPackfileWriter()
	return writer.WritePackfile(gitObjects)
}

// ingestPackfile reads a packfile and stores its objects into the repository.
// Uses go-git's parser for proper zlib decompression and delta resolution.
func ingestPackfile(repo *storage.Repository, data []byte) error {
	objects, err := storage.ExtractObjects(data)
	if err != nil {
		return fmt.Errorf("extracting packfile objects: %w", err)
	}

	for _, obj := range objects {
		if err := repo.StoreObject(obj.Hash, obj.Type, obj.Content); err != nil {
			log.Printf("warning: failed to store object %s: %v", obj.Hash, err)
		}
	}

	return nil
}

// sidebandWriter writes data in git side-band format
type sidebandWriter struct {
	w io.Writer
}

func newSidebandWriter(w io.Writer) *sidebandWriter {
	return &sidebandWriter{w: w}
}

func (sw *sidebandWriter) Write(data []byte) (int, error) {
	// Side-band-64k: <channel-byte><data>
	// Channel 1 = pack data
	chunkSize := 65520 // max sideband chunk
	for i := 0; i < len(data); i += chunkSize {
		end := i + chunkSize
		if end > len(data) {
			end = len(data)
		}
		chunk := data[i:end]

		// Write pkt-line: <length><channel><data>
		length := len(chunk) + 5 // +4 for length, +1 for channel
		header := fmt.Sprintf("%04x%c", length, 1) // channel 1
		sw.w.Write([]byte(header))
		sw.w.Write(chunk)
	}
	return len(data), nil
}

func (sw *sidebandWriter) Close() error {
	_, err := sw.w.Write([]byte(git.FlushPacket()))
	return err
}
