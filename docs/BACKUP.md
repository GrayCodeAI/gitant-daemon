# Backup and restore

Gitant stores all node state under a single data directory (default `~/.gitant`). Back up this directory regularly to recover from disk failure or accidental deletion.

## What to back up

| Path (under data dir) | Contents |
|-----------------------|----------|
| `identity.key` | Server DID private key — **required** to keep the same identity |
| `repos/` | Git repositories (objects, refs, config) |
| `data/` | CRDT stores, blockstore, webhooks, agents, protections |

The built-in CLI commands mirror this layout:

```bash
gitant backup --output /backups/gitant-$(date +%Y%m%d)
gitant restore --input /backups/gitant-20260526
```

Stop the daemon before restoring so files are not overwritten while in use.

## Recommended schedule

| Environment | Frequency | Retention | Method |
|-------------|-----------|-----------|--------|
| Development | Weekly | 4 copies | `gitant backup` to local disk |
| Production | Daily | 30 days | Automated `gitant backup` + volume snapshot |
| Production (critical) | Hourly | 24 hours | Volume snapshot only (fast RPO) |

## Docker Compose

With the stack in `docker-compose.yml`, data lives in the `gitant-data` volume:

```bash
# One-off backup while stack is running (consistent enough for dev)
docker compose exec daemon gitant backup --output /tmp/backup
docker cp $(docker compose ps -q daemon):/tmp/backup ./backup-$(date +%Y%m%d)

# Or stop stack and snapshot the volume
docker compose down
docker run --rm -v gitant-core_gitant-data:/data -v "$PWD":/backup alpine \
  tar czf /backup/gitant-data-$(date +%Y%m%d).tar.gz -C /data .
docker compose up -d
```

For production (`docker-compose.prod.yml`), run backups from a cron job on the host or a sidecar container with the data volume mounted read-only.

## Verify restores

After restore, confirm:

```bash
curl -s http://localhost:7777/health | jq .
curl -s http://localhost:7777/api/v1/status | jq .
git ls-remote http://localhost:7777/api/v1/repos/<repo>/info/refs
```

The restored node should report the same `identity` DID as before backup.

## Off-site copies

Copy backup archives to object storage (S3, GCS, B2) or another machine. Encrypt archives that contain `identity.key`.
