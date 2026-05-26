# Gitant Deployment Guide

## Quick Start

### Docker (Recommended)

```bash
docker compose up -d
```

This starts:
- Gitant daemon on port 7777
- Gitant web UI on port 3303

### Manual Installation

#### Prerequisites

- Go 1.21+
- Node.js 20+ (for web UI)
- SQLite (included)

#### Build from Source

```bash
# Clone repositories
git clone https://github.com/GrayCodeAI/gitant-daemon.git
git clone https://github.com/GrayCodeAI/gitant-web.git

# Build daemon
cd gitant-daemon
make build

# Build web UI
cd ../gitant-web
make build
```

#### Run

```bash
# Start daemon
cd gitant-daemon
./bin/gitant serve --port 7777

# Start web UI (in another terminal)
cd gitant-web
npm run start
```

## Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `GITANT_PORT` | `7777` | HTTP port |
| `GITANT_DATA_DIR` | `~/.gitant` | Data directory |
| `GITANT_DB_PATH` | `~/.gitant/gitant.db` | SQLite database path |
| `GITANT_CORS_ORIGINS` | `http://localhost:3303` | Allowed CORS origins |
| `GITANT_P2P` | `false` | Enable P2P networking |
| `GITANT_SEED_PEERS` | - | Bootstrap peer multiaddrs |

### Command Line Flags

```bash
gitant serve [flags]

Flags:
  -p, --port int           Port to listen on (default 7777)
  -d, --data-dir string    Data directory (default ~/.gitant)
  --database string        Database URL (default sqlite://~/.gitant/gitant.db)
  --tls-cert string        TLS certificate file
  --tls-key string         TLS private key file
  --p2p                    Enable P2P networking
  --p2p-listen string      P2P listen address (default "/ip4/0.0.0.0/tcp/0")
  --p2p-mdns               Enable mDNS discovery (default true)
  --bootstrap-peers        Bootstrap peer multiaddrs
```

## Database

### SQLite (Default)

The default database is SQLite, stored at `~/.gitant/gitant.db`. No additional setup required.

### PostgreSQL (Optional)

For production deployments with multiple users:

```bash
# Set database URL
export GITANT_DB_URL=postgres://user:password@localhost:5432/gitant

# Run migrations
gitant migrate up
```

## Authentication

### User Registration

```bash
# Register a new user
curl -X POST http://localhost:7777/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"username":"alice","email":"alice@example.com","password":"securepassword"}'
```

### Login

```bash
# Login
curl -X POST http://localhost:7777/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"alice","password":"securepassword"}'

# Returns: {"token":"...","user":{...}}
```

### Using Session Token

```bash
# Use token in requests
curl -H "Authorization: Bearer YOUR_TOKEN" \
  http://localhost:7777/api/v1/repos
```

## RBAC Roles

| Role | Permissions |
|------|-------------|
| `owner` | Full access to everything |
| `admin` | Manage users and settings |
| `maintainer` | Manage repositories |
| `developer` | Create and edit content |
| `viewer` | Read-only access |

## TLS/HTTPS

### With Certificates

```bash
gitant serve --tls-cert /path/to/cert.pem --tls-key /path/to/key.pem
```

### Behind Reverse Proxy (Nginx)

```nginx
server {
    listen 443 ssl;
    server_name git.example.com;

    ssl_certificate /etc/letsencrypt/live/git.example.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/git.example.com/privkey.pem;

    location / {
        proxy_pass http://localhost:7777;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    location /ws {
        proxy_pass http://localhost:7777;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
    }
}
```

## Backup & Restore

### Backup

```bash
gitant backup -o /backups
```

### Restore

```bash
gitant restore -i /backups/gitant-backup-20260526-120000
```

## Monitoring

### Health Check

```bash
curl http://localhost:7777/health
# Returns: {"status":"healthy","checks":{...}}
```

### Metrics (Prometheus)

```bash
curl http://localhost:7777/metrics
```

### Grafana Dashboard

Import `deploy/grafana/gitant-dashboard.json` into Grafana.

## CI/CD

### Workflow Files

Create `.gitant/workflows/ci.yml` in your repository:

```yaml
name: CI
on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-node@v4
        with:
          node-version: '20'
      - run: npm ci
      - run: npm test
      - run: npm run lint
```

## Troubleshooting

### Port Already in Use

```bash
# Find process using port
lsof -i :7777

# Kill process
kill -9 <PID>
```

### Database Locked

```bash
# Check for lock file
ls -la ~/.gitant/*.db*

# Remove lock if stale
rm ~/.gitant/gitant.db-wal
rm ~/.gitant/gitant.db-shm
```

### Permission Denied

```bash
# Fix data directory permissions
chmod -R 755 ~/.gitant
```

## Support

- GitHub: https://github.com/GrayCodeAI/gitant-daemon
- Issues: https://github.com/GrayCodeAI/gitant-daemon/issues
- Documentation: https://gitant.dev
