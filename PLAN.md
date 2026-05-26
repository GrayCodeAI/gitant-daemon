# Gitant Product Master Plan

> Goal: make Gitant a production-grade, agent-native git platform — honest about what ships today (single-node) while building toward decentralized P2P.

## Architecture target (informed by Radicle, gitlawb, UCAN spec, Forgejo ops)

```
┌─────────────┐     MCP (stdio)      ┌──────────────┐
│ AI Agents   │◄────────────────────►│  gitant-mcp  │
└─────────────┘                      └──────┬───────┘
                                               │ REST + UCAN
┌─────────────┐     git / gitant CLI   ┌──────────────┐
│  Developers │◄────────────────────►│  gitant-cli  │
└─────────────┘                      └──────┬───────┘
       │                                    │ REST + UCAN
       │     /api/daemon proxy              │
       ▼                                    ▼
┌─────────────┐                      ┌──────────────┐
│  Developers │◄────────────────────►│  gitant-web  │
└─────────────┘                      └──────┬───────┘
                                               │
                                        ┌──────▼───────┐
                                        │ gitant-daemon│
                                        │  HTTP + Git  │
                                        └──────┬───────┘
                                               │
                     Phase 2+ ─────────────────┼──────────────► libp2p DHT + GossipSub
                                               │                IPFS/block replication
                                        ┌──────▼───────┐
                                        │ Local stores │
                                        │ repos + CRDT │
                                        └──────────────┘
```

---

## Current state (honest)

| Layer | Status |
|-------|--------|
| Single-node Git host | **Operational** — push/pull/clone, smart HTTP, packfile |
| Collaboration (issues/PRs/tasks) | **Operational** — CRDT metadata on disk |
| DID + UCAN + HTTP Signatures | **Implemented** — enforcement gaps being closed |
| Web dashboard | **Operational locally** — most daemon APIs wired |
| MCP (64 tools) | **Dev-ready** — GitHub clone + release tarballs |
| P2P / multi-node sync | **Partial** — block exchange + CRDT gossip; bootstrap seeds via env |
| Multi-user auth | **NEW** — SQLite backend, RBAC, session-based auth |
| Inline code review | **NEW** — PR review comments with threads |
| CI/CD runner | **NEW** — Workflow engine for `.gitant/workflows/*.yml` |
| Package registry | **NEW** — npm/Docker package management |
| Wiki system | **NEW** — Markdown wiki per repository |
| Notifications | **NEW** — In-app notifications with unread counts |
| Git LFS | **NEW** — Large file storage support |
| WebSocket | **NEW** — Real-time updates for issues, PRs, pushes |
| Security | **NEW** — CSRF, input validation, security headers |
| Monitoring | **NEW** — Prometheus metrics, health checks |
| Caching | **NEW** — In-memory cache with TTL |
| Import/Export | **NEW** — Repository migration support |

---

## Phase A — Operational trust ✅ complete

### gitant-daemon
- [x] Fix backup/restore (`identity.key`, `repos/`, `data/` subtree)
- [x] Private repo access control (read middleware + list filter)
- [x] PR merge: git fast-forward + approval parse fix + identity context fix
- [x] Live agent count in `/api/v1/status`
- [x] `RequireCapability`: allow HTTP-signature operators; UCAN agents scoped
- [x] Fix `"identity"` context key → `middleware.GetIdentity` across handlers
- [x] `docker-compose.stack.yml` for daemon + web stack

### gitant-web
- [x] UCAN token settings on `/agents`
- [x] Auto-save token after delegation
- [x] Agent registry list UI
- [x] Client error reporting via Next API route (no CORS)
- [x] Docker Compose (daemon + web) + web Dockerfile
- [x] `output: "standalone"` for production builds

### gitant-mcp
- [x] Issue/PR IDs as strings (match daemon)
- [x] `push_code` includes `objects[]`
- [x] `get_daemon_status` tool
- [x] Node `bin` entry, ESM, shebang
- [x] README updated (64 tools, env vars)

### gitant-cli
- [x] Standalone developer CLI repo (`GrayCodeAI/gitant-cli`)
- [x] v0.1.0 release + install script

---

## Phase B — Production single-node ✅ complete

### Daemon
- [x] Real merge commits when branches diverged (not just FF)
- [x] Smart HTTP ref deletion in receive-pack
- [x] Wire `AgentRegistry.Record()` on authenticated requests
- [x] Server integration tests **with** auth middleware
- [x] Enforce UCAN per-route: `repo:{id}` read/write capabilities
- [x] Fix fork: inherit visibility rules, block private fork without auth
- [x] Issue/PR list filters (`status`, `labels` query params)

### Web
- [x] Repo stars UI (`starRepo`, `getStarCount`)
- [x] Label management pages
- [x] Release detail route
- [x] Settings: daemon URL display, connection test
- [x] Metrics dashboard (`/metrics` Prometheus → simple charts)
- [x] E2E tests (Playwright CI job)

### MCP
- [x] `push_packfile` tool (CLI parity)
- [x] Pagination params on list tools
- [x] List filters aligned with daemon (`status`, `labels`)
- [x] MCP tool handler integration tests
- [x] GitHub release workflow (`.github/workflows/release.yml` — tarball on tag)

### Ops
- [x] `docker-compose.prod.yml` with Caddy/nginx TLS
- [x] Documented backup schedule (`docs/BACKUP.md`)
- [x] Health checks in compose for both services
- [x] Install scripts (`gitant-cli/scripts/install.sh`, `gitant-daemon/scripts/install.sh`)

---

## Phase C — Decentralization (major — matches Phase 1 roadmap claims)

Reference: gitlawb (libp2p DHT + GossipSub per repo), Radicle (git-native issues), UCAN delegation chains.

### C1 — Network bootstrap ✅
- [x] Start libp2p host from `gitant serve` (`--p2p`)
- [x] mDNS for LAN; DHT for WAN peer discovery
- [x] Config: `--p2p`, `--bootstrap-peers`, listen addrs
- [x] Status API: real peer count, connected multiaddrs

### C2 — Data replication ✅
- [x] GossipSub topics per repo: `gitant/repo/{id}/events`
- [x] DHT provide repo head on push
- [x] Replicate git objects on push (`/gitant/block/1.0.0` + DHT announce)
- [x] CRDT op broadcast: issue/PR Lamport ops merged across peers
- [x] Conflict resolution policy documented + tested (`docs/CRDT_SYNC.md`)

### C3 — Federation ✅
- [x] Cross-instance discovery endpoint (`GET /api/v1/federation/discover`)
- [x] Bootstrap seed nodes via `GITANT_SEED_PEERS`, embedded JSON, `--bootstrap-peers`
- [x] Optional IPFS warm pinning adapter (`--ipfs-pin`, in-process CID store)

### C4 — Agent economy (partial)
- [x] Trust scores from cross-peer attestation (gossip + `POST /agents/{did}/attest`)
- [x] TypeScript SDK (`gitant-mcp/sdk`, `@gitant/sdk`)
- [ ] Agent marketplace (Phase 6)

---

## Phase D — Production Ready for 10-Dev Team ✅ complete

### D1 — Database Layer
- [x] SQLite database backend (`internal/store/sqlite/`)
- [x] Migration system with versioned migrations
- [x] Store interfaces for pluggable backends
- [x] User, Session, Issue, PR, Label, Task, Release, Protection, ReviewComment stores

### D2 — Multi-User Authentication
- [x] User registration and login
- [x] Session-based authentication (JWT tokens)
- [x] RBAC with 5 roles: owner, admin, maintainer, developer, viewer
- [x] Password hashing with bcrypt
- [x] Auth middleware for session validation

### D3 — Inline Code Review
- [x] Review comment store with SQLite backend
- [x] Create, list, resolve, delete comments
- [x] File path and line number tracking
- [x] Comment threading with parent_id

### D4 — CI/CD Runner
- [x] Workflow engine (`internal/runner/`)
- [x] YAML workflow parser
- [x] Job scheduler and execution
- [x] Log streaming
- [x] API endpoints for runs

### D5 — Package Registry
- [x] Package registry (`internal/packages/`)
- [x] Publish, get, list, search, delete packages
- [x] Version management
- [x] API endpoints

### D6 — Wiki System
- [x] Wiki per repository (`internal/wiki/`)
- [x] Markdown page CRUD
- [x] Page search
- [x] API endpoints

### D7 — Notifications
- [x] Notification manager (`internal/notifications/`)
- [x] Create, list, mark as read
- [x] Unread count
- [x] API endpoints

### D8 — Git LFS
- [x] LFS store (`internal/lfs/`)
- [x] Upload, download, verify objects
- [x] Batch operations
- [x] API endpoints

### D9 — Performance
- [x] In-memory cache with TTL (`internal/cache/`)
- [x] Automatic cleanup of expired items
- [x] Thread-safe operations

### D10 — Security
- [x] Input validation (`internal/security/validation.go`)
- [x] CSRF protection
- [x] Security headers middleware
- [x] CORS middleware
- [x] Request size limiting
- [x] Content type validation

### D11 — Monitoring
- [x] Prometheus metrics (`internal/observability/metrics.go`)
- [x] HTTP, DB, cache, auth, WebSocket metrics
- [x] Structured logging middleware
- [x] Request ID tracking
- [x] Error/panic recovery

### D12 — High Availability
- [x] Health checker (`internal/observability/health.go`)
- [x] Liveness probe (`/live`)
- [x] Readiness probe (`/ready`)
- [x] Health check (`/health`)
- [x] Graceful shutdown with hooks

### D13 — Real-time Updates
- [x] WebSocket hub (`internal/websocket/`)
- [x] Client management
- [x] Repo-scoped broadcasting
- [x] User-scoped notifications
- [x] Event types for issues, PRs, pushes

### D14 — API Completeness
- [x] Import/export handlers
- [x] Batch operations
- [x] OpenAPI specification
- [x] Activity feed

### D15 — Documentation
- [x] Deployment guide (`docs/DEPLOYMENT.md`)
- [x] Configuration reference
- [x] TLS/HTTPS setup
- [x] Backup/restore procedures
- [x] Troubleshooting guide

---

## Phase E — Future (Post v1.0)

- [ ] Agent marketplace
- [ ] IDE extensions (VS Code, JetBrains)
- [ ] Mobile app
- [ ] GitHub/GitLab import
- [ ] Discussions/Q&A
- [ ] Projects (Kanban)
- [ ] SAML/SSO
- [ ] Audit log
- [ ] PostgreSQL support
- [ ] Redis caching
- [ ] Kubernetes deployment
- [ ] Multi-node clustering

---

## Success metrics

| Metric | Target | Status |
|--------|--------|--------|
| `go test ./...` | pass | ✅ |
| SQLite backend | working | ✅ |
| Multi-user auth | working | ✅ |
| Inline code review | working | ✅ |
| CI/CD runner | working | ✅ |
| Package registry | working | ✅ |
| Wiki system | working | ✅ |
| Notifications | working | ✅ |
| Git LFS | working | ✅ |
| WebSocket | working | ✅ |
| Security hardening | working | ✅ |
| Monitoring | working | ✅ |
| Health checks | working | ✅ |
| Import/Export | working | ✅ |
| Batch operations | working | ✅ |
| OpenAPI spec | working | ✅ |
| Deployment docs | complete | ✅ |

---

## Research references

- **UCAN:** https://ucan.xyz — capability delegation for agents
- **HTTP Signatures:** RFC 9421 — request signing (already in daemon)
- **Radicle:** peer-to-peer forge, git-native issues/patches
- **gitlawb:** libp2p + IPFS + UCAN architecture reference
- **Forgejo/Gitea:** Docker compose, Postgres, Caddy TLS patterns for self-hosting
- **freenet-git:** phased decentralization (single-writer → multi-writer ACL)

---

## Repo map

| Repo | Purpose | Port |
|------|---------|------|
| `gitant-cli` | Developer CLI (`gitant`, `git-remote-gitant`) | — |
| `gitant-daemon` | Server (`gitant serve`), API, git transport | 7777 |
| `gitant-web` | Next.js UI | 3303 (dev) / 3000 (prod) |
| `gitant-mcp` | MCP server for agents | stdio |

Local dev: clone all four repos into one folder (e.g. `gitant-core/`) — no wrapper repo required.

**Distribution:** GitHub only (clone, releases, install scripts). Public package registry deferred until post-launch.

---

## New API Endpoints (40+)

### Authentication
```
POST   /api/v1/auth/register     - Register new user
POST   /api/v1/auth/login        - Login
POST   /api/v1/auth/logout       - Logout
GET    /api/v1/auth/profile      - Get current user profile
```

### Users
```
GET    /api/v1/users             - List all users
GET    /api/v1/users/:id         - Get user by ID
```

### Review Comments
```
GET    /api/v1/repos/:id/prs/:prId/review    - List review comments
POST   /api/v1/repos/:id/prs/:prId/review    - Create review comment
POST   /api/v1/review-comments/:id/resolve   - Resolve comment
DELETE /api/v1/review-comments/:id            - Delete comment
```

### Actions (CI/CD)
```
GET    /api/v1/actions/runs      - List workflow runs
GET    /api/v1/actions/runs/:id  - Get workflow run
```

### Packages
```
GET    /api/v1/packages          - List packages
GET    /api/v1/packages/:name    - Get package
GET    /api/v1/packages/:name/:version - Get package version
POST   /api/v1/packages          - Publish package
DELETE /api/v1/packages/:name    - Delete package
```

### Wiki
```
GET    /api/v1/repos/:id/wiki/pages          - List wiki pages
GET    /api/v1/repos/:id/wiki/pages/:slug    - Get wiki page
POST   /api/v1/repos/:id/wiki/pages          - Create wiki page
PUT    /api/v1/repos/:id/wiki/pages/:slug    - Update wiki page
DELETE /api/v1/repos/:id/wiki/pages/:slug    - Delete wiki page
GET    /api/v1/repos/:id/wiki/search?q=      - Search wiki
```

### Notifications
```
GET    /api/v1/notifications              - List notifications
PUT    /api/v1/notifications/:id/read     - Mark as read
PUT    /api/v1/notifications/read-all     - Mark all as read
GET    /api/v1/notifications/unread-count - Get unread count
```

### LFS
```
POST   /api/v1/repos/:id/lfs/objects/batch   - Batch request
GET    /api/v1/repos/:id/lfs/objects/:oid     - Download object
PUT    /api/v1/repos/:id/lfs/objects/:oid     - Upload object
POST   /api/v1/repos/:id/lfs/objects/:oid/verify - Verify object
```

### Import/Export
```
POST   /api/v1/import              - Import repository
POST   /api/v1/export              - Export repository
POST   /api/v1/import/github       - Import from GitHub
POST   /api/v1/import/gitlab       - Import from GitLab
```

### Batch
```
POST   /api/v1/batch               - Execute batch operations
```

### System
```
GET    /health                     - Health check
GET    /ready                      - Readiness probe
GET    /live                       - Liveness probe
GET    /metrics                    - Prometheus metrics
GET    /api/v1/openapi.json        - OpenAPI specification
GET    /ws                         - WebSocket connection
```

---

*Last updated: 2026-05-26 — All phases complete. Ready for 10-dev team.*
