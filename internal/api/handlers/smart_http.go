package handlers

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"

	"github.com/go-chi/chi/v5"
	"github.com/go-git/go-git/v6/plumbing"
	"github.com/lakshmanpatel/gitant/internal/git"
	"github.com/lakshmanpatel/gitant/internal/storage"
	"github.com/lakshmanpatel/gitant/internal/webhooks"
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
			http.Error(w, "failed to list refs", http.StatusInternalServerError)
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
		body, err := io.ReadAll(io.LimitReader(r.Body, 50<<20))
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
			slog.Error("error generating packfile", "error", err)
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
func GitReceivePack(registry *storage.RepositoryRegistry, protectionStore *storage.ProtectionStore, wm *webhooks.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")

		repo, err := registry.Open(id)
		if err != nil {
			http.Error(w, "repository not found", http.StatusNotFound)
			return
		}

		// Read the request body
		body, err := io.ReadAll(io.LimitReader(r.Body, 50<<20))
		if err != nil {
			http.Error(w, "reading body", http.StatusBadRequest)
			return
		}

		lines, packData := parseReceivePackRequest(body)
		if len(lines) == 0 {
			http.Error(w, "empty request", http.StatusBadRequest)
			return
		}

		// Parse ref updates (before the packfile)
		var updates []git.PushRefUpdate
		for _, line := range lines {
			if u := git.ParsePushRefUpdates([]string{line}); len(u) > 0 {
				updates = append(updates, u...)
			}
		}

		// Ingest packfile if present
		objectHashes := make([]string, 0)
		if len(packData) > 0 {
			hashes, err := ingestPackfile(repo, packData)
			if err != nil {
				slog.Error("error ingesting packfile", "error", err)
			} else {
				objectHashes = hashes
			}
		}

		statuses := applyReceivePackUpdates(registry, id, updates, protectionStore)

		// Send response
		w.Header().Set("Content-Type", "application/x-git-receive-pack-result")
		w.Header().Set("Cache-Control", "no-cache")

		var response strings.Builder
		response.WriteString(git.PktLine("unpack ok\n"))
		for _, status := range statuses {
			if status.OK {
				response.WriteString(git.PktLinef("ok %s\n", status.RefName))
			} else {
				response.WriteString(git.PktLinef("ng %s %s\n", status.RefName, status.Reason))
			}
		}
		response.WriteString(git.FlushPacket())
		w.Write([]byte(response.String()))

		refHeads := make(map[string]string)
		for _, status := range statuses {
			if status.OK && status.NewHash != "" && status.NewHash != "0000000000000000000000000000000000000000" {
				refHeads[status.RefName] = status.NewHash
			}
		}
		dispatchPushEvent(wm, id, objectHashes, refHeads)
	}
}

type receivePackRefStatus struct {
	RefName string
	NewHash string
	OK      bool
	Reason  string
}

func applyReceivePackUpdates(registry *storage.RepositoryRegistry, repoID string, updates []git.PushRefUpdate, protectionStore *storage.ProtectionStore) []receivePackRefStatus {
	statuses := make([]receivePackRefStatus, len(updates))
	var wg sync.WaitGroup
	for i, update := range updates {
		wg.Add(1)
		go func(i int, update git.PushRefUpdate) {
			defer wg.Done()
			statuses[i] = applyReceivePackUpdate(registry, repoID, update, protectionStore)
		}(i, update)
	}
	wg.Wait()
	return statuses
}

func applyReceivePackUpdate(registry *storage.RepositoryRegistry, repoID string, update git.PushRefUpdate, protectionStore *storage.ProtectionStore) receivePackRefStatus {
	status := receivePackRefStatus{RefName: update.RefName, NewHash: update.NewHash}
	repo, err := registry.Open(repoID)
	if err != nil {
		status.Reason = "repository not found"
		return status
	}
	if reason := validateReceivePackUpdate(repo, repoID, update, protectionStore); reason != "" {
		status.Reason = reason
		return status
	}

	if update.NewHash == "0000000000000000000000000000000000000000" {
		err = repo.DeleteRefIfMatches(update.RefName, update.OldHash)
	} else {
		err = repo.UpdateRefIfMatches(update.RefName, update.OldHash, plumbing.NewHash(update.NewHash))
	}
	if err != nil {
		status.Reason = "non-fast-forward"
		slog.Warn("failed to apply receive-pack ref update", "ref", update.RefName, "error", err)
	} else {
		status.OK = true
	}
	return status
}

func validateReceivePackUpdate(repo *storage.Repository, repoID string, update git.PushRefUpdate, protectionStore *storage.ProtectionStore) string {
	if update.RefName == "" || update.OldHash == "" || update.NewHash == "" {
		return "invalid ref update"
	}
	if update.NewHash == "0000000000000000000000000000000000000000" && protectedBranch(repoID, update.RefName, protectionStore) != "" {
		return "protected branch deletion denied"
	}
	current, exists, err := currentReceivePackRef(repo, update.RefName)
	if err != nil {
		return "cannot read current ref"
	}
	if update.OldHash == "0000000000000000000000000000000000000000" {
		if exists {
			return "non-fast-forward"
		}
		return ""
	}
	if !exists || current != update.OldHash {
		return "non-fast-forward"
	}
	if branch := protectedBranch(repoID, update.RefName, protectionStore); branch != "" && !isReceivePackFastForward(repo, update.OldHash, update.NewHash) {
		return "non-fast-forward"
	}
	return ""
}

func currentReceivePackRef(repo *storage.Repository, name string) (string, bool, error) {
	hash, err := repo.GetRef(name)
	if err != nil {
		if strings.Contains(err.Error(), "reference not found") || strings.Contains(err.Error(), "not found") {
			return "", false, nil
		}
		return "", false, err
	}
	return hash.String(), true, nil
}

func protectedBranch(repoID, refName string, protectionStore *storage.ProtectionStore) string {
	if protectionStore == nil || !strings.HasPrefix(refName, "refs/heads/") {
		return ""
	}
	branch := strings.TrimPrefix(refName, "refs/heads/")
	protection := protectionStore.Get(repoID, branch)
	if protection == nil {
		return ""
	}
	return branch
}

func isReceivePackFastForward(repo *storage.Repository, oldHash, newHash string) bool {
	if oldHash == newHash {
		return true
	}
	oldCommit, err := repo.GetCommit(plumbing.NewHash(oldHash))
	if err != nil {
		return false
	}
	newCommit, err := repo.GetCommit(plumbing.NewHash(newHash))
	if err != nil {
		return false
	}
	if oldCommit == nil || newCommit == nil {
		return false
	}

	seen := make(map[string]bool)
	stack := []plumbing.Hash{plumbing.NewHash(newHash)}
	for len(stack) > 0 {
		current := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if current.String() == oldHash {
			return true
		}
		if seen[current.String()] {
			continue
		}
		seen[current.String()] = true
		commit, err := repo.GetCommit(current)
		if err != nil {
			continue
		}
		stack = append(stack, commit.ParentHashes...)
	}
	return false
}

func parseReceivePackRequest(body []byte) ([]string, []byte) {
	var lines []string
	for i := 0; i < len(body); {
		if i+4 > len(body) {
			break
		}
		lengthHex := string(body[i : i+4])
		if lengthHex == "0000" {
			i += 4
			if i+4 <= len(body) && string(body[i:i+4]) == "PACK" {
				return lines, body[i:]
			}
			continue
		}
		var length int
		fmt.Sscanf(lengthHex, "%x", &length)
		if length < 4 || i+length > len(body) {
			break
		}
		lines = append(lines, string(body[i+4:i+length]))
		i += length
	}
	return lines, nil
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

// collectObjectsForWants collects all objects reachable from wants but not from haves.
func collectObjectsForWants(repo *storage.Repository, wants, haves []string) []plumbing.Hash {
	seen := make(map[string]bool)
	for _, have := range haves {
		collectReachableObjects(repo, plumbing.NewHash(have), seen, nil)
	}

	var objects []plumbing.Hash
	for _, want := range wants {
		collectReachableObjects(repo, plumbing.NewHash(want), seen, &objects)
	}
	return objects
}

// collectReachableObjects walks the object graph
func collectReachableObjects(repo *storage.Repository, hash plumbing.Hash, seen map[string]bool, objects *[]plumbing.Hash) {
	if seen[hash.String()] {
		return
	}
	seen[hash.String()] = true
	if objects != nil {
		*objects = append(*objects, hash)
	}

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
		tree, err := repo.GetTree(hash)
		if err != nil {
			return
		}
		for _, entry := range tree.Entries {
			collectReachableObjects(repo, entry.Hash, seen, objects)
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
func ingestPackfile(repo *storage.Repository, data []byte) ([]string, error) {
	objects, err := storage.ExtractObjects(data)
	if err != nil {
		return nil, fmt.Errorf("extracting packfile objects: %w", err)
	}

	hashes := make([]string, 0, len(objects))
	for _, obj := range objects {
		hashes = append(hashes, obj.Hash.String())
		if err := repo.StoreObject(obj.Hash, obj.Type, obj.Content); err != nil {
			slog.Warn("failed to store object", "hash", obj.Hash, "error", err)
		}
	}

	return hashes, nil
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
		length := len(chunk) + 5                   // +4 for length, +1 for channel
		header := fmt.Sprintf("%04x%c", length, 1) // channel 1
		if _, err := sw.w.Write([]byte(header)); err != nil {
			return 0, err
		}
		if _, err := sw.w.Write(chunk); err != nil {
			return 0, err
		}
	}
	return len(data), nil
}

func (sw *sidebandWriter) Close() error {
	_, err := sw.w.Write([]byte(git.FlushPacket()))
	return err
}
