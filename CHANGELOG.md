# Changelog

All notable changes to `gitant-daemon` will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Security
- Request body size limits (10MB API, 50MB git operations)
- Input validation on all mutating endpoints (repo names, issue/PR fields, branch names)
- Auth error messages sanitized — no internal details leaked to clients
- Webhook raw secret removed from HTTP headers (HMAC signature only)
- Rate limiting middleware (100 req/min per IP)
- CORS configurability via environment variable

### Fixed
- Race condition in CRDT IssueStore.Save() and PullRequestStore.Save() — deep-copy under lock
- Weak ID generation — switched from math/rand to crypto/rand
- Branch protection enforcement — force-push detection now actually blocks non-fast-forward pushes
- Releases not saved on shutdown — ReleaseStore now persisted in shutdown sequence
- CLI HTTP client — added 30s timeout (was unlimited)
- Unchecked JSON decode error in tasks handler

### Added
- Structured logging with slog (JSON format, request IDs, method/path/status/duration)
- Enhanced health endpoint with dependency status checks
- Real status endpoint (actual uptime, version from ldflags)
- Request logging middleware with request ID correlation
- Makefile with build, test, lint, docker, install targets
- .golangci.yml linter configuration
- GitHub Actions CI with golangci-lint, govulncheck, coverage, Docker smoke test

### Changed
- Dockerfile: non-root user (gitant:1000), healthcheck, fixed Go image version
- docker-compose.yml: healthcheck, fixed volume path

## [0.1.0] - 2026-05-22

### Added
- Initial release with 31 CLI commands
- REST API with 40+ endpoints
- CRDT-based issue/PR store
- P2P networking with libp2p
- IPFS integration for distributed storage
- UCAN-based authentication
- Webhook system with HMAC-SHA256 signatures
- Smart HTTP git protocol support
- Agent registry and DID management
- Branch protection rules
- Star system, labels, tasks, releases
