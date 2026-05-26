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
| P2P / multi-node sync | **Not wired** — libraries exist, `serve` doesn't start libp2p |

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

## Phase B — Production single-node (next 4–8 weeks)

### Daemon
- Real merge commits when branches diverged (not just FF)
- Smart HTTP ref deletion in receive-pack
- Wire `AgentRegistry.Record()` on authenticated requests
- Server integration tests **with** auth middleware
- Enforce UCAN per-route: `repo:{id}` read/write capabilities
- Fix fork: inherit visibility rules, block private fork without auth

### Web
- Repo stars UI (`starRepo`, `getStarCount`)
- Label management pages
- Release detail route
- Settings: daemon URL display, connection test
- Metrics dashboard (`/metrics` Prometheus → simple charts)
- E2E tests (Playwright + docker compose CI job)

### MCP
- `push_packfile` tool (CLI parity)
- Pagination params on list tools
- Remove bogus `status`/`labels` filters or implement in daemon
- npm publish `v0.1.0` + tag
- MCP tool handler integration tests

### Ops
- `docker-compose.prod.yml` with Caddy/nginx TLS
- Documented backup schedule (daemon backup + volume snapshots)
- Health checks in compose for both services
- Install script at `get.gitant.dev` verified against compose stack

---

## Phase C — Decentralization (major — matches Phase 1 roadmap claims)

Reference: gitlawb (libp2p DHT + GossipSub per repo), Radicle (git-native issues), UCAN delegation chains.

### C1 — Network bootstrap
- Start libp2p host from `gitant serve` (`internal/network/host.go`)
- mDNS for LAN; DHT for WAN peer discovery
- Config: `--p2p`, `--bootstrap-peers`, listen addrs
- Status API: real peer count, connected multiaddrs

### C2 — Data replication
- GossipSub topics per repo: `gitant/repo/{id}/events`
- Replicate blockstore objects on push (provide/find via DHT)
- CRDT op broadcast: issue/PR/task Lamport ops merged across peers
- Conflict resolution policy documented + tested

### C3 — Federation
- Cross-instance repo discovery endpoint
- Bootstrap seed nodes (Phase 5 roadmap)
- Optional IPFS pinning adapter (implement or remove stub in `internal/ipfs/`)

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

*Last updated: 2026-05-26 — Phase A implementation in progress.*
