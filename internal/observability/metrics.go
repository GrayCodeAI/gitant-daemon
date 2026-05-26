package observability

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Metrics holds all Prometheus metrics
type Metrics struct {
	HTTPRequestsTotal   *prometheus.CounterVec
	HTTPRequestDuration *prometheus.HistogramVec
	HTTPResponseSize    *prometheus.HistogramVec
	ActiveConnections   prometheus.Gauge
	DBQueriesTotal      *prometheus.CounterVec
	DBQueryDuration     *prometheus.HistogramVec
	CacheHitsTotal      *prometheus.CounterVec
	CacheMissesTotal    *prometheus.CounterVec
	AuthAttemptsTotal   *prometheus.CounterVec
	WSConnectionsActive prometheus.Gauge
}

// NewMetrics creates and registers all metrics
func NewMetrics() *Metrics {
	m := &Metrics{
		HTTPRequestsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "gitant_http_requests_total",
				Help: "Total HTTP requests by method, path, and status",
			},
			[]string{"method", "path", "status"},
		),
		HTTPRequestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "gitant_http_request_duration_seconds",
				Help:    "HTTP request duration in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"method", "path"},
		),
		HTTPResponseSize: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "gitant_http_response_size_bytes",
				Help:    "HTTP response size in bytes",
				Buckets: []float64{100, 1000, 10000, 100000, 1000000},
			},
			[]string{"method", "path"},
		),
		ActiveConnections: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "gitant_active_connections",
				Help: "Number of active HTTP connections",
			},
		),
		DBQueriesTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "gitant_db_queries_total",
				Help: "Total database queries by operation and table",
			},
			[]string{"operation", "table"},
		),
		DBQueryDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "gitant_db_query_duration_seconds",
				Help:    "Database query duration in seconds",
				Buckets: []float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1},
			},
			[]string{"operation", "table"},
		),
		CacheHitsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "gitant_cache_hits_total",
				Help: "Total cache hits by cache name",
			},
			[]string{"cache"},
		),
		CacheMissesTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "gitant_cache_misses_total",
				Help: "Total cache misses by cache name",
			},
			[]string{"cache"},
		),
		AuthAttemptsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "gitant_auth_attempts_total",
				Help: "Total authentication attempts by method and result",
			},
			[]string{"method", "result"},
		),
		WSConnectionsActive: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "gitant_ws_connections_active",
				Help: "Number of active WebSocket connections",
			},
		),
	}

	prometheus.MustRegister(
		m.HTTPRequestsTotal,
		m.HTTPRequestDuration,
		m.HTTPResponseSize,
		m.ActiveConnections,
		m.DBQueriesTotal,
		m.DBQueryDuration,
		m.CacheHitsTotal,
		m.CacheMissesTotal,
		m.AuthAttemptsTotal,
		m.WSConnectionsActive,
	)

	return m
}

// MetricsMiddleware records HTTP metrics
func MetricsMiddleware(metrics *Metrics) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			metrics.ActiveConnections.Inc()
			defer metrics.ActiveConnections.Dec()

			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			next.ServeHTTP(ww, r)

			duration := time.Since(start)
			status := http.StatusText(ww.Status())
			path := r.URL.Path

			metrics.HTTPRequestsTotal.WithLabelValues(r.Method, path, status).Inc()
			metrics.HTTPRequestDuration.WithLabelValues(r.Method, path).Observe(duration.Seconds())
			metrics.HTTPResponseSize.WithLabelValues(r.Method, path).Observe(float64(ww.BytesWritten()))
		})
	}
}

// LoggerMiddleware adds structured logging
func LoggerMiddleware(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			reqID := r.Header.Get("X-Request-ID")
			if reqID == "" {
				reqID = generateRequestID()
			}

			ctx := context.WithValue(r.Context(), "request_id", reqID)
			r = r.WithContext(ctx)

			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			next.ServeHTTP(ww, r)

			duration := time.Since(start)

			logger.Info("request",
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Int("status", ww.Status()),
				slog.Duration("duration", duration),
				slog.Int("bytes", ww.BytesWritten()),
				slog.String("request_id", reqID),
				slog.String("remote_addr", r.RemoteAddr),
				slog.String("user_agent", r.UserAgent()),
			)
		})
	}
}

// ErrorHandler handles panics and errors gracefully
func ErrorHandler(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					logger.Error("panic recovered",
						slog.Any("error", err),
						slog.String("path", r.URL.Path),
						slog.String("method", r.Method),
					)
					http.Error(w, "internal server error", http.StatusInternalServerError)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

// RequestIDMiddleware adds a request ID to the context
func RequestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqID := r.Header.Get("X-Request-ID")
		if reqID == "" {
			reqID = generateRequestID()
		}

		w.Header().Set("X-Request-ID", reqID)
		ctx := context.WithValue(r.Context(), "request_id", reqID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetRequestID gets the request ID from the context
func GetRequestID(ctx context.Context) string {
	if reqID, ok := ctx.Value("request_id").(string); ok {
		return reqID
	}
	return ""
}

// MetricsHandler returns a handler that serves Prometheus metrics
func MetricsHandler() http.Handler {
	return promhttp.Handler()
}

func generateRequestID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}
