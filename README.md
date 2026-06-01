<div align="center">

# ⭐ gitant-daemon

**Decentralized Git hosting server** — P2P networking, CRDT collaboration, AI agent support.

![License](https://img.shields.io/badge/license-MIT-blue)
[![Release](https://img.shields.io/github/v/release/GrayCodeAI/gitant-daemon)](https://github.com/GrayCodeAI/gitant-daemon/releases)
[![Go Version](https://img.shields.io/badge/go-1.26+-blue)](https://go.dev/doc/install)
[![CI](https://github.com/GrayCodeAI/gitant-daemon/actions/workflows/ci.yml/badge.svg)](https://github.com/GrayCodeAI/gitant-daemon/actions)
[![Docker](https://img.shields.io/badge/docker-available-blue)](https://github.com/GrayCodeAI/gitant-daemon/pkgs/container/gitant-daemon)

**Git + Issues + PRs + AI Agents** — all in a single server.

</div>

---

## 🌟 What is Gitant?

**Gitant** is a decentralized Git hosting platform for solo developers and AI agents.

| Component | Role |
|-----------|------|
| [`gitant-cli`](https://github.com/GrayCodeAI/gitant-cli) | **Developer CLI** — push/pull, issues, PRs, agents |
| [`gitant-daemon`](https://github.com/GrayCodeAI/gitant-daemon) | **Server** — HTTP API, git smart HTTP, optional P2P |
| [`gitant-web`](https://github.com/GrayCodeAI/gitant-web) | **Dashboard** — Browser UI (Next.js) |
| [`gitant-mcp`](https://github.com/GrayCodeAI/gitant-mcp) | **Agents** — MCP tools for AI integration |

---

## 🚀 Quick Start

### Option 1: Docker (Recommended)

```bash
git clone https://github.com/GrayCodeAI/gitant-daemon.git
cd gitant-daemon
docker compose up -d
```

Your node is running at **http://localhost:7777** 🎉

### Option 2: Pre-built Binary

```bash
# macOS (Apple Silicon)
curl -LO https://github.com/GrayCodeAI/gitant-daemon/releases/latest/download/gitant-daemon_<version>_Darwin_arm64.tar.gz
tar xzf gitant-daemon_*_Darwin_arm64.tar.gz
sudo mv gitant /usr/local/bin/

# Run
gitant serve
```

### Option 3: Go Install

```bash
go install github.com/GrayCodeAI/gitant/cmd/gitant@v0.1.0
gitant serve
```

### Option 4: From Source

```bash
git clone https://github.com/GrayCodeAI/gitant-daemon.git
cd gitant-daemon
make build
./bin/gitant serve
```

> 💡 **Note**: For CLI commands (`push`, `pull`, `issues`, `PRs`), also install [`gitant-cli`](https://github.com/GrayCodeAI/gitant-cli).

---

## 🏗️ Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                     gitant-daemon (Go)                      │
├─────────────────────────────────────────────────────────────┤
│  🌐 HTTP API (go-chi)    REST endpoints for everything     │
│  📡 P2P Networking       libp2p (DHT + GossipSub + mDNS)   │
│  🔐 Identity             DID:key + UCAN + HTTP Signatures   │
│  💾 Storage              go-git + SQLite + CRDT stores     │
│  🔍 Observability        Prometheus + structured logging    │
│  🔒 Security             Rate limiting, TLS, validation     │
└─────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────┐
│  gitant-mcp (TypeScript)    160 MCP tools for AI agents    │
│  gitant-web (Next.js)       Web dashboard & UI             │
│  gitant-cli (Go)            Developer CLI                   │
└─────────────────────────────────────────────────────────────┘
```

---

## 📖 API Reference

All endpoints under `/api/v1/`. OpenAPI spec at `/api/v1/openapi.json`.

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/health` | GET | Health check with dependencies |
| `/api/v1/status` | GET | Version, uptime, repo count |
| `/api/v1/repos` | GET | List repos (paginated) |
| `/api/v1/repos` | POST | Create repo |
| `/api/v1/repos/{id}` | GET | Get repo by ID |
| `/metrics` | GET | Prometheus metrics |

### Example: Create a repo

```bash
curl -X POST http://localhost:7777/api/v1/repos \
  -H 'Content-Type: application/json' \
  -d '{"name":"my-project","description":"Hello world"}'
```

### Example: Push code

```bash
mkdir my-project && cd my-project
git init && git add . && git commit -m "init"
gitant push --remote http://localhost:7777 --repo my-project
```

---

## 🔐 Authentication

Three auth methods supported (all via `Authorization: Bearer <token>`):

| Method | Use Case |
|--------|----------|
| **UCAN Bearer** | Agents, CLI |
| **HTTP Signatures** (RFC 9421) | Signed requests |
| **Session Token** | Web frontend, username/password login |

### UCAN Example

```bash
# Generate DID
curl -X POST http://localhost:7777/api/v1/agents/generate-did

# Delegate capabilities
curl -X POST http://localhost:7777/api/v1/agents/<did>/delegate \
  -H 'Authorization: Bearer <server-ucan>' \
  -d '{"audience":"<client-did>","resource":"repo:*","actions":["read","write"]}'
```

---

## ⚙️ Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `GITANT_PORT` | `7777` | HTTP port |
| `GITANT_CORS_ORIGINS` | `http://localhost:3303` | CORS origins |
| `GITANT_DAEMON_URL` | `http://localhost:7777` | For CLI/MCP |

### TLS / HTTPS

```bash
# With certificates
gitant serve --tls-cert cert.pem --tls-key key.pem

# Behind reverse proxy
gitant serve --port 7777
```

### SSH Transport

```bash
gitant serve --ssh --ssh-port 2222
```

Clone via SSH:

```bash
git clone ssh://git@localhost:2222/my-repo
```

---

## 💾 Storage & Migrations

State is persisted to **SQLite** with automatic migrations.

```bash
# Manual migrations
gitant migrate up     # Apply migrations
gitant migrate down   # Roll back
```

---

## 🐳 Docker Deployment

### Simple Docker Compose

```bash
docker compose up -d
```

### With nginx reverse proxy

```bash
sudo cp deploy/nginx/gitant.conf /etc/nginx/sites-available/
sudo ln -s /etc/nginx/sites-available/gitant.conf /etc/nginx/sites-enabled/
sudo nginx -t && sudo systemctl reload nginx
```

### With Caddy (auto HTTPS)

```bash
sudo cp deploy/caddy/Caddyfile /etc/caddy/
sudo systemctl reload caddy
```

---

## 🔍 Monitoring

### Health Check

```bash
curl http://localhost:7777/health
# Returns: {"status":"healthy","checks":{"identity":"ok","storage":"ok"}}
```

### Prometheus Metrics

```bash
curl http://localhost:7777/metrics
```

Metrics include:
- `gitant_http_requests_total` — request count by method/path/status
- `gitant_http_request_duration_seconds` — latency histogram
- Standard Go runtime metrics

---

## 🧪 Development

```bash
make build    # Build binary
make run      # Run server
make test     # Run tests
make lint     # Lint code
make all      # All checks
```

---

## 📚 Documentation

- **Full Setup Guide**: [QUICKSTART.md](docs/QUICKSTART.md)
- **Architecture Docs**: [PLAN.md](PLAN.md)
- **CRDT Sync**: [CRDT_SYNC.md](docs/CRDT_SYNC.md)

---

## 📋 License

**MIT** — see [LICENSE](LICENSE).

---

<div align="center">

### 🔗 Related Repos

[gitant-cli](https://github.com/GrayCodeAI/gitant-cli) •
[gitant-web](https://github.com/GrayCodeAI/gitant-web) •
[gitant-mcp](https://github.com/GrayCodeAI/gitant-mcp)

*Made with ❤️ for developers and AI agents*

</div>
