# Migration Guide

## Upgrading to v0.1.1

### Breaking Changes

None. This is a backward-compatible release.

### New Features

- Added Dependabot for automated dependency updates
- Added SECURITY.md for vulnerability reporting
- Added performance benchmarks
- Increased test coverage

### Upgrade Steps

```bash
# Update CLI
curl -fsSL https://gitant.io/install.sh | sh

# Update daemon
docker pull ghcr.io/graycodeai/gitant-daemon:v0.1.1

# Update web
docker pull ghcr.io/graycodeai/gitant-web:v0.1.1
```

## Upgrading from v0.1.0 to v0.1.1

### Database

No schema changes. Existing data is compatible.

### Configuration

No configuration changes required.

### API

No breaking API changes.

## Upgrading from Pre-release to v0.1.0

### Breaking Changes

1. **CLI renamed**: `gitant` → `gt` (alias `gitant` still works)
2. **MCP tools renamed**: All tools now use `gitant_` prefix
3. **API endpoints**: Some endpoints moved to `/api/v1/` prefix

### Database Migration

```bash
# Backup existing data
cp -r ~/.gitant ~/.gitant.backup

# Run migration (if any)
gitant migrate
```

### Configuration Changes

```yaml
# Old config
daemon_url: http://localhost:7777

# New config (same, but documented)
daemon_url: http://localhost:7777
```

### API Changes

| Old Endpoint | New Endpoint |
|--------------|--------------|
| `/repos` | `/api/v1/repos` |
| `/issues` | `/api/v1/repos/{id}/issues` |
| `/prs` | `/api/v1/repos/{id}/prs` |

### MCP Tool Changes

All MCP tools now use `gitant_` prefix:

| Old Name | New Name |
|----------|----------|
| `list_repos` | `gitant_list_repos` |
| `create_issue` | `gitant_create_issue` |
| `open_pr` | `gitant_open_pr` |

## Rollback

If you need to rollback:

```bash
# Restore from backup
cp -r ~/.gitant.backup ~/.gitant

# Downgrade CLI
curl -fsSL https://gitant.io/install.sh | sh -s -- --version v0.1.0
```

## Support

If you encounter issues during migration:

1. Check the [FAQ](https://docs.gitant.io/faq)
2. Search [GitHub Issues](https://github.com/GrayCodeAI/gitant-daemon/issues)
3. Ask on [Discord](https://discord.gg/gitant)
4. Email support@gitant.io
