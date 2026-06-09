package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/go-git/go-git/v6/plumbing"
	"github.com/go-git/go-git/v6/plumbing/filemode"
	"github.com/lakshmanpatel/gitant/internal/git"
	"github.com/lakshmanpatel/gitant/internal/storage"
)

func TestUploadPackHonorsHavesByPruningReachableObjects(t *testing.T) {
	reg := setupTestRegistry(t)
	if _, err := reg.Create("repo", "repo", "", false); err != nil {
		t.Fatal(err)
	}
	repo := openTestRepo(t, reg, "repo")

	base := createTestCommit(t, repo, "base.txt", "base", nil)
	tip := createTestCommit(t, repo, "tip.txt", "tip", []plumbing.Hash{base.Commit})

	objects := collectObjectsForWants(repo, []string{tip.Commit.String()}, []string{base.Commit.String()})
	objectSet := make(map[string]bool, len(objects))
	for _, object := range objects {
		objectSet[object.String()] = true
	}

	for _, excluded := range []plumbing.Hash{base.Commit, base.Tree, base.Blob} {
		if objectSet[excluded.String()] {
			t.Fatalf("expected have-reachable object %s to be pruned, got objects %v", excluded, objects)
		}
	}
	for _, included := range []plumbing.Hash{tip.Commit, tip.Tree, tip.Blob} {
		if !objectSet[included.String()] {
			t.Fatalf("expected wanted-tip object %s to be included, got objects %v", included, objects)
		}
	}
}

func TestGitUploadPackWithTipHaveReturnsMinimalPack(t *testing.T) {
	reg := setupTestRegistry(t)
	if _, err := reg.Create("repo", "repo", "", false); err != nil {
		t.Fatal(err)
	}
	repo := openTestRepo(t, reg, "repo")
	commit := createTestCommit(t, repo, "README.md", "hello", nil)
	if err := repo.UpdateRef("refs/heads/main", commit.Commit); err != nil {
		t.Fatal(err)
	}

	r := chiRouter()
	r.Post("/{id}/git-upload-pack", GitUploadPack(reg))
	requestBody := git.PktLinef("want %s\n", commit.Commit) + git.PktLinef("have %s\n", commit.Commit) + git.FlushPacket() + git.PktLine("done\n")
	req := httptest.NewRequest("POST", "/repo/git-upload-pack", bytes.NewBufferString(requestBody))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected upload-pack success, got %d: %s", w.Code, w.Body.String())
	}

	packData := decodeTestSidebandPackfile(t, w.Body.Bytes())
	objects, err := storage.ExtractObjects(packData)
	if err != nil {
		t.Fatalf("expected parseable minimal packfile: %v", err)
	}
	if len(objects) != 0 {
		t.Fatalf("expected up-to-date have to produce no new objects, got %d", len(objects))
	}
}

func TestGitUploadPackWithBaseHaveReturnsOnlyNewObjects(t *testing.T) {
	reg := setupTestRegistry(t)
	if _, err := reg.Create("repo", "repo", "", false); err != nil {
		t.Fatal(err)
	}
	repo := openTestRepo(t, reg, "repo")
	base := createTestCommit(t, repo, "base.txt", "base", nil)
	tip := createTestCommit(t, repo, "tip.txt", "tip", []plumbing.Hash{base.Commit})
	if err := repo.UpdateRef("refs/heads/main", tip.Commit); err != nil {
		t.Fatal(err)
	}

	r := chiRouter()
	r.Post("/{id}/git-upload-pack", GitUploadPack(reg))
	requestBody := git.PktLinef("want %s\n", tip.Commit) + git.PktLinef("have %s\n", base.Commit) + git.FlushPacket() + git.PktLine("done\n")
	req := httptest.NewRequest("POST", "/repo/git-upload-pack", bytes.NewBufferString(requestBody))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected upload-pack success, got %d: %s", w.Code, w.Body.String())
	}

	packData := decodeTestSidebandPackfile(t, w.Body.Bytes())
	objects, err := storage.ExtractObjects(packData)
	if err != nil {
		t.Fatalf("expected parseable packfile: %v", err)
	}
	objectSet := make(map[string]bool, len(objects))
	for _, object := range objects {
		objectSet[object.Hash.String()] = true
	}

	for _, excluded := range []plumbing.Hash{base.Commit, base.Tree, base.Blob} {
		if objectSet[excluded.String()] {
			t.Fatalf("expected have-reachable object %s to be pruned, got objects %v", excluded, objects)
		}
	}
	for _, included := range []plumbing.Hash{tip.Commit, tip.Tree, tip.Blob} {
		if !objectSet[included.String()] {
			t.Fatalf("expected new wanted object %s in pack, got objects %v", included, objects)
		}
	}
}

func TestGitReceivePackReportsMixedOKAndNGStatus(t *testing.T) {
	reg := setupReceivePackRepo(t)
	protection := setupTestProtectionStore(t)
	protection.Set("repo", storage.BranchProtection{Branch: "main", NoForcePush: true})
	repo := openTestRepo(t, reg, "repo")
	base := createTestCommit(t, repo, "base.txt", "base", nil)
	mainTip := createTestCommit(t, repo, "main.txt", "main", []plumbing.Hash{base.Commit})
	newFeature := createTestCommit(t, repo, "feature.txt", "feature", []plumbing.Hash{mainTip.Commit})
	if err := repo.UpdateRef("refs/heads/main", mainTip.Commit); err != nil {
		t.Fatal(err)
	}

	requestBody := git.PktLinef("%s %s refs/heads/feature\n", zeroHash, newFeature.Commit) +
		git.PktLinef("%s %s refs/heads/main\n", base.Commit, newFeature.Commit) +
		git.FlushPacket()
	response := postReceivePack(t, reg, protection, requestBody)
	lines := parsePktLines(response.Body.String())

	assertPktLinePresent(t, lines, "unpack ok")
	assertPktLinePresent(t, lines, "ok refs/heads/feature")
	assertPktLinePresent(t, lines, "ng refs/heads/main non-fast-forward")

	refs := listRefsByName(t, reg, "repo")
	if refs["refs/heads/feature"] != newFeature.Commit.String() {
		t.Fatalf("expected feature ref to advance to %s, got %q", newFeature.Commit, refs["refs/heads/feature"])
	}
	if refs["refs/heads/main"] != mainTip.Commit.String() {
		t.Fatalf("expected rejected main ref to remain %s, got %q", mainTip.Commit, refs["refs/heads/main"])
	}
}

func TestGitReceivePackRejectsProtectedBranchDeletion(t *testing.T) {
	reg := setupReceivePackRepo(t)
	protection := setupTestProtectionStore(t)
	protection.Set("repo", storage.BranchProtection{Branch: "main", NoForcePush: true})
	repo := openTestRepo(t, reg, "repo")
	commit := createTestCommit(t, repo, "README.md", "hello", nil)
	if err := repo.UpdateRef("refs/heads/main", commit.Commit); err != nil {
		t.Fatal(err)
	}

	requestBody := git.PktLinef("%s %s refs/heads/main\n", commit.Commit, zeroHash) + git.FlushPacket()
	response := postReceivePack(t, reg, protection, requestBody)
	lines := parsePktLines(response.Body.String())

	assertPktLinePresent(t, lines, "unpack ok")
	assertPktLinePresent(t, lines, "ng refs/heads/main protected branch deletion denied")
	refs := listRefsByName(t, reg, "repo")
	if refs["refs/heads/main"] != commit.Commit.String() {
		t.Fatalf("expected protected branch to remain %s, got %q", commit.Commit, refs["refs/heads/main"])
	}
}

func TestGitReceivePackAcceptsFirstPushToEmptyRepo(t *testing.T) {
	reg := setupReceivePackRepo(t)
	protection := setupTestProtectionStore(t)
	repo := openTestRepo(t, reg, "repo")
	commit := createTestCommit(t, repo, "README.md", "hello", nil)

	requestBody := git.PktLinef("%s %s refs/heads/main\n", zeroHash, commit.Commit) + git.FlushPacket()
	response := postReceivePack(t, reg, protection, requestBody)
	lines := parsePktLines(response.Body.String())

	assertPktLinePresent(t, lines, "unpack ok")
	assertPktLinePresent(t, lines, "ok refs/heads/main")
	refs := listRefsByName(t, reg, "repo")
	if refs["refs/heads/main"] != commit.Commit.String() {
		t.Fatalf("expected first push to create main at %s, got %q", commit.Commit, refs["refs/heads/main"])
	}
}

func TestListRefsEnumeratesHeadsAndTagsWithHashes(t *testing.T) {
	reg := setupReceivePackRepo(t)
	repo := openTestRepo(t, reg, "repo")
	main := createTestCommit(t, repo, "main.txt", "main", nil)
	feature := createTestCommit(t, repo, "feature.txt", "feature", []plumbing.Hash{main.Commit})
	if err := repo.UpdateRef("refs/heads/main", main.Commit); err != nil {
		t.Fatal(err)
	}
	if err := repo.UpdateRef("refs/heads/feature", feature.Commit); err != nil {
		t.Fatal(err)
	}
	if err := repo.UpdateRef("refs/tags/v1", main.Commit); err != nil {
		t.Fatal(err)
	}

	refs := listRefsByName(t, reg, "repo")
	want := map[string]string{
		"refs/heads/main":    main.Commit.String(),
		"refs/heads/feature": feature.Commit.String(),
		"refs/tags/v1":       main.Commit.String(),
	}
	for name, hash := range want {
		if refs[name] != hash {
			t.Fatalf("expected %s=%s, got %q in refs %#v", name, hash, refs[name], refs)
		}
	}
}

func TestGitReceivePackFastForwardAdvancesTip(t *testing.T) {
	reg := setupReceivePackRepo(t)
	protection := setupTestProtectionStore(t)
	repo := openTestRepo(t, reg, "repo")
	base := createTestCommit(t, repo, "base.txt", "base", nil)
	fastForward := createTestCommit(t, repo, "next.txt", "next", []plumbing.Hash{base.Commit})
	if err := repo.UpdateRef("refs/heads/main", base.Commit); err != nil {
		t.Fatal(err)
	}

	requestBody := git.PktLinef("%s %s refs/heads/main\n", base.Commit, fastForward.Commit) + git.FlushPacket()
	response := postReceivePack(t, reg, protection, requestBody)
	lines := parsePktLines(response.Body.String())

	assertPktLinePresent(t, lines, "ok refs/heads/main")
	refs := listRefsByName(t, reg, "repo")
	if refs["refs/heads/main"] != fastForward.Commit.String() {
		t.Fatalf("expected fast-forward to advance main to %s, got %q", fastForward.Commit, refs["refs/heads/main"])
	}
}

func TestReceivePackConcurrentPushesExactlyOneWins(t *testing.T) {
	reg := setupReceivePackRepo(t)
	protection := setupTestProtectionStore(t)
	repo := openTestRepo(t, reg, "repo")
	base := createTestCommit(t, repo, "base.txt", "base", nil)
	left := createTestCommit(t, repo, "left.txt", "left", []plumbing.Hash{base.Commit})
	right := createTestCommit(t, repo, "right.txt", "right", []plumbing.Hash{base.Commit})
	if err := repo.UpdateRef("refs/heads/main", base.Commit); err != nil {
		t.Fatal(err)
	}

	start := make(chan struct{})
	results := make(chan receivePackRefStatus, 2)
	var wg sync.WaitGroup
	for _, next := range []plumbing.Hash{left.Commit, right.Commit} {
		wg.Add(1)
		go func(next plumbing.Hash) {
			defer wg.Done()
			<-start
			statuses := applyReceivePackUpdates(reg, "repo", []git.PushRefUpdate{{OldHash: base.Commit.String(), NewHash: next.String(), RefName: "refs/heads/main"}}, protection)
			results <- statuses[0]
		}(next)
	}
	close(start)
	wg.Wait()
	close(results)

	okCount := 0
	ngCount := 0
	var winner string
	for result := range results {
		if result.OK {
			okCount++
			winner = result.NewHash
		} else if result.Reason == "non-fast-forward" {
			ngCount++
		}
	}
	if okCount != 1 || ngCount != 1 {
		t.Fatalf("expected exactly one ok and one non-fast-forward ng, got ok=%d ng=%d", okCount, ngCount)
	}
	refs := listRefsByName(t, reg, "repo")
	if refs["refs/heads/main"] != winner {
		t.Fatalf("expected final main ref to equal winner %s, got %q", winner, refs["refs/heads/main"])
	}
}

func TestPushPackfileFastForwardAdvancesTip(t *testing.T) {
	reg := setupReceivePackRepo(t)
	protection := setupTestProtectionStore(t)
	repo := openTestRepo(t, reg, "repo")
	base := createTestCommit(t, repo, "base.txt", "base", nil)
	fastForward := createTestCommit(t, repo, "next.txt", "next", []plumbing.Hash{base.Commit})
	if err := repo.UpdateRef("refs/heads/main", base.Commit); err != nil {
		t.Fatal(err)
	}

	result := postPushPackfile(t, reg, protection, base.Commit, fastForward.Commit)
	if !result.Success || len(result.Errors) != 0 {
		t.Fatalf("expected push-packfile fast-forward to succeed, got success=%v errors=%v", result.Success, result.Errors)
	}
	refs := listRefsByName(t, reg, "repo")
	if refs["refs/heads/main"] != fastForward.Commit.String() {
		t.Fatalf("expected fast-forward to advance main to %s, got %q", fastForward.Commit, refs["refs/heads/main"])
	}
}

func TestPushPackfileConcurrentPushesExactlyOneWins(t *testing.T) {
	reg := setupReceivePackRepo(t)
	protection := setupTestProtectionStore(t)
	repo := openTestRepo(t, reg, "repo")
	base := createTestCommit(t, repo, "base.txt", "base", nil)
	left := createTestCommit(t, repo, "left.txt", "left", []plumbing.Hash{base.Commit})
	right := createTestCommit(t, repo, "right.txt", "right", []plumbing.Hash{base.Commit})
	if err := repo.UpdateRef("refs/heads/main", base.Commit); err != nil {
		t.Fatal(err)
	}

	start := make(chan struct{})
	results := make(chan pushPackfileResult, 2)
	var wg sync.WaitGroup
	for _, next := range []plumbing.Hash{left.Commit, right.Commit} {
		wg.Add(1)
		go func(next plumbing.Hash) {
			defer wg.Done()
			<-start
			results <- postPushPackfile(t, reg, protection, base.Commit, next)
		}(next)
	}
	close(start)
	wg.Wait()
	close(results)

	okCount := 0
	ngCount := 0
	var winner string
	for result := range results {
		if result.Success {
			okCount++
			winner = result.NewHash
			continue
		}
		for _, err := range result.Errors {
			if strings.Contains(err, "refs/heads/main non-fast-forward") {
				ngCount++
			}
		}
	}
	if okCount != 1 || ngCount != 1 {
		t.Fatalf("expected exactly one ok and one non-fast-forward ng, got ok=%d ng=%d", okCount, ngCount)
	}
	refs := listRefsByName(t, reg, "repo")
	if refs["refs/heads/main"] != winner {
		t.Fatalf("expected final main ref to equal winner %s, got %q", winner, refs["refs/heads/main"])
	}
}

const zeroHash = "0000000000000000000000000000000000000000"

type commitFixture struct {
	Blob   plumbing.Hash
	Tree   plumbing.Hash
	Commit plumbing.Hash
}

func setupReceivePackRepo(t *testing.T) *storage.RepositoryRegistry {
	t.Helper()
	reg := setupTestRegistry(t)
	if _, err := reg.Create("repo", "repo", "", false); err != nil {
		t.Fatal(err)
	}
	return reg
}

func openTestRepo(t *testing.T, reg *storage.RepositoryRegistry, id string) *storage.Repository {
	t.Helper()
	repo, err := reg.Open(id)
	if err != nil {
		t.Fatal(err)
	}
	return repo
}

func createTestCommit(t *testing.T, repo *storage.Repository, fileName, content string, parents []plumbing.Hash) commitFixture {
	t.Helper()
	blob, err := repo.CreateBlob([]byte(content))
	if err != nil {
		t.Fatal(err)
	}
	tree, err := repo.CreateTree([]storage.TreeEntry{{Name: fileName, Mode: filemode.Regular, Hash: blob}})
	if err != nil {
		t.Fatal(err)
	}
	commit, err := repo.CreateCommit(tree, parents, "test", content)
	if err != nil {
		t.Fatal(err)
	}
	return commitFixture{Blob: blob, Tree: tree, Commit: commit}
}

func postReceivePack(t *testing.T, reg *storage.RepositoryRegistry, protection *storage.ProtectionStore, body string) *httptest.ResponseRecorder {
	t.Helper()
	r := chiRouter()
	r.Post("/{id}/git-receive-pack", GitReceivePack(reg, protection, setupTestWebhookManager(t)))
	req := httptest.NewRequest("POST", "/repo/git-receive-pack", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected receive-pack status 200, got %d: %s", w.Code, w.Body.String())
	}
	return w
}

type pushPackfileResult struct {
	Success bool
	Errors  []string
	NewHash string
}

func postPushPackfile(t *testing.T, reg *storage.RepositoryRegistry, protection *storage.ProtectionStore, oldHash, newHash plumbing.Hash) pushPackfileResult {
	t.Helper()
	r := chiRouter()
	r.Post("/{id}/push-packfile", PushPackfile(reg, protection, setupTestWebhookManager(t)))
	body := fmt.Sprintf(`{"ref_updates":[{"name":"refs/heads/main","old_hash":"%s","new_hash":"%s"}]}`, oldHash, newHash)
	req := httptest.NewRequest("POST", "/repo/push-packfile", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected push-packfile status 200, got %d: %s", w.Code, w.Body.String())
	}
	var result pushPackfileResult
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode push-packfile response: %v", err)
	}
	result.NewHash = newHash.String()
	return result
}

func decodeTestSidebandPackfile(t *testing.T, data []byte) []byte {
	t.Helper()
	lines := parsePktLines(string(data))
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "NAK" {
		t.Fatalf("expected upload-pack response to start with NAK, got %q", lines)
	}
	if len(data) < len(git.PktLine("NAK\n")) {
		t.Fatalf("truncated NAK pkt-line in %q", data)
	}
	data = data[len(git.PktLine("NAK\n")):]

	var pack bytes.Buffer
	for len(data) > 0 {
		if len(data) < 4 {
			t.Fatalf("truncated pkt-line header in %q", data)
		}
		if string(data[:4]) == git.FlushPacket() {
			break
		}
		var length int
		if _, err := fmt.Sscanf(string(data[:4]), "%x", &length); err != nil {
			t.Fatalf("invalid pkt-line length %q: %v", data[:4], err)
		}
		if length < 5 || length > len(data) {
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

func assertPktLinePresent(t *testing.T, lines []string, want string) {
	t.Helper()
	for _, line := range lines {
		if strings.TrimSpace(line) == want {
			return
		}
	}
	t.Fatalf("expected pkt-line %q in %q", want, lines)
}

func listRefsByName(t *testing.T, reg *storage.RepositoryRegistry, repoID string) map[string]string {
	t.Helper()
	r := chiRouter()
	r.Get("/{id}/refs", ListRefs(reg))
	req := httptest.NewRequest("GET", "/"+repoID+"/refs", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected refs status 200, got %d: %s", w.Code, w.Body.String())
	}
	var result struct {
		Refs []storage.RefInfo `json:"refs"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode refs: %v", err)
	}
	refs := make(map[string]string, len(result.Refs))
	for _, ref := range result.Refs {
		refs[ref.Name] = ref.Hash
	}
	return refs
}
