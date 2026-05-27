package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/lakshmanpatel/gitant/internal/api/handlers"
	authMiddleware "github.com/lakshmanpatel/gitant/internal/api/middleware"
	"github.com/lakshmanpatel/gitant/internal/crdt"
	"github.com/lakshmanpatel/gitant/internal/identity"
	"github.com/lakshmanpatel/gitant/internal/network"
	"github.com/lakshmanpatel/gitant/internal/runner"
	"github.com/lakshmanpatel/gitant/internal/storage"
	"github.com/lakshmanpatel/gitant/internal/store"
	ws "github.com/lakshmanpatel/gitant/internal/websocket"
	"github.com/lakshmanpatel/gitant/internal/webhooks"
)

// Prometheus metrics.
var (
	httpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gitant_http_requests_total",
			Help: "Total HTTP requests by method, path, and status.",
		},
		[]string{"method", "path", "status"},
	)
	httpRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "gitant_http_request_duration_seconds",
			Help:    "HTTP request duration in seconds.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)
)

func init() {
	prometheus.MustRegister(httpRequestsTotal, httpRequestDuration)
}

type Server struct {
	router      *chi.Mux
	httpServer  *http.Server
	port        int
	identity    *identity.Identity
	repos       *storage.RepositoryRegistry
	issues      *crdt.IssueStore
	prs         *crdt.PullRequestStore
	blockstore  *storage.Blockstore
	agents      *handlers.AgentRegistry
	webhooks    *webhooks.Manager
	labels      *crdt.LabelStore
	tasks       *crdt.TaskStore
	releases    *crdt.ReleaseStore
	protection  *storage.ProtectionStore
	revocations *identity.RevocationStore
	rateLimiter *authMiddleware.RateLimiter
	corsOrigins []string
	startTime   time.Time
	network     *network.Node
	sync        *network.SyncCoordinator
	pinner      network.ObjectPinner
	authService *store.AuthService
	reviewStore store.ReviewCommentStore
	runner      *runner.Runner
	wsHub       *ws.Hub
}

func NewServer(port int, id *identity.Identity, repos *storage.RepositoryRegistry, issues *crdt.IssueStore, prs *crdt.PullRequestStore, blockstore *storage.Blockstore, labels *crdt.LabelStore, tasks *crdt.TaskStore, releases *crdt.ReleaseStore, protection *storage.ProtectionStore, webhookMgr *webhooks.Manager, revocations *identity.RevocationStore, dataDir string, corsOrigins []string) *Server {
	s := &Server{
		router:      chi.NewRouter(),
		port:        port,
		identity:    id,
		repos:       repos,
		issues:      issues,
		prs:         prs,
		blockstore:  blockstore,
		agents:      handlers.NewAgentRegistry(dataDir),
		webhooks:    webhookMgr,
		labels:      labels,
		tasks:       tasks,
		releases:    releases,
		protection:  protection,
		revocations: revocations,
		rateLimiter: authMiddleware.NewRateLimiter(100), // 100 req/min
		corsOrigins: corsOrigins,
		startTime:   time.Now(),
		runner:      runner.NewRunner(dataDir),
		wsHub:       ws.NewHub(),
	}

	s.setupMiddleware()
	s.setupRoutes()

	// Start WebSocket hub
	go s.wsHub.Run()

	// Load persisted agent registry
	if err := s.agents.Load(); err != nil {
		slog.Warn("failed to load agent registry", "error", err)
	}

	return s
}

// SetNetwork attaches the libp2p node and wires federated replication.
func (s *Server) SetNetwork(node *network.Node, pinner network.ObjectPinner) {
	s.network = node
	s.pinner = pinner
	if node == nil {
		return
	}

	s.sync = network.NewSyncCoordinator(
		node,
		newRepoObjectStore(s.repos),
		newCRDTSyncStore(s.issues, s.prs),
		newAgentTrustStore(s.agents),
		pinner,
	)

	if s.webhooks == nil {
		return
	}

	s.webhooks.SetEventHook(func(event webhooks.Event) {
		ctx := context.Background()

		fedEvent := network.FederatedEvent{
			Type:      string(event.Type),
			Repo:      event.Repo,
			Timestamp: event.Timestamp,
			Data:      event.Data,
		}
		if err := node.PublishRepoEvent(event.Repo, fedEvent); err != nil {
			slog.Warn("failed to publish federated event", "type", event.Type, "repo", event.Repo, "error", err)
		}

		switch event.Type {
		case webhooks.EventPush:
			hashes := network.ParseObjectHashes(event.Data)
			s.sync.AnnouncePushObjects(ctx, event.Repo, hashes)
			for _, head := range network.ParseRefHeads(event.Data) {
				node.ProvideRepoHead(ctx, event.Repo, head)
			}
		case webhooks.EventIssueCreated, webhooks.EventIssueClosed, webhooks.EventIssueCommented:
			issueID, _ := event.Data["issue_id"].(string)
			if issueID == "" {
				return
			}
			issue, err := s.issues.Get(event.Repo, issueID)
			if err != nil {
				return
			}
			if err := s.sync.PublishIssue(event.Repo, issue); err != nil {
				slog.Warn("failed to publish issue CRDT", "repo", event.Repo, "issue", issueID, "error", err)
			}
		case webhooks.EventPROpened, webhooks.EventPRMerged, webhooks.EventPRReviewed:
			prID, _ := event.Data["pr_id"].(string)
			if prID == "" {
				return
			}
			pr, err := s.prs.Get(event.Repo, prID)
			if err != nil {
				return
			}
			if err := s.sync.PublishPR(event.Repo, pr); err != nil {
				slog.Warn("failed to publish PR CRDT", "repo", event.Repo, "pr", prID, "error", err)
			}
		}
	})
}

// SetAuthService sets the auth service for user authentication
func (s *Server) SetAuthService(auth *store.AuthService) {
	s.authService = auth
}

// SetReviewStore sets the review comment store
func (s *Server) SetReviewStore(reviewStore store.ReviewCommentStore) {
	s.reviewStore = reviewStore
}

func requestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		reqID := r.Header.Get("X-Request-ID")
		if reqID == "" {
			reqID = uuid.New().String()[:8]
		}
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		next.ServeHTTP(ww, r)

		duration := time.Since(start)
		statusStr := fmt.Sprintf("%d", ww.Status())
		httpRequestsTotal.WithLabelValues(r.Method, r.URL.Path, statusStr).Inc()
		httpRequestDuration.WithLabelValues(r.Method, r.URL.Path).Observe(duration.Seconds())

		slog.Info("request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", ww.Status(),
			"duration_ms", duration.Milliseconds(),
			"request_id", reqID,
			"remote", r.RemoteAddr,
		)
	})
}

func bodySizeLimit(maxBytes int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
			next.ServeHTTP(w, r)
		})
	}
}

func (s *Server) setupMiddleware() {
	s.router.Use(requestLogger)
	s.router.Use(middleware.Recoverer)
	s.router.Use(bodySizeLimit(10 << 20))
	s.router.Use(middleware.RequestID)
	origins := s.corsOrigins
	if len(origins) == 0 {
		origins = []string{
			"http://localhost:3303",
			"http://localhost:3456",
			"http://localhost:3000",
			"https://*.gitant.dev",
		}
	}
	s.router.Use(cors.Handler(cors.Options{
		AllowedOrigins:   origins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Requested-With"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))
	s.router.Use(authMiddleware.SecurityHeaders)
	s.router.Use(authMiddleware.NewHTTPSignatureMiddleware(s.revocations, s.identity.DID))
	s.router.Use(s.recordAgentActivity)
	s.router.Use(s.rateLimiter.Middleware)
}

func (s *Server) recordAgentActivity(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if did := authMiddleware.GetIdentity(r); did != "" {
			s.agents.Record(did)
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) setupRoutes() {
	// Health, status, metrics, and API docs (public)
	s.router.Get("/health", s.handleHealth)
	s.router.Get("/api/v1/status", s.handleStatus)
	s.router.Get("/api/v1/network/peers", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlers.NetworkStatus(s.network)(w, r)
	}))
	s.router.Get("/api/v1/network/bootstrap", handlers.BootstrapPeers())
	s.router.Get("/api/v1/federation/discover", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlers.DiscoverFederation(s.network)(w, r)
	}))
	s.router.Handle("/metrics", promhttp.Handler())
	s.router.Get("/api/v1/openapi.json", handleOpenAPI)

	// Repository endpoints
	s.router.Route("/api/v1/repos", func(r chi.Router) {
		r.Get("/", handlers.ListRepos(s.repos, s.identity.DID))

		// Public read-only (private repos require auth)
		r.Group(func(r chi.Router) {
			r.Use(handlers.RequireRepoReadAccess(s.repos, s.identity.DID))
			r.Get("/{id}", handlers.GetRepo(s.repos))
			r.Get("/{id}/stars", handlers.GetStarCount(s.repos))
			r.Get("/{id}/clone", handlers.CloneRepo(s.repos))
			r.Get("/{id}/refs", handlers.ListRefs(s.repos))
			r.Get("/{id}/objects/{hash}", handlers.GetObject(s.repos))
			r.Get("/{id}/info/refs", handlers.InfoRefs(s.repos))
			r.Get("/{id}/issues", handlers.ListIssues(s.issues))
			r.Get("/{id}/issues/{issueId}", handlers.GetIssue(s.issues))
			r.Get("/{id}/issues/{issueId}/comments", handlers.ListIssueComments(s.issues))
			r.Get("/{id}/prs", handlers.ListPRs(s.prs))
			r.Get("/{id}/prs/{prId}", handlers.GetPR(s.prs))
			r.Get("/{id}/prs/{prId}/comments", handlers.ListPRComments(s.prs))
			r.Get("/{id}/files", handlers.ListFiles(s.repos))
			r.Get("/{id}/files/{path}", handlers.GetFile(s.repos))
			r.Get("/{id}/search", handlers.SearchCode(s.repos))
			r.Get("/{id}/commits", handlers.GetCommitLog(s.repos))
			r.Get("/{id}/diff", handlers.DiffCommits(s.repos))
			r.Get("/{id}/commits/{hash}/parents", handlers.DiffCommitAllParents(s.repos))
			r.Get("/{id}/labels", handlers.ListLabels(s.labels))
			r.Get("/{id}/protections", handlers.ListProtections(s.protection))
			r.Get("/{id}/protections/{branch}", handlers.GetProtection(s.protection))
			r.Get("/{id}/tasks", handlers.ListTasks(s.tasks))
			r.Get("/{id}/releases", handlers.ListReleases(s.releases))
			r.Get("/{id}/releases/{releaseId}", handlers.GetRelease(s.releases))
		})

		// Authenticated mutating endpoints (repo creation — no repo id yet)
		r.Group(func(r chi.Router) {
			r.Use(authMiddleware.RequireIdentity)
			r.Post("/", handlers.CreateRepo(s.repos, s.webhooks))
		})

		// Repo-scoped mutating endpoints (UCAN write capability enforced)
		r.Group(func(r chi.Router) {
			r.Use(authMiddleware.RequireIdentity)
			r.Use(handlers.RequireRepoReadAccess(s.repos, s.identity.DID))
			r.Use(authMiddleware.RequireRepoWriteCapability("id"))
			r.Delete("/{id}", handlers.DeleteRepo(s.repos, s.webhooks))
			r.Post("/{id}/fork", handlers.ForkRepo(s.repos, s.webhooks, s.identity.DID))
			r.Post("/{id}/star", handlers.StarRepo(s.repos))
			r.Post("/{id}/unstar", handlers.UnstarRepo(s.repos))
			r.Post("/{id}/push", handlers.PushObjects(s.repos, s.protection, s.webhooks))
			r.Post("/{id}/push-packfile", handlers.PushPackfile(s.repos, s.protection, s.webhooks))
			r.Post("/{id}/git-upload-pack", handlers.GitUploadPack(s.repos))
			r.Post("/{id}/git-receive-pack", handlers.GitReceivePack(s.repos, s.protection, s.webhooks))
			r.Post("/{id}/issues", handlers.CreateIssue(s.issues, s.webhooks))
			r.Post("/{id}/issues/{issueId}/comment", handlers.CommentIssue(s.issues, s.webhooks))
			r.Post("/{id}/issues/{issueId}/close", handlers.CloseIssue(s.issues, s.webhooks))
			r.Post("/{id}/prs", handlers.OpenPR(s.prs, s.webhooks))
			r.Post("/{id}/prs/{prId}/review", handlers.ReviewPR(s.prs, s.webhooks))
			r.Post("/{id}/prs/{prId}/merge", handlers.MergePR(s.prs, s.repos, s.protection, s.webhooks))
			r.Post("/{id}/branches", handlers.CreateBranch(s.repos))
			r.Post("/{id}/labels", handlers.CreateLabel(s.labels))
			r.Delete("/{id}/labels/{name}", handlers.DeleteLabel(s.labels))
			r.Put("/{id}/protections/{branch}", handlers.SetProtection(s.protection))
			r.Delete("/{id}/protections/{branch}", handlers.RemoveProtection(s.protection))
			r.Post("/{id}/tasks", handlers.CreateTask(s.tasks))
			r.Post("/{id}/tasks/{taskId}/claim", handlers.ClaimTask(s.tasks))
			r.Post("/{id}/tasks/{taskId}/complete", handlers.CompleteTask(s.tasks))
			r.Post("/{id}/releases", handlers.CreateRelease(s.releases, s.webhooks))
			r.Delete("/{id}/releases/{releaseId}", handlers.DeleteRelease(s.releases, s.webhooks))
		})
	})

	// Activity feed (public)
	activityFeed := handlers.NewActivityFeed(s.issues, s.prs, s.tasks, s.releases)
	s.router.Get("/api/v1/activity", activityFeed.GetActivity)

	// Agent endpoints
	s.router.Route("/api/v1/agents", func(r chi.Router) {
		r.Get("/", handlers.ListAgents(s.agents))
		r.Get("/resolve/{did}", handlers.ResolveDID())
		r.Get("/{did}", handlers.GetAgent(s.agents))
		r.Post("/generate-did", handlers.GenerateDID()) // public — bootstrapping
		r.Group(func(r chi.Router) {
			r.Use(authMiddleware.RequireIdentity)
			r.Post("/verify", handlers.VerifyUCAN())
			r.Post("/{did}/delegate", handlers.DelegateCapability(s.identity))
			r.Post("/{did}/attest", handlers.AttestAgent(s.agents, func(targetDID string, score float64, reason string) error {
				if s.sync == nil {
					return nil
				}
				return s.sync.PublishAttestation(targetDID, score, reason)
			}))
		})
	})

	// Webhook endpoints
	s.router.Route("/api/v1/webhooks", func(r chi.Router) {
		r.Get("/", handlers.ListWebhooks(s.webhooks))
		r.Group(func(r chi.Router) {
			r.Use(authMiddleware.RequireIdentity)
			r.Post("/", handlers.RegisterWebhook(s.webhooks))
			r.Delete("/{id}", handlers.DeleteWebhook(s.webhooks))
		})
	})

	// UCAN endpoints
	s.router.Route("/api/v1/ucan", func(r chi.Router) {
		r.Get("/revocations", handlers.ListRevocations(s.revocations))
		r.Group(func(r chi.Router) {
			r.Use(authMiddleware.RequireIdentity)
			r.Post("/revoke", handlers.RevokeUCAN(s.revocations))
		})
	})

	// Auth endpoints
	if s.authService != nil {
		authHandler := handlers.NewAuthHandler(s.authService)
		s.router.Post("/api/v1/auth/register", authHandler.Register)
		s.router.Post("/api/v1/auth/login", authHandler.Login)
		s.router.Post("/api/v1/auth/logout", authHandler.Logout)

		s.router.Group(func(r chi.Router) {
			r.Use(authMiddleware.SessionAuthMiddleware(s.authService))
			r.Get("/api/v1/auth/profile", authHandler.GetProfile)
		})

		userHandler := handlers.NewUserHandler(s.authService.Users)
		s.router.Get("/api/v1/users", userHandler.ListUsers)
		s.router.Get("/api/v1/users/{id}", userHandler.GetUser)
	}

	// Review comment endpoints
	if s.reviewStore != nil {
		reviewHandler := handlers.NewReviewHandler(s.reviewStore)
		s.router.Route("/api/v1/repos/{id}/prs/{prId}/review", func(r chi.Router) {
			r.Get("/", reviewHandler.ListComments)
			r.Group(func(r chi.Router) {
				r.Use(authMiddleware.RequireIdentity)
				r.Post("/", reviewHandler.CreateComment)
			})
		})
		s.router.Post("/api/v1/review-comments/{commentId}/resolve", reviewHandler.ResolveComment)
		s.router.Delete("/api/v1/review-comments/{commentId}", reviewHandler.DeleteComment)
	}

	// Actions/CI endpoints
	actionsHandler := handlers.NewActionsHandler(s.runner)
	s.router.Get("/api/v1/actions/runs", actionsHandler.ListRuns)
	s.router.Get("/api/v1/actions/runs/{runId}", actionsHandler.GetRun)

	// Import/Export endpoints
	importHandler := handlers.NewImportHandler(s.repos, s.issues, s.prs, s.webhooks)
	s.router.Post("/api/v1/import", importHandler.Import)
	s.router.Post("/api/v1/export", importHandler.Export)
	s.router.Post("/api/v1/import/github", importHandler.GitHubImport)
	s.router.Post("/api/v1/import/gitlab", importHandler.GitLabImport)

	// Batch operations
	batchHandler := handlers.NewBatchHandler(s.issues, s.prs, s.webhooks)
	s.router.Post("/api/v1/batch", batchHandler.Execute)

	// OpenAPI spec
	s.router.Get("/api/v1/openapi.json", handlers.HandleOpenAPI)

	// WebSocket endpoint
	if s.authService != nil {
		s.router.Get("/ws", func(w http.ResponseWriter, r *http.Request) {
			user := authMiddleware.GetUser(r)
			userID := ""
			if user != nil {
				userID = user.ID
			}
			ws.HandleWebSocket(s.wsHub, userID)(w, r)
		})
	}

	// Package endpoints
	// (placeholder - requires package registry initialization)

	// Wiki endpoints
	// (placeholder - requires wiki initialization)

	// Notification endpoints
	// (placeholder - requires notification manager initialization)

	// LFS endpoints
	// (placeholder - requires LFS store initialization)

	// Extended endpoints (packages, wiki, notifications, bounties, governance, etc.)
	// These are wired when the extended handler is initialized
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	checks := map[string]string{
		"identity": "ok",
		"storage":  "ok",
	}
	status := "healthy"
	code := http.StatusOK

	if s.identity == nil {
		checks["identity"] = "missing"
		status = "degraded"
		code = http.StatusServiceUnavailable
	}

	if s.repos == nil {
		checks["storage"] = "missing"
		status = "degraded"
		code = http.StatusServiceUnavailable
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": status,
		"checks": checks,
	})
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	peerCount := 0
	status := map[string]interface{}{
		"version":  Version,
		"peers":    peerCount,
		"repos":    len(s.repos.List()),
		"agents":   len(s.agents.List()),
		"uptime":   time.Since(s.startTime).String(),
		"identity": s.identity.DID,
	}

	if s.network != nil {
		peerCount = s.network.PeerCount()
		status["peers"] = peerCount
		status["p2p"] = map[string]interface{}{
			"enabled": true,
			"peer_id": s.network.Host.ID().String(),
			"addrs":   s.network.AdvertisedAddrs(),
		}
	} else {
		status["p2p"] = map[string]interface{}{
			"enabled": false,
		}
	}

	if counter, ok := s.pinner.(interface{ PinCount() int }); ok {
		status["ipfs_pins"] = counter.PinCount()
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// Start starts the HTTP(S) server. If tlsCert and tlsKey are non-empty, TLS is used.
func (s *Server) Start(tlsCert, tlsKey string) error {
	addr := fmt.Sprintf(":%d", s.port)
	s.httpServer = &http.Server{
		Addr:         addr,
		Handler:      s.router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	if tlsCert != "" && tlsKey != "" {
		slog.Info("gitant daemon listening (TLS)", "addr", addr)
		if err := s.httpServer.ListenAndServeTLS(tlsCert, tlsKey); err != nil && err != http.ErrServerClosed {
			return fmt.Errorf("listen error: %w", err)
		}
	} else {
		slog.Info("gitant daemon listening", "addr", addr, "tls", false)
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			return fmt.Errorf("listen error: %w", err)
		}
	}
	return nil
}

// Shutdown gracefully stops the HTTP server and persists all stores to disk.
func (s *Server) Shutdown(ctx context.Context) error {
	slog.Info("shutting down gitant daemon")

	// Stop accepting new connections; wait for in-flight requests to finish.
	if s.httpServer != nil {
		if err := s.httpServer.Shutdown(ctx); err != nil {
			slog.Error("HTTP shutdown error", "error", err)
		}
	}

	// Persist all stores.
	var firstErr error
	save := func(name string, fn func() error) {
		if err := fn(); err != nil {
			slog.Error("error saving store", "store", name, "error", err)
			if firstErr == nil {
				firstErr = err
			}
		}
	}
	save("issues", s.issues.Save)
	save("prs", s.prs.Save)
	save("blockstore", s.blockstore.Save)
	save("labels", s.labels.Save)
	save("tasks", s.tasks.Save)
	save("releases", s.releases.Save)
	save("protections", s.protection.Save)
	save("webhooks", s.webhooks.Save)
	save("revocations", s.revocations.Save)
	save("agents", s.agents.Save)

	if s.network != nil {
		if err := s.network.Close(); err != nil {
			slog.Error("P2P shutdown error", "error", err)
			if firstErr == nil {
				firstErr = err
			}
		}
	}

	slog.Info("shutdown complete")
	return firstErr
}
