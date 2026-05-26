package observability

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

// HealthStatus represents the health status
type HealthStatus string

const (
	HealthStatusHealthy   HealthStatus = "healthy"
	HealthStatusDegraded  HealthStatus = "degraded"
	HealthStatusUnhealthy HealthStatus = "unhealthy"
)

// HealthCheck represents a health check
type HealthCheck struct {
	Name     string
	CheckFn  func(ctx context.Context) error
	Required bool
}

// HealthChecker manages health checks
type HealthChecker struct {
	mu      sync.RWMutex
	checks  []HealthCheck
	timeout time.Duration
}

// NewHealthChecker creates a new health checker
func NewHealthChecker(timeout time.Duration) *HealthChecker {
	return &HealthChecker{
		checks:  make([]HealthCheck, 0),
		timeout: timeout,
	}
}

// Register registers a health check
func (h *HealthChecker) Register(check HealthCheck) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.checks = append(h.checks, check)
}

// Check runs all health checks
func (h *HealthChecker) Check(ctx context.Context) map[string]interface{} {
	h.mu.RLock()
	defer h.mu.RUnlock()

	results := make(map[string]interface{})
	status := HealthStatusHealthy

	for _, check := range h.checks {
		checkCtx, cancel := context.WithTimeout(ctx, h.timeout)
		err := check.CheckFn(checkCtx)
		cancel()

		if err != nil {
			results[check.Name] = map[string]interface{}{
				"status": "unhealthy",
				"error":  err.Error(),
			}
			if check.Required {
				status = HealthStatusUnhealthy
			} else if status != HealthStatusUnhealthy {
				status = HealthStatusDegraded
			}
		} else {
			results[check.Name] = map[string]interface{}{
				"status": "healthy",
			}
		}
	}

	return map[string]interface{}{
		"status": string(status),
		"checks": results,
	}
}

// HealthHandler returns a handler that serves health checks
func HealthHandler(checker *HealthChecker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		result := checker.Check(r.Context())

		status := result["status"].(string)
		code := http.StatusOK
		switch status {
		case "unhealthy":
			code = http.StatusServiceUnavailable
		case "degraded":
			code = http.StatusOK
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(code)
		json.NewEncoder(w).Encode(result)
	}
}

// ReadinessHandler returns a handler that serves readiness checks
func ReadinessHandler(checker *HealthChecker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		result := checker.Check(r.Context())

		status := result["status"].(string)
		if status == "unhealthy" {
			http.Error(w, "service unavailable", http.StatusServiceUnavailable)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"ready": true,
		})
	}
}

// LivenessHandler returns a handler that serves liveness checks
func LivenessHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"alive": true,
		})
	}
}

// GracefulShutdown handles graceful shutdown
type GracefulShutdown struct {
	mu       sync.Mutex
	hooks    []func(ctx context.Context) error
	timeout  time.Duration
}

// NewGracefulShutdown creates a new graceful shutdown handler
func NewGracefulShutdown(timeout time.Duration) *GracefulShutdown {
	return &GracefulShutdown{
		hooks:   make([]func(ctx context.Context) error, 0),
		timeout: timeout,
	}
}

// Register registers a shutdown hook
func (g *GracefulShutdown) Register(hook func(ctx context.Context) error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.hooks = append(g.hooks, hook)
}

// Shutdown runs all shutdown hooks
func (g *GracefulShutdown) Shutdown(ctx context.Context) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	var firstErr error
	for _, hook := range g.hooks {
		if err := hook(ctx); err != nil && firstErr == nil {
			firstErr = err
		}
	}

	return firstErr
}

// ShutdownWithTimeout runs all shutdown hooks with a timeout
func (g *GracefulShutdown) ShutdownWithTimeout() error {
	ctx, cancel := context.WithTimeout(context.Background(), g.timeout)
	defer cancel()
	return g.Shutdown(ctx)
}
