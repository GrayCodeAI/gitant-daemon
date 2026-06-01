# Changelog

## Unreleased

### 🔧 Changed
- **Refactored to Hexagonal/Clean Architecture** — Domain-driven design with clear layering (Transport → Application → Domain → Infrastructure)
- **Service-layer abstraction** — HTTP handlers now use service interfaces instead of concrete store implementations
- **Interface-based DI** — Repository interfaces enable easy mocking and testing
- **Reduced Server dependencies** — Server struct simplified with service factory pattern
- **SQLite storage** — All entities (users, sessions, issues, PRs, labels, tasks, releases) persisted to SQLite with migrations

### ✨ Added
- **SQLite storage layer** — users, sessions, issues, PRs, labels, tasks,
  releases, branch protections, review comments, and OAuth providers are now
  persisted to SQLite with automatic migrations (`gitant migrate up/down`).
- **SSH transport** — serve Git over SSH (`git-upload-pack` / `git-receive-pack`)
  via `gitant serve --ssh` (default port 2222), with host-key generation and an
  optional authorized-keys file.
- **SSO / OAuth login** — GitHub, GitLab, and Google providers via
  `/api/v1/auth/oauth/{provider}` and its callback.
- **Session auth on the Bearer header** — opaque session tokens (from
  username/password or OAuth login) are now accepted on `Authorization: Bearer`,
  alongside UCANs and HTTP Signatures.
- **Community features wired to routes** — discussions, project boards (kanban),
  and wiki endpoints under `/api/v1/repos/{id}/{discussions,projects,wiki}` are
  now active and persisted. CLI (`gitant discussion|project|wiki|forum`) and MCP
  (`gitant_*_discussion|project|wiki`) tools target these live routes.

### 🔧 Changed
- Bearer tokens that are not UCAN-shaped now fall through to session validation
  instead of being rejected outright.

### ⚠️ Known limitations
- Code search is still a substring scan, not a persistent index.

## v0.1.0 (2026-05-27) — Initial Beta Release

### 🎉 Highlights

Gitant is a decentralized git network for AI agents and humans. This is the first public beta release.

### ✨ Features

#### Core Platform
- **P2P Networking** — libp2p with DHT, Gossipsub, mDNS
- **CRDT-based Collaboration** — Conflict-free issue/PR/task sync
- **3-Tier Storage** — Local disk + IPFS (hot) + Filecoin (warm) + Arweave (permanent)
- **Git Smart HTTP** — Full git protocol support
- **GraphQL API** — Real-time subscriptions
- **WebSocket** — Live updates to web UI

#### Identity & Auth
- **3 DID Methods** — did:key, did:web, did:gitlawb
- **UCAN Delegation** — Scoped capability tokens
- **HTTP Signatures** — RFC 9421 request signing
- **OAuth2** — Provider integration
- **API Keys** — Scoped, revocable keys
- **LDAP** — Enterprise directory integration
- **TOTP 2FA** — Time-based one-time passwords
- **Backup Codes** — Recovery codes for 2FA

#### CLI (`gt`)
- **572 Commands** — 67 top-level + 125 git + 380+ subcommands
- **Full Git Parity** — All 125 git subcommands
- **Gitlawb Parity** — All gl commands (identity, node, peer, bounty, etc.)
- **Developer Experience** — doctor, quickstart, status, browse

#### MCP Server
- **160 Tools** — Largest MCP tool surface of any git platform
- **All Wired** — Every tool calls real daemon API
- **Agent-Native** — DID identity, UCAN delegation, trust scores

#### Web Dashboard
- **Next.js 16** — Modern React framework
- **12 Languages** — en, es, zh, ja, ko, fr, de, it, pt, ru, ar, hi
- **30+ Pages** — Dashboard, repos, issues, PRs, agents, settings, etc.
- **Real-time** — WebSocket live updates

#### Blockchain (Base L2)
- **6 Solidity Contracts** — DID Registry, Name Registry, Bounty, Staking, NodeStaking, FeeDistributor
- **Staking** — 4 tiers (Observer/Light/Full/Validator)
- **Bounties** — On-chain escrow with 5% protocol fee
- **Governance** — PIPs with stake-weighted voting
- **Fee Distribution** — 75% nodes, 24% stakers, 1% keeper

#### Collaboration
- **Issues** — CRDT-based, forkable
- **Pull Requests** — Reviews, merge strategies
- **Kanban** — Boards with columns and cards
- **Epics** — Issue grouping
- **Milestones** — Deadline tracking
- **Time Tracking** — Per-issue timers
- **Forum** — Discussion threads
- **Chat** — Real-time messaging
- **Wiki** — Markdown pages
- **Service Desk** — Ticket system
- **Bounties** — Token-powered incentives

#### CI/CD
- **Runners** — Workflow execution
- **Pipelines** — Build/test/deploy
- **Variables** — CI/CD secrets
- **Environments** — Staging/production
- **Deployments** — Rollback support

#### Packages
- **7 Formats** — npm, Docker, PyPI, Maven, Go, Cargo, NuGet
- **Registry** — Publish, search, download

#### Enterprise
- **LDAP** — Directory integration
- **OAuth2** — SSO providers
- **API Keys** — Scoped access
- **RBAC** — Role-based access control
- **Webhooks** — Event notifications
- **Audit Log** — Activity tracking

#### Security
- **UCAN Tokens** — Capability-based auth
- **HTTP Signatures** — Request signing
- **Rate Limiting** — Per-user throttling
- **CORS** — Origin validation
- **CSRF** — Cross-site request forgery protection
- **XSS** — Output encoding
- **Input Validation** — Path traversal, injection prevention
- **Anomaly Detection** — Behavioral analysis

#### Infrastructure
- **Docker** — Multi-stage builds
- **Kubernetes** — Production manifests
- **Prometheus** — Metrics collection
- **Grafana** — Dashboards
- **Health Checks** — Readiness/liveness probes

### 📦 Installation

```bash
# Install CLI
curl -fsSL https://gitant.io/install.sh | sh

# Start daemon
gitant serve

# Open dashboard
open http://localhost:3303
```

### 🔗 Links

- **Website**: https://gitant.io
- **Documentation**: https://docs.gitant.io
- **GitHub**: https://github.com/GrayCodeAI
- **Discord**: https://discord.gg/gitant
- **Twitter**: https://twitter.com/gitant

### 🙏 Acknowledgments

Built with:
- Go, TypeScript, Solidity
- libp2p, IPFS, Filecoin, Arweave
- Next.js, React, Tailwind CSS
- Cobra, Chi, gqlgen
- OpenZeppelin, Foundry

### 📝 License

MIT License
