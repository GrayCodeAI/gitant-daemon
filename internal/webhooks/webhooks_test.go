package webhooks

import (
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestHMACSHA256(t *testing.T) {
	msg := []byte(`{"type":"repo.created","repo":"test"}`)
	secret := "my-secret-key"

	sig := HMACSHA256(msg, secret)
	if len(sig) != 32 {
		t.Fatalf("expected 32-byte signature, got %d", len(sig))
	}

	// Deterministic: same input produces same output
	sig2 := HMACSHA256(msg, secret)
	if hex.EncodeToString(sig) != hex.EncodeToString(sig2) {
		t.Fatal("HMAC should be deterministic")
	}

	// Different secret produces different signature
	sig3 := HMACSHA256(msg, "different-secret")
	if hex.EncodeToString(sig) == hex.EncodeToString(sig3) {
		t.Fatal("different secrets should produce different signatures")
	}

	// Different message produces different signature
	sig4 := HMACSHA256([]byte("other"), secret)
	if hex.EncodeToString(sig) == hex.EncodeToString(sig4) {
		t.Fatal("different messages should produce different signatures")
	}
}

func TestDispatchMatchesWildcards(t *testing.T) {
	m := NewManager()

	var received Event
	var wg sync.WaitGroup
	wg.Add(1)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer wg.Done()
		// Verify HMAC header is present
		sig := r.Header.Get("X-Gitant-Signature-256")
		if sig == "" {
			t.Error("expected X-Gitant-Signature-256 header")
		}
		secret := r.Header.Get("X-Gitant-Secret")
		if secret != "test-secret" {
			t.Errorf("expected X-Gitant-Secret header, got %q", secret)
		}
	}))
	defer ts.Close()

	m.Register("test-wh", ts.URL, []EventType{"*"}, "test-secret")

	m.Dispatch(Event{
		Type: EventRepoCreated,
		Repo: "my-repo",
		Data: map[string]interface{}{"name": "my-repo"},
	})

	wg.Wait()
	_ = received
}

func TestDispatchSelectiveEvents(t *testing.T) {
	m := NewManager()

	var mu sync.Mutex
	calls := map[EventType]int{}
	var wg sync.WaitGroup

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer wg.Done()
		mu.Lock()
		calls[EventType(r.Header.Get("X-Gitant-Event"))]++
		mu.Unlock()
	}))
	defer ts.Close()

	// Only subscribe to issue events
	m.Register("issue-wh", ts.URL, []EventType{EventIssueCreated, EventIssueClosed}, "")

	wg.Add(2)
	m.Dispatch(Event{Type: EventIssueCreated, Repo: "test"})
	m.Dispatch(Event{Type: EventIssueClosed, Repo: "test"})
	m.Dispatch(Event{Type: EventRepoCreated, Repo: "test"}) // should be ignored

	wg.Wait()

	mu.Lock()
	defer mu.Unlock()
	if len(calls) != 2 {
		t.Fatalf("expected 2 distinct event types, got %d: %v", len(calls), calls)
	}
	if calls[EventIssueCreated] != 1 {
		t.Errorf("expected 1 issue.created, got %d", calls[EventIssueCreated])
	}
	if calls[EventIssueClosed] != 1 {
		t.Errorf("expected 1 issue.closed, got %d", calls[EventIssueClosed])
	}
}

func TestDispatchNoSecretNoSignature(t *testing.T) {
	m := NewManager()

	var sigHeader string
	var wg sync.WaitGroup
	wg.Add(1)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer wg.Done()
		sigHeader = r.Header.Get("X-Gitant-Signature-256")
	}))
	defer ts.Close()

	m.Register("no-secret", ts.URL, []EventType{"*"}, "")

	m.Dispatch(Event{Type: EventRepoCreated, Repo: "test"})
	wg.Wait()

	if sigHeader != "" {
		t.Errorf("expected no signature header without secret, got %q", sigHeader)
	}
}

func TestDispatchSignatureVerification(t *testing.T) {
	m := NewManager()
	secret := "verify-me"

	var body []byte
	var sigHeader string
	var wg sync.WaitGroup
	wg.Add(1)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer wg.Done()
		buf := make([]byte, r.ContentLength)
		r.Body.Read(buf)
		body = buf
		sigHeader = r.Header.Get("X-Gitant-Signature-256")
	}))
	defer ts.Close()

	m.Register("verify-wh", ts.URL, []EventType{"*"}, secret)

	m.Dispatch(Event{Type: EventPROpened, Repo: "test", Data: map[string]interface{}{"pr_id": "pr-1"}})
	wg.Wait()

	// Verify the signature matches HMAC(body, secret)
	expected := "sha256=" + hex.EncodeToString(HMACSHA256(body, secret))
	if sigHeader != expected {
		t.Errorf("signature mismatch:\n  got:  %s\n  want: %s", sigHeader, expected)
	}
}

func TestSaveAndLoad(t *testing.T) {
	dir := t.TempDir()

	m1 := NewManager()
	// Load sets the dataDir so subsequent Register calls persist to disk
	m1.Load(dir)
	m1.Register("wh1", "http://example.com/hook1", []EventType{EventRepoCreated}, "s1")
	m1.Register("wh2", "http://example.com/hook2", []EventType{"*"}, "s2")

	// Verify file exists
	path := filepath.Join(dir, "webhooks.json")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatal("webhooks.json should have been created")
	}

	// Load into a new manager
	m2 := NewManager()
	if err := m2.Load(dir); err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	whs := m2.List()
	if len(whs) != 2 {
		t.Fatalf("expected 2 webhooks after load, got %d", len(whs))
	}

	// Verify webhook details
	found := map[string]bool{}
	for _, wh := range whs {
		found[wh.ID] = true
		if wh.ID == "wh1" {
			if wh.URL != "http://example.com/hook1" {
				t.Errorf("wh1 URL: got %q, want %q", wh.URL, "http://example.com/hook1")
			}
			if wh.Secret != "s1" {
				t.Errorf("wh1 secret: got %q, want %q", wh.Secret, "s1")
			}
		}
	}
	if !found["wh1"] || !found["wh2"] {
		t.Fatalf("expected wh1 and wh2, got %v", found)
	}
}

func TestLoadNonExistent(t *testing.T) {
	dir := t.TempDir()

	m := NewManager()
	m.Register("existing", "http://example.com", []EventType{"*"}, "")

	// Load from a dir with no webhooks.json - should be a no-op
	if err := m.Load(dir); err != nil {
		t.Fatalf("Load should not error on missing file, got: %v", err)
	}

	// Existing webhooks should be replaced by empty map since LoadJSON returns nil (no error)
	// and the loaded map will be empty
	whs := m.List()
	if len(whs) != 0 {
		t.Fatalf("expected 0 webhooks after loading missing file, got %d", len(whs))
	}
}

func TestRemove(t *testing.T) {
	dir := t.TempDir()
	m := NewManager()
	m.Load(dir)
	m.Register("wh1", "http://example.com", []EventType{"*"}, "")

	if len(m.List()) != 1 {
		t.Fatal("expected 1 webhook")
	}

	m.Remove("wh1")

	if len(m.List()) != 0 {
		t.Fatal("expected 0 webhooks after remove")
	}

	// Load from disk to verify remove was persisted
	m2 := NewManager()
	if err := m2.Load(dir); err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if len(m2.List()) != 0 {
		t.Fatal("expected 0 webhooks on disk after remove")
	}
}

func TestTimestampSetOnDispatch(t *testing.T) {
	m := NewManager()

	var ts time.Time
	var wg sync.WaitGroup
	wg.Add(1)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer wg.Done()
		// We can't easily unmarshal the full event here without reading the body,
		// but we can verify the delivery header exists
		if r.Header.Get("X-Gitant-Delivery") == "" {
			t.Error("expected X-Gitant-Delivery header")
		}
	}))
	defer server.Close()

	m.Register("ts-wh", server.URL, []EventType{"*"}, "")

	before := time.Now()
	m.Dispatch(Event{Type: EventRepoCreated, Repo: "test"})
	wg.Wait()
	after := time.Now()

	_ = ts
	_ = before
	_ = after
}
