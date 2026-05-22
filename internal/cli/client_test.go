package cli

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewClientDefaultURL(t *testing.T) {
	// Clear env
	t.Setenv("GITANT_DAEMON_URL", "")

	c := NewClient("")
	if c.BaseURL != "http://localhost:7777" {
		t.Fatalf("expected http://localhost:7777, got %s", c.BaseURL)
	}
}

func TestNewClientFromEnv(t *testing.T) {
	t.Setenv("GITANT_DAEMON_URL", "http://custom:9999")

	c := NewClient("")
	if c.BaseURL != "http://custom:9999" {
		t.Fatalf("expected http://custom:9999, got %s", c.BaseURL)
	}
}

func TestNewClientExplicit(t *testing.T) {
	c := NewClient("http://explicit:1234")
	if c.BaseURL != "http://explicit:1234" {
		t.Fatalf("expected http://explicit:1234, got %s", c.BaseURL)
	}
}

func TestClientGet(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Fatalf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/test" {
			t.Fatalf("expected /api/v1/test, got %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(map[string]string{"result": "ok"})
	}))
	defer server.Close()

	c := NewClient(server.URL)
	var result map[string]string
	err := c.Get("/api/v1/test", &result)
	if err != nil {
		t.Fatal(err)
	}
	if result["result"] != "ok" {
		t.Fatalf("expected ok, got %s", result["result"])
	}
}

func TestClientGetError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer server.Close()

	c := NewClient(server.URL)
	err := c.Get("/nonexistent", nil)
	if err == nil {
		t.Fatal("expected error for 404")
	}
}

func TestClientPost(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		var body map[string]string
		json.NewDecoder(r.Body).Decode(&body)
		if body["name"] != "test" {
			t.Fatalf("expected name=test, got %s", body["name"])
		}
		json.NewEncoder(w).Encode(map[string]bool{"created": true})
	}))
	defer server.Close()

	c := NewClient(server.URL)
	var result map[string]bool
	err := c.Post("/api/v1/repos", map[string]string{"name": "test"}, &result)
	if err != nil {
		t.Fatal(err)
	}
	if !result["created"] {
		t.Fatal("expected created=true")
	}
}

func TestClientDelete(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Fatalf("expected DELETE, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	c := NewClient(server.URL)
	err := c.Delete("/api/v1/repos/test")
	if err != nil {
		t.Fatal(err)
	}
}
