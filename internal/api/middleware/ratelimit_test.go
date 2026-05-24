package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRateLimiterAllowsUnderLimit(t *testing.T) {
	rl := NewRateLimiter(10) // 10 req/min
	defer rl.cleanup.Stop()

	handler := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// First 10 requests should pass
	for i := 0; i < 10; i++ {
		req := httptest.NewRequest("GET", "/", nil)
		req.RemoteAddr = "1.2.3.4:1234"
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("request %d: expected 200, got %d", i+1, rec.Code)
		}
	}
}

func TestRateLimiterBlocksOverLimit(t *testing.T) {
	rl := NewRateLimiter(5) // 5 req/min
	defer rl.cleanup.Stop()

	handler := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Exhaust the limit
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest("GET", "/", nil)
		req.RemoteAddr = "10.0.0.1:5555"
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("request %d: expected 200, got %d", i+1, rec.Code)
		}
	}

	// Next request should be rate limited
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "10.0.0.1:5555"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", rec.Code)
	}
}

func TestRateLimiterSeparatesIPs(t *testing.T) {
	rl := NewRateLimiter(2) // 2 req/min per IP
	defer rl.cleanup.Stop()

	handler := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// IP A uses 2 requests
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest("GET", "/", nil)
		req.RemoteAddr = "1.1.1.1:1111"
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("IP A request %d: expected 200, got %d", i+1, rec.Code)
		}
	}

	// IP A should be blocked
	reqA := httptest.NewRequest("GET", "/", nil)
	reqA.RemoteAddr = "1.1.1.1:1111"
	recA := httptest.NewRecorder()
	handler.ServeHTTP(recA, reqA)
	if recA.Code != http.StatusTooManyRequests {
		t.Fatalf("IP A: expected 429, got %d", recA.Code)
	}

	// IP B should still be allowed
	reqB := httptest.NewRequest("GET", "/", nil)
	reqB.RemoteAddr = "2.2.2.2:2222"
	recB := httptest.NewRecorder()
	handler.ServeHTTP(recB, reqB)
	if recB.Code != http.StatusOK {
		t.Fatalf("IP B: expected 200, got %d", recB.Code)
	}
}

func TestRateLimiterIgnoresXForwardedFor(t *testing.T) {
	rl := NewRateLimiter(2)
	defer rl.cleanup.Stop()

	handler := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Use up limit on RemoteAddr 9.9.9.9
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest("GET", "/", nil)
		req.RemoteAddr = "9.9.9.9:9999"
		req.Header.Set("X-Forwarded-For", "3.3.3.3")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("request %d: expected 200, got %d", i+1, rec.Code)
		}
	}

	// Same X-Forwarded-For but different RemoteAddr — should succeed (XFF is ignored)
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "8.8.8.8:8888"
	req.Header.Set("X-Forwarded-For", "3.3.3.3")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 (XFF ignored, different RemoteAddr), got %d", rec.Code)
	}

	// Same RemoteAddr as original — should be blocked
	req2 := httptest.NewRequest("GET", "/", nil)
	req2.RemoteAddr = "9.9.9.9:9999"
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429 (same RemoteAddr exhausted), got %d", rec2.Code)
	}
}

func TestNewRateLimiterCreatesCleanup(t *testing.T) {
	rl := NewRateLimiter(100)
	defer rl.cleanup.Stop()

	if rl.rate != 100.0/60.0 {
		t.Errorf("expected rate %f, got %f", 100.0/60.0, rl.rate)
	}
	if rl.burst != 100 {
		t.Errorf("expected burst 100, got %f", rl.burst)
	}
	if rl.visitors == nil {
		t.Fatal("visitors map should not be nil")
	}
}
