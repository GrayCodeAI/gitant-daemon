# Gitant Product Master Plan

> Goal: make Gitant a production-grade, agent-native git platform — honest about what ships today (single-node) while building toward decentralized P2P.

## Architecture target (informed by Radicle, gitlawb, UCAN spec, Forgejo ops)

```
┌─────────────┐     MCP (stdio)      ┌──────────────┐
│ AI Agents   │◄────────────────────►│  gitant-mcp  │
└─────────────┘                      └──────┬───────┘
                                              │ REST + UCAN
┌─────────────┐     /api/daemon proxy  ┌──────▼───────┐
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
| MCP (58 tools) | **Dev-ready** — schema fixes + npm publish pending |
| P2P / multi-node sync | **Partial** — block exchange + CRDT gossip; bootstrap seeds via env |

---

## Phase A — Operational trust (THIS SPRINT) ✅ implementing

### gitant-daemon
- [x] Fix backup/restore (`identity.key`, `repos/`, `data/` subtree)
- [x] Private repo access control (read middleware + list filter)
- [x] PR merge: git fast-forward + approval parse fix + identity context fix
- [x] Live agent count in `/api/v1/status`
- [x] `RequireCapability`: allow HTTP-signature operators; UCAN agents scoped
- [x] Fix `"identity"` context key → `middleware.GetIdentity` across handlers

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
- [x] npm `bin`, `type: module`, shebang
- [x] README updated (58 tools, env vars)

### gitant-core (monorepo)
- [x] Root `docker-compose.yml` for one-command stack
- [x] This plan document

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
- [ ] npm publish (deferred — Go binary release path preferred)

### Ops
- [x] `docker-compose.prod.yml` with Caddy/nginx TLS
- [x] Documented backup schedule (`docs/BACKUP.md`)
- [x] Health checks in compose for both services
- [x] Install script (`scripts/install.sh` for get.gitant.dev)

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

### C3 — Federation (partial)
- [x] Cross-instance discovery endpoint (`GET /api/v1/federation/discover`)
- [x] Bootstrap seed nodes via `GITANT_SEED_PEERS` / `--bootstrap-peers`
- [ ] Optional IPFS pinning adapter (stub remains in `internal/ipfs/`)

### C4 — Agent economy
- Trust scores from cross-peer attestation
- Agent marketplace (Phase 6)
- TypeScript SDK for agent developers

---

## Phase D — Ecosystem (6+ months)

- Package registry
- IDE extensions (VS Code, JetBrains)
- Mobile notifications
- CI running on Gitant-hosted repos
- Security audit (crypto + UCAN chain validation)
- Filecoin/Arweave warm/cold storage (optional)

---

## Success metrics

| Metric | Phase A target | Phase C target |
|--------|----------------|----------------|
| `go test ./...` | pass | pass + multi-node integration |
| Web tests | 20+ pass | + E2E |
| MCP npm | publishable | 1k+ downloads |
| Docker one-command up | < 60s | + 3-node cluster |
| P2P peer sync | N/A | push on A → visible on B < 30s |
| Auth bypass | none on private repos | UCAN scoped per repo |

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
| `gitant-daemon` | Go binary, API, git transport | 7777 |
| `gitant-web` | Next.js UI | 3303 (dev) / 3000 (prod) |
| `gitant-mcp` | MCP server for agents | stdio |
| `gitant-core/` | Monorepo wrapper, compose, this plan | — |

---

*Last updated: 2026-05-26 — Phase C3/C4 foundation complete; marketplace pending.*
