package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

// newTestServer creates an httptest server that responds to API routes
// based on the handler map. Keys are "METHOD /path" patterns.
func newTestServer(t *testing.T, handlers map[string]http.HandlerFunc) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	for pattern, handler := range handlers {
		mux.HandleFunc(pattern, handler)
	}
	ts := httptest.NewServer(mux)
	t.Cleanup(ts.Close)
	return ts
}

// jsonHandler returns a handler that writes v as JSON.
func jsonHandler(status int, v interface{}) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		json.NewEncoder(w).Encode(v)
	}
}

// execCmd runs a cobra command with the given args and returns stdout.
// It appends --daemon-url=<testServerURL> to the args (subcommand flags).
// Captures both os.Stdout and os.Stderr since commands use fmt.Printf/Fprintf directly.
func execCmd(t *testing.T, ts *httptest.Server, args []string) string {
	t.Helper()

	// Capture stdout
	oldStdout := os.Stdout
	oldStderr := os.Stderr
	rOut, wOut, _ := os.Pipe()
	rErr, wErr, _ := os.Pipe()
	os.Stdout = wOut
	os.Stderr = wErr

	fullArgs := append(args, "--daemon-url", ts.URL)
	rootCmd.SetArgs(fullArgs)
	err := rootCmd.Execute()

	wOut.Close()
	wErr.Close()
	os.Stdout = oldStdout
	os.Stderr = oldStderr

	var buf bytes.Buffer
	buf.ReadFrom(rOut)
	buf.ReadFrom(rErr)

	if err != nil {
		t.Fatalf("command failed: %v\noutput: %s", err, buf.String())
	}
	return buf.String()
}

// readBody reads the request body as a string.
func readBody(r *http.Request) string {
	body, _ := io.ReadAll(r.Body)
	return string(body)
}

// --------------- Repo commands ---------------

func TestRepoList(t *testing.T) {
	ts := newTestServer(t, map[string]http.HandlerFunc{
		"GET /api/v1/repos": jsonHandler(200, map[string]interface{}{
			"repos": []map[string]interface{}{
				{"id": "repo1", "name": "my-repo", "description": "test repo", "stars": 5},
				{"id": "repo2", "name": "other-repo", "description": "", "stars": 0},
			},
			"total": 2,
		}),
	})

	out := execCmd(t, ts, []string{"repo", "list"})
	if !strings.Contains(out, "repo1") || !strings.Contains(out, "my-repo") {
		t.Errorf("expected repo1/my-repo in output, got: %s", out)
	}
	if !strings.Contains(out, "stars=5") {
		t.Errorf("expected stars=5 in output, got: %s", out)
	}
}

func TestRepoStar(t *testing.T) {
	ts := newTestServer(t, map[string]http.HandlerFunc{
		"POST /api/v1/repos/repo1/star": jsonHandler(200, map[string]interface{}{"stars": 6}),
	})

	out := execCmd(t, ts, []string{"repo", "star", "repo1"})
	if !strings.Contains(out, "Starred repo1") {
		t.Errorf("expected 'Starred repo1' in output, got: %s", out)
	}
	if !strings.Contains(out, "6 stars") {
		t.Errorf("expected '6 stars' in output, got: %s", out)
	}
}

func TestRepoUnstar(t *testing.T) {
	ts := newTestServer(t, map[string]http.HandlerFunc{
		"POST /api/v1/repos/repo1/unstar": jsonHandler(200, map[string]interface{}{"stars": 4}),
	})

	out := execCmd(t, ts, []string{"repo", "unstar", "repo1"})
	if !strings.Contains(out, "Unstarred repo1") {
		t.Errorf("expected 'Unstarred repo1' in output, got: %s", out)
	}
}

func TestRepoFork(t *testing.T) {
	ts := newTestServer(t, map[string]http.HandlerFunc{
		"POST /api/v1/repos/repo1/fork": func(w http.ResponseWriter, r *http.Request) {
			body := readBody(r)
			if !strings.Contains(body, "my-fork") {
				t.Errorf("expected 'my-fork' in body, got: %s", body)
			}
			jsonHandler(200, map[string]interface{}{"id": "fork1", "forked_from": "repo1"})(w, r)
		},
	})

	out := execCmd(t, ts, []string{"repo", "fork", "repo1", "my-fork"})
	if !strings.Contains(out, "Forked repo1 -> fork1") {
		t.Errorf("expected fork output, got: %s", out)
	}
}

// --------------- Issue commands ---------------

func TestIssueList(t *testing.T) {
	ts := newTestServer(t, map[string]http.HandlerFunc{
		"GET /api/v1/repos/myrepo/issues": jsonHandler(200, map[string]interface{}{
			"issues": []map[string]interface{}{
				{"id": "iss1", "title": "Bug report", "status": "open", "author": "alice"},
			},
			"total": 1,
		}),
	})

	out := execCmd(t, ts, []string{"issue", "list", "-r", "myrepo"})
	if !strings.Contains(out, "iss1") || !strings.Contains(out, "Bug report") {
		t.Errorf("expected issue data in output, got: %s", out)
	}
}

func TestIssueListWithStatus(t *testing.T) {
	var capturedPath string
	ts := newTestServer(t, map[string]http.HandlerFunc{
		"GET /api/v1/repos/myrepo/issues": func(w http.ResponseWriter, r *http.Request) {
			capturedPath = r.URL.String()
			jsonHandler(200, map[string]interface{}{"issues": []interface{}{}, "total": 0})(w, r)
		},
	})

	execCmd(t, ts, []string{"issue", "list", "-r", "myrepo", "--status", "closed"})
	if !strings.Contains(capturedPath, "status=closed") {
		t.Errorf("expected status=closed in path, got: %s", capturedPath)
	}
}

func TestIssueCreate(t *testing.T) {
	ts := newTestServer(t, map[string]http.HandlerFunc{
		"POST /api/v1/repos/myrepo/issues": func(w http.ResponseWriter, r *http.Request) {
			body := readBody(r)
			if !strings.Contains(body, "New bug") {
				t.Errorf("expected 'New bug' in request body, got: %s", body)
			}
			jsonHandler(200, map[string]interface{}{"id": "iss2"})(w, r)
		},
	})

	out := execCmd(t, ts, []string{"issue", "create", "-r", "myrepo", "-t", "New bug", "-b", "details here"})
	if !strings.Contains(out, "Created issue: iss2") {
		t.Errorf("expected 'Created issue: iss2', got: %s", out)
	}
}

func TestIssueCreateWithLabels(t *testing.T) {
	ts := newTestServer(t, map[string]http.HandlerFunc{
		"POST /api/v1/repos/myrepo/issues": func(w http.ResponseWriter, r *http.Request) {
			body := readBody(r)
			if !strings.Contains(body, "bug") || !strings.Contains(body, "urgent") {
				t.Errorf("expected labels in body, got: %s", body)
			}
			jsonHandler(200, map[string]interface{}{"id": "iss3"})(w, r)
		},
	})

	out := execCmd(t, ts, []string{"issue", "create", "-r", "myrepo", "-t", "Critical", "-l", "bug,urgent"})
	if !strings.Contains(out, "Created issue: iss3") {
		t.Errorf("expected 'Created issue: iss3', got: %s", out)
	}
}

func TestIssueClose(t *testing.T) {
	ts := newTestServer(t, map[string]http.HandlerFunc{
		"POST /api/v1/repos/myrepo/issues/iss1/close": jsonHandler(200, map[string]interface{}{}),
	})

	out := execCmd(t, ts, []string{"issue", "close", "iss1", "-r", "myrepo"})
	if !strings.Contains(out, "Closed issue iss1") {
		t.Errorf("expected 'Closed issue iss1', got: %s", out)
	}
}

func TestIssueComment(t *testing.T) {
	ts := newTestServer(t, map[string]http.HandlerFunc{
		"POST /api/v1/repos/myrepo/issues/iss1/comment": func(w http.ResponseWriter, r *http.Request) {
			body := readBody(r)
			if !strings.Contains(body, "great idea") {
				t.Errorf("expected comment body, got: %s", body)
			}
			jsonHandler(200, map[string]interface{}{})(w, r)
		},
	})

	out := execCmd(t, ts, []string{"issue", "comment", "iss1", "-r", "myrepo", "-b", "great idea"})
	if !strings.Contains(out, "Commented on issue iss1") {
		t.Errorf("expected 'Commented on issue iss1', got: %s", out)
	}
}

func TestIssueComments(t *testing.T) {
	ts := newTestServer(t, map[string]http.HandlerFunc{
		"GET /api/v1/repos/myrepo/issues/iss1/comments": jsonHandler(200, map[string]interface{}{
			"comments": []map[string]interface{}{
				{"id": "c1", "author": "bob", "body": "looks good", "timestamp": "2026-05-24"},
			},
			"total": 1,
		}),
	})

	out := execCmd(t, ts, []string{"issue", "comments", "iss1", "-r", "myrepo"})
	if !strings.Contains(out, "looks good") || !strings.Contains(out, "bob") {
		t.Errorf("expected comment data in output, got: %s", out)
	}
}

// --------------- PR commands ---------------

func TestPRList(t *testing.T) {
	ts := newTestServer(t, map[string]http.HandlerFunc{
		"GET /api/v1/repos/myrepo/prs": jsonHandler(200, map[string]interface{}{
			"prs": []map[string]interface{}{
				{"id": "pr1", "title": "Add feature", "status": "open", "author": "alice", "source_branch": "feat", "target_branch": "main"},
			},
			"total": 1,
		}),
	})

	out := execCmd(t, ts, []string{"pr", "list", "-r", "myrepo"})
	if !strings.Contains(out, "pr1") || !strings.Contains(out, "feat -> main") {
		t.Errorf("expected PR data in output, got: %s", out)
	}
}

func TestPRCreate(t *testing.T) {
	ts := newTestServer(t, map[string]http.HandlerFunc{
		"POST /api/v1/repos/myrepo/prs": func(w http.ResponseWriter, r *http.Request) {
			body := readBody(r)
			if !strings.Contains(body, "feat") || !strings.Contains(body, "Add feature") {
				t.Errorf("expected PR body, got: %s", body)
			}
			jsonHandler(200, map[string]interface{}{"id": "pr2"})(w, r)
		},
	})

	out := execCmd(t, ts, []string{"pr", "create", "-r", "myrepo", "-t", "Add feature", "-s", "feat", "--target", "develop"})
	if !strings.Contains(out, "Created PR: pr2") {
		t.Errorf("expected 'Created PR: pr2', got: %s", out)
	}
}

func TestPRMerge(t *testing.T) {
	ts := newTestServer(t, map[string]http.HandlerFunc{
		"POST /api/v1/repos/myrepo/prs/pr1/merge": jsonHandler(200, map[string]interface{}{}),
	})

	out := execCmd(t, ts, []string{"pr", "merge", "pr1", "-r", "myrepo"})
	if !strings.Contains(out, "Merged PR pr1") {
		t.Errorf("expected 'Merged PR pr1', got: %s", out)
	}
}

func TestPRReview(t *testing.T) {
	ts := newTestServer(t, map[string]http.HandlerFunc{
		"POST /api/v1/repos/myrepo/prs/pr1/review": func(w http.ResponseWriter, r *http.Request) {
			body := readBody(r)
			if !strings.Contains(body, "approve") {
				t.Errorf("expected verdict in body, got: %s", body)
			}
			jsonHandler(200, map[string]interface{}{})(w, r)
		},
	})

	out := execCmd(t, ts, []string{"pr", "review", "pr1", "-r", "myrepo", "-v", "approve", "-b", "LGTM"})
	if !strings.Contains(out, "Reviewed PR pr1: approve") {
		t.Errorf("expected review output, got: %s", out)
	}
}

func TestPRComments(t *testing.T) {
	ts := newTestServer(t, map[string]http.HandlerFunc{
		"GET /api/v1/repos/myrepo/prs/pr1/comments": jsonHandler(200, map[string]interface{}{
			"comments": []map[string]interface{}{
				{"id": "pc1", "author": "bob", "body": "nice work", "timestamp": "2026-05-24"},
			},
			"total": 1,
		}),
	})

	out := execCmd(t, ts, []string{"pr", "comments", "pr1", "-r", "myrepo"})
	if !strings.Contains(out, "nice work") {
		t.Errorf("expected PR comment in output, got: %s", out)
	}
}

// --------------- Task commands ---------------

func TestTaskList(t *testing.T) {
	ts := newTestServer(t, map[string]http.HandlerFunc{
		"GET /api/v1/repos/myrepo/tasks": jsonHandler(200, map[string]interface{}{
			"tasks": []map[string]interface{}{
				{"id": "t1", "title": "Fix CI", "status": "open", "claimed_by": "", "created_by": "alice"},
			},
			"total": 1,
		}),
	})

	out := execCmd(t, ts, []string{"task", "list", "-r", "myrepo"})
	if !strings.Contains(out, "t1") || !strings.Contains(out, "Fix CI") {
		t.Errorf("expected task data in output, got: %s", out)
	}
}

func TestTaskCreate(t *testing.T) {
	ts := newTestServer(t, map[string]http.HandlerFunc{
		"POST /api/v1/repos/myrepo/tasks": func(w http.ResponseWriter, r *http.Request) {
			body := readBody(r)
			if !strings.Contains(body, "Write docs") {
				t.Errorf("expected task title in body, got: %s", body)
			}
			jsonHandler(200, map[string]interface{}{"id": "t2"})(w, r)
		},
	})

	out := execCmd(t, ts, []string{"task", "create", "-r", "myrepo", "-t", "Write docs", "-d", "API docs"})
	if !strings.Contains(out, "Created task: t2") {
		t.Errorf("expected 'Created task: t2', got: %s", out)
	}
}

func TestTaskClaim(t *testing.T) {
	ts := newTestServer(t, map[string]http.HandlerFunc{
		"POST /api/v1/repos/myrepo/tasks/t1/claim": jsonHandler(200, map[string]interface{}{}),
	})

	out := execCmd(t, ts, []string{"task", "claim", "t1", "-r", "myrepo"})
	if !strings.Contains(out, "Claimed task t1") {
		t.Errorf("expected 'Claimed task t1', got: %s", out)
	}
}

func TestTaskComplete(t *testing.T) {
	ts := newTestServer(t, map[string]http.HandlerFunc{
		"POST /api/v1/repos/myrepo/tasks/t1/complete": func(w http.ResponseWriter, r *http.Request) {
			body := readBody(r)
			if !strings.Contains(body, "done") {
				t.Errorf("expected result in body, got: %s", body)
			}
			jsonHandler(200, map[string]interface{}{})(w, r)
		},
	})

	out := execCmd(t, ts, []string{"task", "complete", "t1", "-r", "myrepo", "--result", "done"})
	if !strings.Contains(out, "Completed task t1") {
		t.Errorf("expected 'Completed task t1', got: %s", out)
	}
}

// --------------- Label commands ---------------

func TestLabelList(t *testing.T) {
	ts := newTestServer(t, map[string]http.HandlerFunc{
		"GET /api/v1/repos/myrepo/labels": jsonHandler(200, map[string]interface{}{
			"labels": []map[string]interface{}{
				{"name": "bug", "color": "#ef4444"},
				{"name": "feature", "color": "#22c55e"},
			},
			"total": 2,
		}),
	})

	out := execCmd(t, ts, []string{"label", "list", "-r", "myrepo"})
	if !strings.Contains(out, "bug") || !strings.Contains(out, "#ef4444") {
		t.Errorf("expected label data in output, got: %s", out)
	}
}

func TestLabelCreate(t *testing.T) {
	ts := newTestServer(t, map[string]http.HandlerFunc{
		"POST /api/v1/repos/myrepo/labels": func(w http.ResponseWriter, r *http.Request) {
			body := readBody(r)
			if !strings.Contains(body, "priority") {
				t.Errorf("expected label name in body, got: %s", body)
			}
			jsonHandler(200, map[string]interface{}{})(w, r)
		},
	})

	out := execCmd(t, ts, []string{"label", "create", "-r", "myrepo", "-n", "priority", "-c", "#f59e0b"})
	if !strings.Contains(out, "Created label: priority") {
		t.Errorf("expected 'Created label: priority', got: %s", out)
	}
}

func TestLabelDelete(t *testing.T) {
	ts := newTestServer(t, map[string]http.HandlerFunc{
		"DELETE /api/v1/repos/myrepo/labels/old-label": func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(200)
		},
	})

	out := execCmd(t, ts, []string{"label", "delete", "-r", "myrepo", "-n", "old-label"})
	if !strings.Contains(out, "Deleted label: old-label") {
		t.Errorf("expected 'Deleted label: old-label', got: %s", out)
	}
}

// --------------- Webhook commands ---------------

func TestWebhookList(t *testing.T) {
	ts := newTestServer(t, map[string]http.HandlerFunc{
		"GET /api/v1/webhooks": jsonHandler(200, map[string]interface{}{
			"webhooks": []map[string]interface{}{
				{"id": "wh1", "url": "https://example.com/hook", "events": []string{"push", "issue"}},
			},
			"total": 1,
		}),
	})

	out := execCmd(t, ts, []string{"webhook", "list"})
	if !strings.Contains(out, "wh1") || !strings.Contains(out, "example.com") {
		t.Errorf("expected webhook data in output, got: %s", out)
	}
}

func TestWebhookRegister(t *testing.T) {
	ts := newTestServer(t, map[string]http.HandlerFunc{
		"POST /api/v1/webhooks": func(w http.ResponseWriter, r *http.Request) {
			body := readBody(r)
			if !strings.Contains(body, "https://example.com") {
				t.Errorf("expected URL in body, got: %s", body)
			}
			jsonHandler(200, map[string]interface{}{"id": "wh2"})(w, r)
		},
	})

	out := execCmd(t, ts, []string{"webhook", "register", "--url", "https://example.com", "--events", "push,issue"})
	if !strings.Contains(out, "Registered webhook: wh2") {
		t.Errorf("expected 'Registered webhook: wh2', got: %s", out)
	}
}

func TestWebhookDelete(t *testing.T) {
	ts := newTestServer(t, map[string]http.HandlerFunc{
		"DELETE /api/v1/webhooks/wh1": func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(200)
		},
	})

	out := execCmd(t, ts, []string{"webhook", "delete", "wh1"})
	if !strings.Contains(out, "Deleted webhook wh1") {
		t.Errorf("expected 'Deleted webhook wh1', got: %s", out)
	}
}

// --------------- Protection commands ---------------

func TestProtectionShow(t *testing.T) {
	ts := newTestServer(t, map[string]http.HandlerFunc{
		"GET /api/v1/repos/myrepo/protections/main": jsonHandler(200, map[string]interface{}{
			"branch":           "main",
			"protected":        true,
			"require_pr":       true,
			"require_approval": true,
			"no_force_push":    false,
		}),
	})

	out := execCmd(t, ts, []string{"protection", "show", "myrepo", "main"})
	if !strings.Contains(out, "Require PR") || !strings.Contains(out, "true") {
		t.Errorf("expected protection rules in output, got: %s", out)
	}
}

func TestProtectionShowNotProtected(t *testing.T) {
	ts := newTestServer(t, map[string]http.HandlerFunc{
		"GET /api/v1/repos/myrepo/protections/dev": jsonHandler(200, map[string]interface{}{
			"protected": false,
		}),
	})

	out := execCmd(t, ts, []string{"protection", "show", "myrepo", "dev"})
	if !strings.Contains(out, "not protected") {
		t.Errorf("expected 'not protected' in output, got: %s", out)
	}
}

func TestProtectionSet(t *testing.T) {
	ts := newTestServer(t, map[string]http.HandlerFunc{
		"PUT /api/v1/repos/myrepo/protections/main": func(w http.ResponseWriter, r *http.Request) {
			body := readBody(r)
			if !strings.Contains(body, "require_pr") {
				t.Errorf("expected protection flags in body, got: %s", body)
			}
			jsonHandler(200, map[string]interface{}{
				"require_pr":       true,
				"require_approval": false,
				"no_force_push":    true,
			})(w, r)
		},
	})

	out := execCmd(t, ts, []string{"protection", "set", "myrepo", "main", "--require-pr", "--no-force-push"})
	if !strings.Contains(out, "Protection rules set") {
		t.Errorf("expected 'Protection rules set', got: %s", out)
	}
}

func TestProtectionRemove(t *testing.T) {
	ts := newTestServer(t, map[string]http.HandlerFunc{
		"DELETE /api/v1/repos/myrepo/protections/main": func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(200)
		},
	})

	out := execCmd(t, ts, []string{"protection", "remove", "myrepo", "main"})
	if !strings.Contains(out, "Protection rules removed") {
		t.Errorf("expected 'Protection rules removed', got: %s", out)
	}
}

// --------------- Agent commands ---------------

func TestAgentList(t *testing.T) {
	ts := newTestServer(t, map[string]http.HandlerFunc{
		"GET /api/v1/agents": jsonHandler(200, map[string]interface{}{
			"agents": []map[string]interface{}{
				{"did": "did:key:abc", "trust_score": 0.95, "repos": 3, "commits": 42},
			},
			"total": 1,
		}),
	})

	out := execCmd(t, ts, []string{"agent", "list"})
	if !strings.Contains(out, "did:key:abc") || !strings.Contains(out, "trust=0.95") {
		t.Errorf("expected agent data in output, got: %s", out)
	}
}

func TestAgentShow(t *testing.T) {
	ts := newTestServer(t, map[string]http.HandlerFunc{
		"GET /api/v1/agents/did%3Akey%3Aabc": jsonHandler(200, map[string]interface{}{
			"did":         "did:key:abc",
			"trust_score": 0.95,
		}),
	})

	out := execCmd(t, ts, []string{"agent", "show", "did:key:abc"})
	if !strings.Contains(out, "did:key:abc") {
		t.Errorf("expected agent DID in output, got: %s", out)
	}
}

func TestAgentGenerate(t *testing.T) {
	ts := newTestServer(t, map[string]http.HandlerFunc{
		"POST /api/v1/agents/generate-did": jsonHandler(200, map[string]interface{}{
			"did": "did:key:new123",
		}),
	})

	out := execCmd(t, ts, []string{"agent", "generate"})
	if !strings.Contains(out, "did:key:new123") {
		t.Errorf("expected new DID in output, got: %s", out)
	}
}

func TestAgentDelegate(t *testing.T) {
	ts := newTestServer(t, map[string]http.HandlerFunc{
		"POST /api/v1/agents/did%3Akey%3Atarget/delegate": func(w http.ResponseWriter, r *http.Request) {
			body := readBody(r)
			if !strings.Contains(body, `"read"`) || !strings.Contains(body, `"write"`) {
				t.Errorf("expected actions array in body, got: %s", body)
			}
			jsonHandler(200, map[string]interface{}{"token": "ucan.abc123"})(w, r)
		},
	})

	out := execCmd(t, ts, []string{"agent", "delegate", "--did", "did:key:target", "--resource", "repo:myrepo", "--actions", "read,write"})
	if !strings.Contains(out, "ucan.abc123") {
		t.Errorf("expected token in output, got: %s", out)
	}
}

func TestAgentVerify(t *testing.T) {
	ts := newTestServer(t, map[string]http.HandlerFunc{
		"POST /api/v1/agents/verify": func(w http.ResponseWriter, r *http.Request) {
			body := readBody(r)
			if !strings.Contains(body, "ucan.abc123") {
				t.Errorf("expected token in body, got: %s", body)
			}
			jsonHandler(200, map[string]interface{}{"valid": true, "issuer": "did:key:abc"})(w, r)
		},
	})

	out := execCmd(t, ts, []string{"agent", "verify", "--token", "ucan.abc123"})
	if !strings.Contains(out, "Token valid") {
		t.Errorf("expected 'Token valid' in output, got: %s", out)
	}
}

// --------------- UCAN commands ---------------

func TestUCANRevoke(t *testing.T) {
	ts := newTestServer(t, map[string]http.HandlerFunc{
		"POST /api/v1/ucan/revoke": func(w http.ResponseWriter, r *http.Request) {
			body := readBody(r)
			if !strings.Contains(body, "nonce123") {
				t.Errorf("expected nonce in body, got: %s", body)
			}
			jsonHandler(200, map[string]interface{}{})(w, r)
		},
	})

	out := execCmd(t, ts, []string{"ucan", "revoke", "nonce123"})
	if !strings.Contains(out, "Revoked UCAN with nonce: nonce123") {
		t.Errorf("expected revoke output, got: %s", out)
	}
}

func TestUCANListRevocations(t *testing.T) {
	ts := newTestServer(t, map[string]http.HandlerFunc{
		"GET /api/v1/ucan/revocations": jsonHandler(200, map[string]interface{}{
			"revocations": []map[string]interface{}{
				{"nonce": "nonce123", "revoked_at": 1716528000},
			},
			"total": 1,
		}),
	})

	out := execCmd(t, ts, []string{"ucan", "list-revocations"})
	if !strings.Contains(out, "nonce123") || !strings.Contains(out, "revoked_at=") {
		t.Errorf("expected revocation data in output, got: %s", out)
	}
}

func TestUCANListRevocationsEmpty(t *testing.T) {
	ts := newTestServer(t, map[string]http.HandlerFunc{
		"GET /api/v1/ucan/revocations": jsonHandler(200, map[string]interface{}{
			"revocations": []interface{}{},
			"total":       0,
		}),
	})

	out := execCmd(t, ts, []string{"ucan", "list-revocations"})
	if !strings.Contains(out, "No revoked UCANs") {
		t.Errorf("expected 'No revoked UCANs', got: %s", out)
	}
}

