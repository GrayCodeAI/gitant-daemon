# gitant

Decentralized Git hosting for solo developers and AI agents.

## Quick Start

### Docker (recommended)

```bash
git clone https://github.com/lakshmanpatel/gitant.git
cd gitant/gitant-daemon
docker compose up -d
```

Your node is running at `http://localhost:7777`.

### From source

Requires Go 1.25+.

```bash
go build -o gitant ./cmd/gitant/
go build -o git-remote-gitant ./cmd/git-remote-gitant/

./gitant serve
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
./gitant push --remote http://localhost:7777 --repo my-project

# Clone elsewhere
./gitant clone my-project --remote http://localhost:7777 ./my-project-clone
```

## Architecture

```
gitant-daemon (Go)
├── HTTP API (go-chi)     REST endpoints for repos, issues, PRs, files, commits
├── P2P Networking         libp2p (DHT + GossipSub)
├── Identity               DID:key (Ed25519) + HTTP Signatures (RFC 9421)
├── Storage                go-git + content-addressed blockstore
└── CRDT                   Issues and PRs with Lamport clocks

gitant-mcp (TypeScript)    MCP server for AI agent integration (24 tools)
gitant-web (Next.js)       Web frontend (13 routes)
```

## CLI Reference

| Command | Description |
|---------|-------------|
| `gitant serve` | Start the daemon |
| `gitant init` | Initialize a local repo |
| `gitant push --repo <id> --remote <url>` | Push to daemon |
| `gitant pull --repo <id> --remote <url>` | Pull from daemon |
| `gitant clone <repo-id> [dir] --remote <url>` | Clone from daemon |
| `gitant issue list --repo <id>` | List issues |
| `gitant issue create --repo <id> --title <t>` | Create issue |
| `gitant issue close --repo <id> <id>` | Close issue |
| `gitant issue comment --repo <id> <id> --body <t>` | Comment on issue |
| `gitant pr list --repo <id>` | List PRs |
| `gitant pr create --repo <id> --title <t> -s <branch>` | Create PR |
| `gitant pr merge --repo <id> <id>` | Merge PR |
| `gitant pr review --repo <id> <id> -v approve` | Review PR |

## API

All endpoints are under `/api/v1/`. See [docs/api.md](docs/api.md) for full reference.

```bash
# Health check
curl http://localhost:7777/health

# Status
curl http://localhost:7777/api/v1/status

# List repos
curl http://localhost:7777/api/v1/repos
```

## Configuration

| Env Variable | Default | Description |
|-------------|---------|-------------|
| `GITANT_PORT` | `7777` | HTTP port |
| `GITANT_DAEMON_URL` | `http://localhost:7777` | Daemon URL (for CLI) |

## Development

```bash
go test ./...
go vet ./...
go build ./...
```

## License

MIT
