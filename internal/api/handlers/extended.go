package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/lakshmanpatel/gitant/internal/bounties"
	"github.com/lakshmanpatel/gitant/internal/chat"
	"github.com/lakshmanpatel/gitant/internal/deployments"
	"github.com/lakshmanpatel/gitant/internal/epics"
	"github.com/lakshmanpatel/gitant/internal/forum"
	"github.com/lakshmanpatel/gitant/internal/governance"
	"github.com/lakshmanpatel/gitant/internal/kanban"
	"github.com/lakshmanpatel/gitant/internal/notifications"
	"github.com/lakshmanpatel/gitant/internal/packages"
	"github.com/lakshmanpatel/gitant/internal/servicedesk"
	"github.com/lakshmanpatel/gitant/internal/stacked"
	"github.com/lakshmanpatel/gitant/internal/timetracking"
	"github.com/lakshmanpatel/gitant/internal/wiki"
)

// ExtendedHandler handles all extended API endpoints
type ExtendedHandler struct {
	packages    *packages.EnhancedRegistry
	wiki        *wiki.Wiki
	notifications *notifications.Manager
	deployments *deployments.Store
	kanban      *kanban.Store
	epics       *epics.Store
	bounties    *bounties.Store
	governance  *governance.Store
	forum       *forum.Store
	chat        *chat.Store
	stacked     *stacked.Store
	timetracking *timetracking.Store
	servicedesk *servicedesk.Store
}

// NewExtendedHandler creates a new extended handler
func NewExtendedHandler(
	pkg *packages.EnhancedRegistry,
	wiki *wiki.Wiki,
	notif *notifications.Manager,
	deploy *deployments.Store,
	kanban *kanban.Store,
	epics *epics.Store,
	bounties *bounties.Store,
	gov *governance.Store,
	forum *forum.Store,
	chat *chat.Store,
	stacked *stacked.Store,
	time *timetracking.Store,
	sd *servicedesk.Store,
) *ExtendedHandler {
	return &ExtendedHandler{
		packages:      pkg,
		wiki:          wiki,
		notifications: notif,
		deployments:   deploy,
		kanban:        kanban,
		epics:         epics,
		bounties:      bounties,
		governance:    gov,
		forum:         forum,
		chat:          chat,
		stacked:       stacked,
		timetracking:  time,
		servicedesk:   sd,
	}
}

// RegisterRoutes registers all extended routes
func (h *ExtendedHandler) RegisterRoutes(r chi.Router) {
	// Package routes
	r.Route("/api/v1/packages/{format}", func(r chi.Router) {
		r.Get("/", h.ListPackages)
		r.Post("/", h.PublishPackage)
		r.Get("/{name}", h.GetPackage)
		r.Delete("/{name}", h.DeletePackage)
		r.Get("/{name}/{version}", h.GetPackageVersion)
	})

	// Wiki routes
	r.Route("/api/v1/repos/{id}/wiki", func(r chi.Router) {
		r.Get("/", h.ListWikiPages)
		r.Post("/", h.CreateWikiPage)
		r.Get("/{page}", h.GetWikiPage)
		r.Put("/{page}", h.UpdateWikiPage)
		r.Delete("/{page}", h.DeleteWikiPage)
	})

	// Notification routes
	r.Route("/api/v1/notifications", func(r chi.Router) {
		r.Get("/", h.ListNotifications)
		r.Post("/{id}/read", h.MarkNotificationRead)
		r.Post("/read-all", h.MarkAllNotificationsRead)
	})

	// Deployment routes
	r.Route("/api/v1/repos/{id}/deployments", func(r chi.Router) {
		r.Get("/", h.ListDeployments)
		r.Post("/", h.CreateDeployment)
		r.Get("/{deployId}", h.GetDeployment)
		r.Post("/{deployId}/rollback", h.RollbackDeployment)
	})

	// Kanban routes
	r.Route("/api/v1/repos/{id}/kanban", func(r chi.Router) {
		r.Get("/", h.ListKanbanBoards)
		r.Post("/", h.CreateKanbanBoard)
		r.Get("/{boardId}", h.GetKanbanBoard)
	})

	// Epic routes
	r.Route("/api/v1/repos/{id}/epics", func(r chi.Router) {
		r.Get("/", h.ListEpics)
		r.Post("/", h.CreateEpic)
		r.Get("/{epicId}", h.GetEpic)
	})

	// Bounty routes
	r.Route("/api/v1/repos/{id}/bounties", func(r chi.Router) {
		r.Get("/", h.ListBounties)
		r.Post("/", h.CreateBounty)
		r.Get("/stats", h.BountyStats)
		r.Get("/{bountyId}", h.GetBounty)
		r.Post("/{bountyId}/claim", h.ClaimBounty)
		r.Post("/{bountyId}/submit", h.SubmitBounty)
		r.Post("/{bountyId}/approve", h.ApproveBounty)
		r.Post("/{bountyId}/cancel", h.CancelBounty)
	})

	// Governance routes
	r.Route("/api/v1/repos/{id}/governance", func(r chi.Router) {
		r.Get("/", h.ListProposals)
		r.Post("/", h.CreateProposal)
		r.Post("/{proposalId}/vote", h.VoteOnProposal)
	})

	// Forum routes
	r.Route("/api/v1/repos/{id}/forum", func(r chi.Router) {
		r.Get("/", h.ListForumThreads)
		r.Post("/", h.CreateForumThread)
	})

	// Chat routes
	r.Route("/api/v1/repos/{id}/chat", func(r chi.Router) {
		r.Get("/", h.ListChatMessages)
		r.Post("/", h.SendChatMessage)
	})

	// Stacked diff routes
	r.Route("/api/v1/repos/{id}/stacks", func(r chi.Router) {
		r.Get("/", h.ListStacks)
		r.Get("/{stackId}", h.GetStack)
	})

	// Time tracking routes
	r.Route("/api/v1/repos/{id}/time", func(r chi.Router) {
		r.Get("/", h.ListTimeEntries)
		r.Post("/start", h.StartTimer)
		r.Post("/stop", h.StopTimer)
	})

	// Service desk routes
	r.Route("/api/v1/repos/{id}/service-desk", func(r chi.Router) {
		r.Get("/", h.ListServiceDeskTickets)
		r.Post("/", h.CreateServiceDeskTicket)
	})

	// Milestone routes
	r.Route("/api/v1/repos/{id}/milestones", func(r chi.Router) {
		r.Get("/", h.ListMilestones)
		r.Post("/", h.CreateMilestone)
		r.Get("/{milestoneId}", h.GetMilestone)
	})

	// Secret routes
	r.Route("/api/v1/repos/{id}/secrets", func(r chi.Router) {
		r.Get("/", h.ListSecrets)
		r.Post("/", h.SetSecret)
		r.Get("/{name}", h.GetSecret)
		r.Delete("/{name}", h.DeleteSecret)
	})

	// Cert routes
	r.Route("/api/v1/repos/{id}/certs", func(r chi.Router) {
		r.Get("/", h.ListCerts)
		r.Get("/{certId}", h.GetCert)
		r.Get("/{certId}/verify", h.VerifyCert)
		r.Post("/threshold", h.SetCertThreshold)
		r.Post("/sign", h.SignCert)
	})

	// Maintainer routes
	r.Route("/api/v1/repos/{id}/maintainers", func(r chi.Router) {
		r.Get("/", h.ListMaintainers)
		r.Post("/", h.AddMaintainer)
		r.Delete("/{did}", h.RemoveMaintainer)
	})

	// Identity routes
	r.Route("/api/v1/identity", func(r chi.Router) {
		r.Get("/", h.GetIdentity)
		r.Post("/generate", h.GenerateIdentity)
		r.Get("/export", h.ExportIdentity)
		r.Post("/sign", h.SignMessage)
		r.Post("/register-did", h.RegisterDID)
		r.Get("/resolve/{did}", h.ResolveDID)
	})

	// Trust routes
	r.Route("/api/v1/agents/{did}/trust", func(r chi.Router) {
		r.Get("/", h.GetTrustScore)
		r.Post("/issue", h.IssueTrustVC)
	})
	r.Post("/api/v1/trust/verify", h.VerifyTrustVC)

	// Node routes
	r.Get("/api/v1/network/seeds", h.ListSeeds)
	r.Post("/api/v1/network/seeds", h.AddSeed)
	r.Delete("/api/v1/network/seeds/{multiaddr}", h.RemoveSeed)

	// Sync routes
	r.Post("/api/v1/sync/trigger", h.TriggerSync)
	r.Get("/api/v1/sync/status", h.GetSyncStatus)

	// IPFS routes
	r.Get("/api/v1/ipfs/pins", h.ListIPFSPins)
	r.Get("/api/v1/ipfs/{cid}", h.GetIPFSObject)

	// Name routes
	r.Route("/api/v1/names", func(r chi.Router) {
		r.Post("/register", h.RegisterName)
		r.Get("/{name}/resolve", h.ResolveName)
		r.Get("/lookup", h.LookupName)
		r.Get("/{name}/available", h.CheckNameAvailable)
	})

	// Mirror routes
	r.Route("/api/v1/mirrors", func(r chi.Router) {
		r.Get("/", h.ListMirrors)
		r.Post("/", h.MirrorRepo)
	})

	// Workspace routes
	r.Route("/api/v1/workspaces", func(r chi.Router) {
		r.Get("/", h.ListWorkspaces)
		r.Post("/", h.CreateWorkspace)
	})

	// Repo tokenize
	r.Post("/api/v1/repos/{id}/tokenize", h.TokenizeRepo)
}

// Package handlers
func (h *ExtendedHandler) ListPackages(w http.ResponseWriter, r *http.Request) {
	format := chi.URLParam(r, "format")
	pkgs := h.packages.List(packages.PackageFormat(format))
	writeJSON(w, http.StatusOK, pkgs)
}

func (h *ExtendedHandler) PublishPackage(w http.ResponseWriter, r *http.Request) {
	format := chi.URLParam(r, "format")
	var pkg packages.EnhancedPackage
	if err := json.NewDecoder(r.Body).Decode(&pkg); err != nil {
		writeJSON(w, http.StatusBadRequest, err.Error())
		return
	}
	pkg.Format = packages.PackageFormat(format)
	if err := h.packages.Publish(&pkg); err != nil {
		writeJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, pkg)
}

func (h *ExtendedHandler) GetPackage(w http.ResponseWriter, r *http.Request) {
	format := chi.URLParam(r, "format")
	name := chi.URLParam(r, "name")
	pkg, err := h.packages.Get(packages.PackageFormat(format), name)
	if err != nil {
		writeJSON(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, pkg)
}

func (h *ExtendedHandler) DeletePackage(w http.ResponseWriter, r *http.Request) {
	format := chi.URLParam(r, "format")
	name := chi.URLParam(r, "name")
	if err := h.packages.Delete(packages.PackageFormat(format), name); err != nil {
		writeJSON(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (h *ExtendedHandler) GetPackageVersion(w http.ResponseWriter, r *http.Request) {
	format := chi.URLParam(r, "format")
	name := chi.URLParam(r, "name")
	version := chi.URLParam(r, "version")
	v, err := h.packages.GetVersion(packages.PackageFormat(format), name, version)
	if err != nil {
		writeJSON(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, v)
}

// Wiki handlers
func (h *ExtendedHandler) ListWikiPages(w http.ResponseWriter, r *http.Request) {
	pages, err := h.wiki.ListPages()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"pages": pages})
}

func (h *ExtendedHandler) CreateWikiPage(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Title   string `json:"title"`
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, err.Error())
		return
	}
	page, err := h.wiki.CreatePage(req.Title, req.Content, "")
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, page)
}

func (h *ExtendedHandler) GetWikiPage(w http.ResponseWriter, r *http.Request) {
	page := chi.URLParam(r, "page")
	p, err := h.wiki.GetPage(page)
	if err != nil {
		writeJSON(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (h *ExtendedHandler) UpdateWikiPage(w http.ResponseWriter, r *http.Request) {
	page := chi.URLParam(r, "page")
	var req struct {
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, err.Error())
		return
	}
	_, err := h.wiki.UpdatePage(page, req.Content, "")
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

func (h *ExtendedHandler) DeleteWikiPage(w http.ResponseWriter, r *http.Request) {
	page := chi.URLParam(r, "page")
	if err := h.wiki.DeletePage(page); err != nil {
		writeJSON(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// Notification handlers
func (h *ExtendedHandler) ListNotifications(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	unreadOnly := r.URL.Query().Get("unread") == "true"
	notifs := h.notifications.List(userID, unreadOnly)
	writeJSON(w, http.StatusOK, map[string]interface{}{"notifications": notifs})
}

func (h *ExtendedHandler) MarkNotificationRead(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	userID := r.URL.Query().Get("user_id")
	if err := h.notifications.MarkAsRead(userID, id); err != nil {
		writeJSON(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "read"})
}

func (h *ExtendedHandler) MarkAllNotificationsRead(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	if err := h.notifications.MarkAllAsRead(userID); err != nil {
		writeJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "all_read"})
}

// Deployment handlers
func (h *ExtendedHandler) ListDeployments(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	environment := r.URL.Query().Get("environment")
	deploys := h.deployments.ListDeployments(id, environment)
	writeJSON(w, http.StatusOK, map[string]interface{}{"deployments": deploys})
}

func (h *ExtendedHandler) CreateDeployment(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req struct {
		Environment string `json:"environment"`
		Ref         string `json:"ref"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, err.Error())
		return
	}
	deploy := &deployments.Deployment{
		RepoID:      id,
		Environment: req.Environment,
		Ref:         req.Ref,
		Status:      "pending",
	}
	if err := h.deployments.CreateDeployment(deploy); err != nil {
		writeJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, deploy)
}

func (h *ExtendedHandler) GetDeployment(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	deployId := chi.URLParam(r, "deployId")
	deploy, err := h.deployments.GetDeployment(id, deployId)
	if err != nil {
		writeJSON(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, deploy)
}

func (h *ExtendedHandler) RollbackDeployment(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	deployId := chi.URLParam(r, "deployId")
	if err := h.deployments.UpdateDeploymentStatus(id, deployId, "rolled_back"); err != nil {
		writeJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "rolled_back"})
}

// Kanban handlers
func (h *ExtendedHandler) ListKanbanBoards(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	boards := h.kanban.List(id)
	writeJSON(w, http.StatusOK, map[string]interface{}{"boards": boards})
}

func (h *ExtendedHandler) CreateKanbanBoard(w http.ResponseWriter, r *http.Request) {
	var board kanban.Board
	if err := json.NewDecoder(r.Body).Decode(&board); err != nil {
		writeJSON(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.kanban.Create(&board); err != nil {
		writeJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, board)
}

func (h *ExtendedHandler) GetKanbanBoard(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	boardId := chi.URLParam(r, "boardId")
	board, err := h.kanban.Get(id, boardId)
	if err != nil {
		writeJSON(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, board)
}

// Epic handlers
func (h *ExtendedHandler) ListEpics(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	status := r.URL.Query().Get("status")
	epicsList := h.epics.List(id, status)
	writeJSON(w, http.StatusOK, map[string]interface{}{"epics": epicsList})
}

func (h *ExtendedHandler) CreateEpic(w http.ResponseWriter, r *http.Request) {
	var epic epics.Epic
	if err := json.NewDecoder(r.Body).Decode(&epic); err != nil {
		writeJSON(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.epics.Create(&epic); err != nil {
		writeJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, epic)
}

func (h *ExtendedHandler) GetEpic(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	epicId := chi.URLParam(r, "epicId")
	epic, err := h.epics.Get(id, epicId)
	if err != nil {
		writeJSON(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, epic)
}

// Bounty handlers
func (h *ExtendedHandler) ListBounties(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	bountiesList := h.bounties.ListRepo(id, "")
	writeJSON(w, http.StatusOK, map[string]interface{}{"bounties": bountiesList})
}

func (h *ExtendedHandler) CreateBounty(w http.ResponseWriter, r *http.Request) {
	var bounty bounties.Bounty
	if err := json.NewDecoder(r.Body).Decode(&bounty); err != nil {
		writeJSON(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.bounties.Create(&bounty); err != nil {
		writeJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, bounty)
}

func (h *ExtendedHandler) GetBounty(w http.ResponseWriter, r *http.Request) {
	bountyId := chi.URLParam(r, "bountyId")
	bounty, err := h.bounties.Get(bountyId)
	if err != nil {
		writeJSON(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, bounty)
}

func (h *ExtendedHandler) ClaimBounty(w http.ResponseWriter, r *http.Request) {
	bountyId := chi.URLParam(r, "bountyId")
	did := r.URL.Query().Get("did")
	if err := h.bounties.Claim(bountyId, did); err != nil {
		writeJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "claimed"})
}

func (h *ExtendedHandler) SubmitBounty(w http.ResponseWriter, r *http.Request) {
	bountyId := chi.URLParam(r, "bountyId")
	did := r.URL.Query().Get("did")
	var req struct {
		Submission string `json:"submission"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	if err := h.bounties.Submit(bountyId, did, req.Submission); err != nil {
		writeJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "submitted"})
}

func (h *ExtendedHandler) ApproveBounty(w http.ResponseWriter, r *http.Request) {
	bountyId := chi.URLParam(r, "bountyId")
	did := r.URL.Query().Get("did")
	if err := h.bounties.Approve(bountyId, did); err != nil {
		writeJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "approved"})
}

func (h *ExtendedHandler) CancelBounty(w http.ResponseWriter, r *http.Request) {
	bountyId := chi.URLParam(r, "bountyId")
	did := r.URL.Query().Get("did")
	if err := h.bounties.Cancel(bountyId, did); err != nil {
		writeJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "cancelled"})
}

func (h *ExtendedHandler) BountyStats(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	all := h.bounties.ListRepo(id, "")
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"total":   len(all),
		"repo_id": id,
	})
}

// Governance handlers
func (h *ExtendedHandler) ListProposals(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	status := r.URL.Query().Get("status")
	proposals := h.governance.List(id, status)
	writeJSON(w, http.StatusOK, map[string]interface{}{"proposals": proposals})
}

func (h *ExtendedHandler) CreateProposal(w http.ResponseWriter, r *http.Request) {
	var proposal governance.Proposal
	if err := json.NewDecoder(r.Body).Decode(&proposal); err != nil {
		writeJSON(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.governance.Create(&proposal); err != nil {
		writeJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, proposal)
}

func (h *ExtendedHandler) VoteOnProposal(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	proposalId := chi.URLParam(r, "proposalId")
	var req struct {
		Voter string `json:"voter"`
		Vote  string `json:"vote"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.governance.Vote(id, proposalId, req.Voter, req.Vote); err != nil {
		writeJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "voted"})
}

// Forum handlers
func (h *ExtendedHandler) ListForumThreads(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	category := r.URL.Query().Get("category")
	threads := h.forum.ListThreads(id, category)
	writeJSON(w, http.StatusOK, map[string]interface{}{"threads": threads})
}

func (h *ExtendedHandler) CreateForumThread(w http.ResponseWriter, r *http.Request) {
	var thread forum.Thread
	if err := json.NewDecoder(r.Body).Decode(&thread); err != nil {
		writeJSON(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.forum.CreateThread(&thread); err != nil {
		writeJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, thread)
}

// Chat handlers
func (h *ExtendedHandler) ListChatMessages(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	channel := r.URL.Query().Get("channel")
	limit := 100
	messages := h.chat.GetMessages(id, channel, limit)
	writeJSON(w, http.StatusOK, map[string]interface{}{"messages": messages})
}

func (h *ExtendedHandler) SendChatMessage(w http.ResponseWriter, r *http.Request) {
	var msg chat.Message
	if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
		writeJSON(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.chat.SendMessage(&msg); err != nil {
		writeJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, msg)
}

// Stacked diff handlers
func (h *ExtendedHandler) ListStacks(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	status := r.URL.Query().Get("status")
	stacks := h.stacked.List(id, status)
	writeJSON(w, http.StatusOK, map[string]interface{}{"stacks": stacks})
}

func (h *ExtendedHandler) GetStack(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	stackId := chi.URLParam(r, "stackId")
	stack, err := h.stacked.Get(id, stackId)
	if err != nil {
		writeJSON(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, stack)
}

// Time tracking handlers
func (h *ExtendedHandler) ListTimeEntries(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	user := r.URL.Query().Get("user")
	entries := h.timetracking.List(id, user, time.Time{}, time.Time{})
	writeJSON(w, http.StatusOK, map[string]interface{}{"entries": entries})
}

func (h *ExtendedHandler) StartTimer(w http.ResponseWriter, r *http.Request) {
	var entry timetracking.TimeEntry
	if err := json.NewDecoder(r.Body).Decode(&entry); err != nil {
		writeJSON(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.timetracking.Start(&entry); err != nil {
		writeJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, entry)
}

func (h *ExtendedHandler) StopTimer(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	entryId := r.URL.Query().Get("entry_id")
	if err := h.timetracking.Stop(id, entryId); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "stopped"})
}

// Service desk handlers
func (h *ExtendedHandler) ListServiceDeskTickets(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	status := r.URL.Query().Get("status")
	priority := r.URL.Query().Get("priority")
	tickets := h.servicedesk.List(id, status, priority)
	writeJSON(w, http.StatusOK, map[string]interface{}{"tickets": tickets})
}

func (h *ExtendedHandler) CreateServiceDeskTicket(w http.ResponseWriter, r *http.Request) {
	var ticket servicedesk.Ticket
	if err := json.NewDecoder(r.Body).Decode(&ticket); err != nil {
		writeJSON(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.servicedesk.Create(&ticket); err != nil {
		writeJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, ticket)
}

// Milestone handlers
func (h *ExtendedHandler) ListMilestones(w http.ResponseWriter, r *http.Request) {
	// Milestones use the same store as releases
	id := chi.URLParam(r, "id")
	releases := h.bounties.ListRepo(id, "") // placeholder
	writeJSON(w, http.StatusOK, map[string]interface{}{"milestones": releases})
}

func (h *ExtendedHandler) CreateMilestone(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "created"})
}

func (h *ExtendedHandler) GetMilestone(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "not_found"})
}

// Stub handlers for features that need store initialization
func (h *ExtendedHandler) ListSecrets(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]interface{}{"secrets": []interface{}{}})
}

func (h *ExtendedHandler) SetSecret(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "set"})
}

func (h *ExtendedHandler) GetSecret(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"value": ""})
}

func (h *ExtendedHandler) DeleteSecret(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (h *ExtendedHandler) ListCerts(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]interface{}{"certificates": []interface{}{}})
}

func (h *ExtendedHandler) GetCert(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "not_found"})
}

func (h *ExtendedHandler) VerifyCert(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]interface{}{"valid": true})
}

func (h *ExtendedHandler) SetCertThreshold(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "set"})
}

func (h *ExtendedHandler) SignCert(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "signed"})
}

func (h *ExtendedHandler) ListMaintainers(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]interface{}{"maintainers": []interface{}{}})
}

func (h *ExtendedHandler) AddMaintainer(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "added"})
}

func (h *ExtendedHandler) RemoveMaintainer(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "removed"})
}

func (h *ExtendedHandler) GetIdentity(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"did": "did:key:example"})
}

func (h *ExtendedHandler) GenerateIdentity(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"did": "did:key:generated", "key_path": "~/.gitant/identity.pem"})
}

func (h *ExtendedHandler) ExportIdentity(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"type": "did:web"})
}

func (h *ExtendedHandler) SignMessage(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"signature": "ed25519:..."})
}

func (h *ExtendedHandler) RegisterDID(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"tx_hash": "0x...", "did": "did:gitlawb:example"})
}

func (h *ExtendedHandler) ResolveDID(w http.ResponseWriter, r *http.Request) {
	did := chi.URLParam(r, "did")
	writeJSON(w, http.StatusOK, map[string]string{"did": did, "resolved": "true"})
}

func (h *ExtendedHandler) GetTrustScore(w http.ResponseWriter, r *http.Request) {
	did := chi.URLParam(r, "did")
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"did":   did,
		"score": 0.85,
		"tier":  "full",
		"breakdown": map[string]float64{
			"longevity": 0.2,
			"activity":  0.3,
			"vouching":  0.3,
		},
	})
}

func (h *ExtendedHandler) IssueTrustVC(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"vc": "eyJ..."})
}

func (h *ExtendedHandler) VerifyTrustVC(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]interface{}{"valid": true})
}

func (h *ExtendedHandler) ListSeeds(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]interface{}{"seeds": []interface{}{}})
}

func (h *ExtendedHandler) AddSeed(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "added"})
}

func (h *ExtendedHandler) RemoveSeed(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "removed"})
}

func (h *ExtendedHandler) TriggerSync(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]interface{}{"queued": 0})
}

func (h *ExtendedHandler) GetSyncStatus(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]interface{}{"pending": 0, "completed": 0})
}

func (h *ExtendedHandler) ListIPFSPins(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]interface{}{"pins": []interface{}{}})
}

func (h *ExtendedHandler) GetIPFSObject(w http.ResponseWriter, r *http.Request) {
	cid := chi.URLParam(r, "cid")
	writeJSON(w, http.StatusOK, map[string]string{"cid": cid, "type": "blob"})
}

func (h *ExtendedHandler) RegisterName(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"tx_hash": "0x..."})
}

func (h *ExtendedHandler) ResolveName(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	writeJSON(w, http.StatusOK, map[string]string{"name": name, "did": "did:key:example"})
}

func (h *ExtendedHandler) LookupName(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"name": "example"})
}

func (h *ExtendedHandler) CheckNameAvailable(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]interface{}{"available": true})
}

func (h *ExtendedHandler) ListMirrors(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]interface{}{"mirrors": []interface{}{}})
}

func (h *ExtendedHandler) MirrorRepo(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "mirrored"})
}

func (h *ExtendedHandler) ListWorkspaces(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]interface{}{"workspaces": []interface{}{}})
}

func (h *ExtendedHandler) CreateWorkspace(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "created"})
}

func (h *ExtendedHandler) TokenizeRepo(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"token_address": "0x...", "tx_hash": "0x..."})
}
