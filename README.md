# gitant

Decentralized Git hosting for solo developers and AI agents.

## Quick Start

### Docker (recommended)

```bash
git clone https://github.com/GrayCodeAI/gitant-daemon.git
cd gitant-daemon
docker compose up -d
```

Your node is running at `http://localhost:7777`.

### Install a release

**Pre-built binaries** — [GitHub Releases](https://github.com/GrayCodeAI/gitant-daemon/releases):

```bash
# macOS (Apple Silicon example)
curl -LO https://github.com/GrayCodeAI/gitant-daemon/releases/latest/download/gitant-daemon_<version>_Darwin_arm64.tar.gz
tar xzf gitant-daemon_*_Darwin_arm64.tar.gz
sudo mv gitant git-remote-gitant /usr/local/bin/
```

**Go install** (requires Go 1.26+):

```bash
go install github.com/lakshmanpatel/gitant/cmd/gitant@v0.1.0
go install github.com/lakshmanpatel/gitant/cmd/git-remote-gitant@v0.1.0
```

**Container** (published on tag push):

```bash
docker pull ghcr.io/graycodeai/gitant-daemon:latest
docker run -p 7777:7777 -v gitant-data:/home/gitant/.gitant ghcr.io/graycodeai/gitant-daemon:latest
```

Check the running version:

```bash
gitant version
curl -s http://localhost:7777/api/v1/status | jq .version
```

### From source

Requires Go 1.26+.

```bash
make build
make run
```

Or manually:

```bash
go build -o bin/gitant ./cmd/gitant/
go build -o bin/git-remote-gitant ./cmd/git-remote-gitant/

./bin/gitant serve
```

### First repo

```bash
# Create a repo
curl -X POST http://localhost:7777/api/v1/repos \
  -H 'Content-Type: application/json' \
  -d '{"name":"my-project","description":"Hello world"}'

# Init locally and push
mkdir my-project && cd my-project
git init && git add . && git commit -m "init"
./bin/gitant push --remote http://localhost:7777 --repo my-project

# Clone elsewhere
./bin/gitant clone my-project --remote http://localhost:7777 ./my-project-clone
```

## Architecture

```
gitant-daemon (Go)
├── HTTP API (go-chi)     REST endpoints for repos, issues, PRs, files, commits
├── P2P Networking         libp2p (DHT + GossipSub + mDNS)
├── Identity               DID:key (Ed25519) + UCAN tokens + HTTP Signatures (RFC 9421)
├── Storage                go-git + file-per-block content-addressed blockstore
├── CRDT                   Issues and PRs with Lamport clocks
├── Observability          slog structured logging + Prometheus /metrics
└── Security               Rate limiting, input validation, TLS support

gitant-mcp (TypeScript)    MCP server for AI agent integration (52 tools)
gitant-web (Next.js)       Web frontend (18 routes, paginated, responsive)
```

## CLI Reference

| Command | Description |
|---------|-------------|
| `gitant serve [--port P] [--data-dir D] [--tls-cert F] [--tls-key F] [--p2p] [--p2p-listen ADDR] [--p2p-mdns] [--bootstrap-peers ...]` | Start the daemon |
| `gitant init` | Initialize a local repo |
| `gitant push --repo <id> --remote <url>` | Push to daemon (packfile) |
| `gitant pull --repo <id> --remote <url>` | Pull from daemon |
| `gitant clone <repo-id> [dir] --remote <url>` | Clone from daemon |
| `gitant backup -o <dir>` | Backup data directory |
| `gitant restore -i <dir>` | Restore from backup |
| `gitant issue list --repo <id>` | List issues |
| `gitant issue create --repo <id> --title <t>` | Create issue |
| `gitant issue close --repo <id> <id>` | Close issue |
| `gitant issue comment --repo <id> <id> --body <t>` | Comment on issue |
| `gitant pr list --repo <id>` | List PRs |
| `gitant pr create --repo <id> --title <t> -s <branch>` | Create PR |
| `gitant pr merge --repo <id> <id>` | Merge PR |
| `gitant pr review --repo <id> <id> -v approve` | Review PR |
| `gitant task list --repo <id>` | List tasks |
| `gitant task create --repo <id> --title <t>` | Create task |
| `gitant task claim --repo <id> <id>` | Claim task |
| `gitant task complete --repo <id> <id>` | Complete task |

## API

All endpoints are under `/api/v1/`. OpenAPI spec available at `/api/v1/openapi.json`.

```bash
# Health check (with dependency status)
curl http://localhost:7777/health

# Status (version, uptime, repo count, identity)
curl http://localhost:7777/api/v1/status

# Prometheus metrics
curl http://localhost:7777/metrics

# OpenAPI spec
curl http://localhost:7777/api/v1/openapi.json

# List repos (paginated)
curl http://localhost:7777/api/v1/repos?offset=0&limit=20

# Search code
curl "http://localhost:7777/api/v1/repos/my-project/search?q=function"
```

## Authentication

Endpoints that modify state (POST/PUT/DELETE) require authentication via UCAN Bearer tokens:

```bash
# Generate a DID
curl -X POST http://localhost:7777/api/v1/agents/generate-did

# Delegate capabilities
curl -X POST http://localhost:7777/api/v1/agents/<did>/delegate \
  -H 'Authorization: Bearer <server-ucan>' \
  -d '{"audience":"<client-did>","resource":"repo:*","actions":["read","write"]}'

# Use the UCAN token
curl -X POST http://localhost:7777/api/v1/repos \
  -H 'Authorization: Bearer <ucan-token>' \
  -d '{"name":"my-repo"}'
```

GET endpoints are public (no auth required).

## Configuration

| Env Variable | Default | Description |
|-------------|---------|-------------|
| `GITANT_PORT` | `7777` | HTTP port |
| `GITANT_DAEMON_URL` | `http://localhost:7777` | Daemon URL (for CLI/MCP) |
| `GITANT_UCAN_TOKEN` | (none) | UCAN token (for MCP server) |
| `GITANT_CORS_ORIGINS` | `http://localhost:3303` | Comma-separated CORS origins |

### TLS

```bash
# With TLS certificates
./bin/gitant serve --tls-cert /path/to/cert.pem --tls-key /path/to/key.pem

# Behind reverse proxy (no TLS flags needed)
./bin/gitant serve --port 7777
```

## Production Deployment

### Docker Compose (simple)

```bash
docker compose up -d
```

### With nginx reverse proxy

```bash
# Copy nginx config
sudo cp deploy/nginx/gitant.conf /etc/nginx/sites-available/
sudo ln -s /etc/nginx/sites-available/gitant.conf /etc/nginx/sites-enabled/
sudo nginx -t && sudo systemctl reload nginx
```

### With Caddy (auto HTTPS)

```bash
sudo cp deploy/caddy/Caddyfile /etc/caddy/
sudo systemctl reload caddy
```

### Backup & Restore

```bash
# Create backup
./bin/gitant backup -o /backups

# Restore (won't overwrite existing files)
./bin/gitant restore -i /backups/gitant-backup-20260522-143000
```

## Monitoring

### Prometheus

The `/metrics` endpoint exports:
- `gitant_http_requests_total` — request count by method/path/status
- `gitant_http_request_duration_seconds` — latency histogram
- Standard Go runtime metrics

### Grafana

Import the dashboard from `deploy/grafana/gitant-dashboard.json` into Grafana. Panels include request rate, latency percentiles, error rate, goroutines, and memory usage.

### Health Check

```bash
curl http://localhost:7777/health
# Returns: {"status":"healthy","checks":{"identity":"ok","storage":"ok"}}
```

Returns 503 with `{"status":"degraded"}` if dependencies are unhealthy.

## Development

```bash
# Build
make build
./bin/gitant version

# Run tests with race detector
make test

# Lint
make lint

# All checks
make all

# Validate release config (local snapshot, no publish)
make release
```

## License

MIT
