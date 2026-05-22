package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/lakshmanpatel/gitant/internal/api/handlers"
	authMiddleware "github.com/lakshmanpatel/gitant/internal/api/middleware"
	"github.com/lakshmanpatel/gitant/internal/crdt"
	"github.com/lakshmanpatel/gitant/internal/identity"
	"github.com/lakshmanpatel/gitant/internal/storage"
	"github.com/lakshmanpatel/gitant/internal/webhooks"
)

type Server struct {
	router     *chi.Mux
	port       int
	identity   *identity.Identity
	repos      *storage.RepositoryRegistry
	issues     *crdt.IssueStore
	prs        *crdt.PullRequestStore
	blockstore *storage.Blockstore
	agents     *handlers.AgentRegistry
	webhooks   *webhooks.Manager
	labels     *crdt.LabelStore
	tasks      *crdt.TaskStore
	protection *storage.ProtectionStore
}

func NewServer(port int, id *identity.Identity, repos *storage.RepositoryRegistry, issues *crdt.IssueStore, prs *crdt.PullRequestStore, blockstore *storage.Blockstore, labels *crdt.LabelStore, tasks *crdt.TaskStore, protection *storage.ProtectionStore, webhookMgr *webhooks.Manager) *Server {
	s := &Server{
		router:     chi.NewRouter(),
		port:       port,
		identity:   id,
		repos:      repos,
		issues:     issues,
		prs:        prs,
		blockstore: blockstore,
		agents:     handlers.NewAgentRegistry(),
		webhooks:   webhookMgr,
		labels:     labels,
		tasks:      tasks,
		protection: protection,
	}

	s.setupMiddleware()
	s.setupRoutes()

	return s
}

func (s *Server) setupMiddleware() {
	s.router.Use(middleware.Logger)
	s.router.Use(middleware.Recoverer)
	s.router.Use(middleware.RequestID)
	s.router.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:3303", "https://*.gitant.dev"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Requested-With"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))
	s.router.Use(authMiddleware.HTTPSignatureMiddleware)
}

func (s *Server) setupRoutes() {
	// Health and status
	s.router.Get("/health", s.handleHealth)
	s.router.Get("/api/v1/status", s.handleStatus)

	// Repository endpoints
	s.router.Route("/api/v1/repos", func(r chi.Router) {
		r.Post("/", handlers.CreateRepo(s.repos, s.webhooks))
		r.Get("/", handlers.ListRepos(s.repos))
		r.Get("/{id}", handlers.GetRepo(s.repos))
		r.Delete("/{id}", handlers.DeleteRepo(s.repos, s.webhooks))
		r.Post("/{id}/star", handlers.StarRepo(s.repos))
		r.Post("/{id}/unstar", handlers.UnstarRepo(s.repos))
		r.Get("/{id}/stars", handlers.GetStarCount(s.repos))

		// Push endpoint
		r.Post("/{id}/push", handlers.PushObjects(s.repos, s.protection))
		r.Get("/{id}/clone", handlers.CloneRepo(s.repos))

		// Refs
		r.Get("/{id}/refs", handlers.ListRefs(s.repos))

		// Objects
		r.Get("/{id}/objects/{hash}", handlers.GetObject(s.repos))

		// Git smart-HTTP
		r.Get("/{id}/info/refs", handlers.InfoRefs(s.repos))
		r.Post("/{id}/git-upload-pack", handlers.GitUploadPack(s.repos))
		r.Post("/{id}/git-receive-pack", handlers.GitReceivePack(s.repos, s.protection))

		// Issues
		r.Post("/{id}/issues", handlers.CreateIssue(s.issues, s.webhooks))
		r.Get("/{id}/issues", handlers.ListIssues(s.issues))
		r.Get("/{id}/issues/{issueId}", handlers.GetIssue(s.issues))
		r.Post("/{id}/issues/{issueId}/comment", handlers.CommentIssue(s.issues, s.webhooks))
		r.Post("/{id}/issues/{issueId}/close", handlers.CloseIssue(s.issues, s.webhooks))
		r.Get("/{id}/issues/{issueId}/comments", handlers.ListIssueComments(s.issues))

		// Pull Requests
		r.Post("/{id}/prs", handlers.OpenPR(s.prs, s.webhooks))
		r.Get("/{id}/prs", handlers.ListPRs(s.prs))
		r.Get("/{id}/prs/{prId}", handlers.GetPR(s.prs))
		r.Post("/{id}/prs/{prId}/review", handlers.ReviewPR(s.prs, s.webhooks))
		r.Post("/{id}/prs/{prId}/merge", handlers.MergePR(s.prs, s.webhooks))
		r.Get("/{id}/prs/{prId}/comments", handlers.ListPRComments(s.prs))

		// Branches
		r.Post("/{id}/branches", handlers.CreateBranch(s.repos))

		// Files
		r.Get("/{id}/files", handlers.ListFiles(s.repos))
		r.Get("/{id}/files/{path}", handlers.GetFile(s.repos))
		r.Get("/{id}/search", handlers.SearchCode(s.repos))

		// Commits
		r.Get("/{id}/commits", handlers.GetCommitLog(s.repos))
		r.Get("/{id}/diff", handlers.DiffCommits(s.repos))

		// Labels
		r.Get("/{id}/labels", handlers.ListLabels(s.labels))
		r.Post("/{id}/labels", handlers.CreateLabel(s.labels))
		r.Delete("/{id}/labels/{name}", handlers.DeleteLabel(s.labels))

		// Branch protection
		r.Get("/{id}/protections", handlers.ListProtections(s.protection))
		r.Get("/{id}/protections/{branch}", handlers.GetProtection(s.protection))
		r.Put("/{id}/protections/{branch}", handlers.SetProtection(s.protection))
		r.Delete("/{id}/protections/{branch}", handlers.RemoveProtection(s.protection))

		// Tasks
		r.Get("/{id}/tasks", handlers.ListTasks(s.tasks))
		r.Post("/{id}/tasks", handlers.CreateTask(s.tasks))
		r.Post("/{id}/tasks/{taskId}/claim", handlers.ClaimTask(s.tasks))
		r.Post("/{id}/tasks/{taskId}/complete", handlers.CompleteTask(s.tasks))
	})

	// Activity feed
	s.router.Get("/api/v1/activity", handlers.GetActivity(s.issues, s.prs, s.tasks))

	// Agent endpoints
	s.router.Route("/api/v1/agents", func(r chi.Router) {
		r.Get("/", handlers.ListAgents(s.agents))
		r.Post("/generate-did", handlers.GenerateDID())
		r.Post("/verify", handlers.VerifyUCAN())
		r.Get("/resolve/{did}", handlers.ResolveDID())
		r.Get("/{did}", handlers.GetAgent(s.agents))
		r.Post("/{did}/delegate", handlers.DelegateCapability(s.identity))
	})

	// Webhook endpoints
	s.router.Route("/api/v1/webhooks", func(r chi.Router) {
		r.Get("/", handlers.ListWebhooks(s.webhooks))
		r.Post("/", handlers.RegisterWebhook(s.webhooks))
		r.Delete("/{id}", handlers.DeleteWebhook(s.webhooks))
	})
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"version": "0.1.0",
		"peers":   0,
		"repos":   len(s.repos.List()),
		"agents":  1,
		"uptime":  0,
		"identity": s.identity.DID,
	})
}

func (s *Server) Start() error {
	addr := fmt.Sprintf(":%d", s.port)
	fmt.Printf("gitant daemon listening on %s\n", addr)
	return http.ListenAndServe(addr, s.router)
}
