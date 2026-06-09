package api

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/go-git/go-git/v6/plumbing"
	"github.com/go-git/go-git/v6/plumbing/filemode"
	"github.com/lakshmanpatel/gitant/internal/crdt"
	gitproto "github.com/lakshmanpatel/gitant/internal/git"
	"github.com/lakshmanpatel/gitant/internal/identity"
	"github.com/lakshmanpatel/gitant/internal/storage"
	"github.com/lakshmanpatel/gitant/internal/webhooks"
)

func setupSmartHTTPRouteServer(t *testing.T) *Server {
	t.Helper()

	_, server := setupSmartHTTPRouteServerWithRegistry(t)
	return server
}

func setupSmartHTTPRouteServerWithRegistry(t *testing.T) (*storage.RepositoryRegistry, *Server) {
	t.Helper()

	dataDir := t.TempDir()
	reposDir := filepath.Join(dataDir, "repos")
	storeDir := filepath.Join(dataDir, "data")

	id, err := identity.NewIdentity()
	if err != nil {
		t.Fatal(err)
	}
	repos, err := storage.NewRepositoryRegistry(reposDir, storeDir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repos.Create("public-repo", "public-repo", "public", false); err != nil {
		t.Fatal(err)
	}
	if _, err := repos.Create("private-repo", "private-repo", "secret", true); err != nil {
		t.Fatal(err)
	}

	server := NewServer(
		7777,
		id,
		repos,
		crdt.NewIssueStore(""),
		crdt.NewPullRequestStore(""),
		storage.NewBlockstore(filepath.Join(storeDir, "blockstore.json"), filepath.Join(storeDir, "blocks")),
		crdt.NewLabelStore(""),
		crdt.NewTaskStore(""),
		crdt.NewReleaseStore(""),
		storage.NewProtectionStore(""),
		webhooks.NewManager(),
		identity.NewRevocationStore(""),
		storeDir,
		nil,
	)
	return repos, server
}

func TestServerUploadPackUsesReadAccessGroup(t *testing.T) {
	s := setupSmartHTTPRouteServer(t)

	req := httptest.NewRequest("POST", "/api/v1/repos/public-repo/git-upload-pack", bytes.NewBufferString(""))
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	if w.Code == http.StatusUnauthorized || w.Code == http.StatusForbidden {
		t.Fatalf("public upload-pack should not require identity or write capability, got %d: %s", w.Code, w.Body.String())
	}
	if w.Code != http.StatusBadRequest || w.Body.String() != "empty request\n" {
		t.Fatalf("expected upload-pack handler to reject empty body, got %d: %q", w.Code, w.Body.String())
	}
}

func TestServerPrivateUploadPackRequiresReadAccessNotWriteAccess(t *testing.T) {
	s := setupSmartHTTPRouteServer(t)

	req := httptest.NewRequest("POST", "/api/v1/repos/private-repo/git-upload-pack", bytes.NewBufferString(""))
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("private upload-pack without read access should be hidden, got %d: %s", w.Code, w.Body.String())
	}
}

func TestServerReceivePackStillRequiresIdentity(t *testing.T) {
	s := setupSmartHTTPRouteServer(t)

	req := httptest.NewRequest("POST", "/api/v1/repos/public-repo/git-receive-pack", bytes.NewBufferString(""))
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("receive-pack should remain identity/write gated, got %d: %s", w.Code, w.Body.String())
	}
}

func TestServerPublicUploadPackStreamsValidPackfile(t *testing.T) {
	reg, s := setupSmartHTTPRouteServerWithRegistry(t)
	commitHash := seedPublicRepoCommit(t, reg)

	requestBody := gitproto.PktLinef("want %s\n", commitHash.String()) + gitproto.FlushPacket() + gitproto.PktLine("done\n")
	req := httptest.NewRequest("POST", "/api/v1/repos/public-repo/git-upload-pack", bytes.NewBufferString(requestBody))
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected upload-pack success, got %d: %s", w.Code, w.Body.String())
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/x-git-upload-pack-result" {
		t.Fatalf("expected upload-pack result content type, got %q", ct)
	}

	packData := decodeSidebandPackfile(t, w.Body.Bytes())
	if !bytes.HasPrefix(packData, []byte("PACK")) {
		t.Fatalf("expected sideband payload to start with PACK, got %q", packData[:min(len(packData), 16)])
	}
	objects, err := storage.ExtractObjects(packData)
	if err != nil {
		t.Fatalf("expected parseable packfile: %v", err)
	}
	if len(objects) == 0 {
		t.Fatal("expected upload-pack to include at least one object")
	}
}

func seedPublicRepoCommit(t *testing.T, reg *storage.RepositoryRegistry) plumbing.Hash {
	t.Helper()

	repo, err := reg.Open("public-repo")
	if err != nil {
		t.Fatal(err)
	}
	blobHash, err := repo.CreateBlob([]byte("hello\n"))
	if err != nil {
		t.Fatal(err)
	}
	treeHash, err := repo.CreateTree([]storage.TreeEntry{{Name: "README.md", Mode: filemode.Regular, Hash: blobHash}})
	if err != nil {
		t.Fatal(err)
	}
	commitHash, err := repo.CreateCommit(treeHash, nil, "test", "initial commit")
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.UpdateRef("refs/heads/main", commitHash); err != nil {
		t.Fatal(err)
	}
	return commitHash
}

func decodeSidebandPackfile(t *testing.T, data []byte) []byte {
	t.Helper()

	var pack bytes.Buffer
	for len(data) > 0 {
		if len(data) < 4 {
			t.Fatalf("truncated pkt-line header in %q", data)
		}
		if string(data[:4]) == gitproto.FlushPacket() {
			break
		}
		length, err := strconv.ParseInt(string(data[:4]), 16, 32)
		if err != nil {
			t.Fatalf("invalid pkt-line length %q: %v", data[:4], err)
		}
		if length < 5 || int(length) > len(data) {
			t.Fatalf("invalid pkt-line length %d for %d bytes", length, len(data))
		}
		payload := data[4:length]
		if payload[0] != 1 {
			t.Fatalf("expected sideband channel 1, got %d", payload[0])
		}
		pack.Write(payload[1:])
		data = data[length:]
	}
	return pack.Bytes()
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
