package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestServerCORSOriginsFromEnvironmentAllowOnlyConfiguredOrigins(t *testing.T) {
	t.Setenv("GITANT_CORS_ORIGINS", "https://app.example.test")
	s := setupSmartHTTPRouteServer(t)

	allowed := executeStatusRequestWithOrigin(s, "https://app.example.test")
	if got := allowed.Header().Get("Access-Control-Allow-Origin"); got != "https://app.example.test" {
		t.Fatalf("allowed configured origin should be echoed, got %q", got)
	}
	if got := allowed.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
		t.Fatalf("credentialed CORS should remain enabled for allowed origins, got %q", got)
	}

	disallowed := executeStatusRequestWithOrigin(s, "http://localhost:3303")
	if got := disallowed.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("localhost default should not be allowed when GITANT_CORS_ORIGINS is configured, got %q", got)
	}
	if got := disallowed.Header().Get("Access-Control-Allow-Credentials"); got != "" {
		t.Fatalf("disallowed origins should not receive credentialed CORS headers, got %q", got)
	}
}

func TestServerCORSConfiguredOriginsRejectWildcardAndLocalhostEntries(t *testing.T) {
	t.Setenv("GITANT_CORS_ORIGINS", "*, http://localhost:3303, https://prod.example.test")
	s := setupSmartHTTPRouteServer(t)

	allowed := executeStatusRequestWithOrigin(s, "https://prod.example.test")
	if got := allowed.Header().Get("Access-Control-Allow-Origin"); got != "https://prod.example.test" {
		t.Fatalf("allowed production origin should be echoed, got %q", got)
	}

	wildcard := executeStatusRequestWithOrigin(s, "https://attacker.example")
	if got := wildcard.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("wildcard entries should not grant credentialed CORS, got %q", got)
	}

	localhost := executeStatusRequestWithOrigin(s, "http://localhost:3303")
	if got := localhost.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("localhost entries should be ignored in configured production CORS, got %q", got)
	}
}

func TestServerCORSUnsetKeepsLocalDevelopmentDefaults(t *testing.T) {
	t.Setenv("GITANT_CORS_ORIGINS", "")
	s := setupSmartHTTPRouteServer(t)

	response := executeStatusRequestWithOrigin(s, "http://localhost:3303")
	if got := response.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:3303" {
		t.Fatalf("local development origin should remain allowed by default, got %q", got)
	}
}

func TestServerResponsesIncludeBaselineSecurityHeaders(t *testing.T) {
	s := setupSmartHTTPRouteServer(t)
	response := executeStatusRequestWithOrigin(s, "")

	if got := response.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Fatalf("missing nosniff content-type header, got %q", got)
	}
	if got := response.Header().Get("X-Frame-Options"); got != "DENY" {
		t.Fatalf("missing frame-control header, got %q", got)
	}
	if got := response.Header().Get("Content-Security-Policy"); got == "" {
		t.Fatalf("missing Content-Security-Policy frame-ancestors baseline header")
	}
}

func executeStatusRequestWithOrigin(s *Server, origin string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/status", nil)
	if origin != "" {
		req.Header.Set("Origin", origin)
	}
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)
	return w
}
