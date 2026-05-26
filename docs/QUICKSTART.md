# Gitant developer quickstart

Get from zero to push, issues, web UI, and MCP in about 10 minutes.

**Distribution:** everything ships from **GitHub** — clone repos, [releases](https://github.com/GrayCodeAI), and install scripts. Same pattern as `gitant-cli` and `gitant-daemon`.

## What you need

| Component | Repo | Install |
|-----------|------|---------|
| **CLI** (push, issues, PRs) | [gitant-cli](https://github.com/GrayCodeAI/gitant-cli) | [install.sh](https://github.com/GrayCodeAI/gitant-cli/blob/main/scripts/install.sh) or `make build` |
| **Daemon** (API + optional P2P) | [gitant-daemon](https://github.com/GrayCodeAI/gitant-daemon) | Docker or `make build && make run` |
| **Web** (dashboard) | [gitant-web](https://github.com/GrayCodeAI/gitant-web) | clone + `make dev` |
| **MCP** (AI agents) | [gitant-mcp](https://github.com/GrayCodeAI/gitant-mcp) | clone + `make build` or [release tarball](https://github.com/GrayCodeAI/gitant-mcp/releases) |

All four talk to the same daemon URL (default `http://localhost:7777`).

## 1. Start the daemon

**Docker (recommended)**

```bash
git clone https://github.com/GrayCodeAI/gitant-daemon.git
cd gitant-daemon
docker compose up -d
curl -s http://localhost:7777/health | jq .
```

**From source**

```bash
git clone https://github.com/GrayCodeAI/gitant-daemon.git
cd gitant-daemon
make build && make run
```

**With P2P federation** (multi-node mesh):

```bash
gitant serve --p2p --port 7777
# Other nodes: gitant serve --p2p --bootstrap-peers /ip4/.../p2p/...
```

See [FEDERATION.md](./FEDERATION.md) for bootstrap seeds and WAN discovery.

## 2. Install and configure the CLI

```bash
curl -fsSL https://raw.githubusercontent.com/GrayCodeAI/gitant-cli/main/scripts/install.sh | bash
export GITANT_DAEMON_URL=http://localhost:7777
gitant doctor
```

Interactive setup:

```bash
gitant quickstart --yes
```

## 3. Authenticate (UCAN)

Write operations (push, create repo, issues) need a UCAN token.

```bash
# Generate a client DID and save a delegated token
gitant auth login
gitant auth status
```

Or set `GITANT_UCAN_TOKEN` manually after delegating on the Agents page or via:

```bash
gitant agent delegate --did YOUR_DID --resource 'repo:*' --actions read,write
```

## 4. Create a repo and push

```bash
gitant repo create my-app --description "Hello Gitant"
mkdir my-app && cd my-app
git init && echo "# my-app" > README.md
git add . && git commit -m "init"
gitant push --remote "$GITANT_DAEMON_URL" --repo my-app
gitant issue create --repo my-app --title "First issue"
gitant repo list
```

## 5. Web dashboard

```bash
git clone https://github.com/GrayCodeAI/gitant-web.git
cd gitant-web
make dev
```

Open [http://localhost:3303](http://localhost:3303). Paste your UCAN on **Agents → UCAN** for write access in the browser.

**Full stack in Docker** (daemon + web):

```bash
# From gitant-daemon with gitant-web as sibling folder
docker compose -f docker-compose.stack.yml up --build
```

## 6. MCP for AI agents

```bash
git clone https://github.com/GrayCodeAI/gitant-mcp.git
cd gitant-mcp
make build
```

Or download a [GitHub release](https://github.com/GrayCodeAI/gitant-mcp/releases) tarball (includes pre-built `dist/`).

Cursor / Claude Desktop config (use your clone path):

```json
{
  "mcpServers": {
    "gitant": {
      "command": "node",
      "args": ["/path/to/gitant-mcp/dist/index.js"],
      "env": {
        "GITANT_DAEMON_URL": "http://localhost:7777",
        "GITANT_UCAN_TOKEN": "your-delegated-ucan"
      }
    }
  }
}
```

64 tools cover repos, issues, PRs, agents, webhooks, and federation status.

## 7. Verify everything

```bash
gitant doctor          # 13 health checks against the daemon
gitant status          # daemon summary
curl -s $GITANT_DAEMON_URL/api/v1/status | jq .
```

In the web UI: **Settings → Test connection** shows daemon version, P2P peers, and federation when `--p2p` is enabled.

## Common issues

| Problem | Fix |
|---------|-----|
| `command not found: gitant` | Add install dir to `PATH` or use `./bin/gitant` |
| Push returns 401 | Run `gitant auth login` or set `GITANT_UCAN_TOKEN` |
| Web cannot connect | Start daemon; set `NEXT_PUBLIC_DAEMON_URL` |
| P2P peers = 0 | Use `--p2p` and `--bootstrap-peers` or same LAN + mDNS |

## Next steps

- [API reference](./api.md)
- [Federation & bootstrap](./FEDERATION.md)
- [Backup & restore](./BACKUP.md)
- [Roadmap / plan](../PLAN.md)
