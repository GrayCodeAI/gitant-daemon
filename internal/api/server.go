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
)

type Server struct {
	router     *chi.Mux
	port       int
	identity   *identity.Identity
	repos      *storage.RepositoryRegistry
	issues     *crdt.IssueStore
	prs        *crdt.PullRequestStore
	blockstore *storage.Blockstore
}

func NewServer(port int, id *identity.Identity, repos *storage.RepositoryRegistry, issues *crdt.IssueStore, prs *crdt.PullRequestStore, blockstore *storage.Blockstore) *Server {
	s := &Server{
		router:     chi.NewRouter(),
		port:       port,
		identity:   id,
		repos:      repos,
		issues:     issues,
		prs:        prs,
		blockstore: blockstore,
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
		r.Post("/", handlers.CreateRepo(s.repos))
		r.Get("/", handlers.ListRepos(s.repos))
		r.Get("/{id}", handlers.GetRepo(s.repos))
		r.Delete("/{id}", handlers.DeleteRepo(s.repos))

		// Push endpoint
		r.Post("/{id}/push", handlers.PushObjects(s.repos))
		r.Get("/{id}/clone", handlers.CloneRepo(s.repos))

		// Refs
		r.Get("/{id}/refs", handlers.ListRefs(s.repos))

		// Issues
		r.Post("/{id}/issues", handlers.CreateIssue(s.issues))
		r.Get("/{id}/issues", handlers.ListIssues(s.issues))
		r.Get("/{id}/issues/{issueId}", handlers.GetIssue(s.issues))
		r.Post("/{id}/issues/{issueId}/comment", handlers.CommentIssue(s.issues))
		r.Post("/{id}/issues/{issueId}/close", handlers.CloseIssue(s.issues))

		// Pull Requests
		r.Post("/{id}/prs", handlers.OpenPR(s.prs))
		r.Get("/{id}/prs", handlers.ListPRs(s.prs))
		r.Get("/{id}/prs/{prId}", handlers.GetPR(s.prs))
		r.Post("/{id}/prs/{prId}/review", handlers.ReviewPR(s.prs))
		r.Post("/{id}/prs/{prId}/merge", handlers.MergePR(s.prs))

		// Branches
		r.Post("/{id}/branches", handlers.CreateBranch(s.repos))

		// Files
		r.Get("/{id}/files", handlers.ListFiles(s.repos))
		r.Get("/{id}/files/{path}", handlers.GetFile(s.repos))
		r.Get("/{id}/search", handlers.SearchCode(s.repos))

		// Commits
		r.Get("/{id}/commits", handlers.GetCommitLog(s.repos))
		r.Get("/{id}/diff", handlers.DiffCommits(s.repos))
	})

	// Agent endpoints
	s.router.Route("/api/v1/agents", func(r chi.Router) {
		r.Get("/", handlers.ListAgents())
		r.Get("/{did}", handlers.GetAgent())
		r.Post("/{did}/delegate", handlers.DelegateCapability(s.identity))
		r.Post("/generate-did", handlers.GenerateDID())
		r.Get("/resolve/{did}", handlers.ResolveDID())
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
