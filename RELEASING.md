# Releasing gitant-daemon

This repo uses [GoReleaser](https://goreleaser.com/) to publish binaries, GitHub Releases, and container images.

## Maintainer checklist

1. Ensure `main` is green (CI runs `go test`, Docker smoke test, and a Goreleaser snapshot).
2. Update `CHANGELOG.md` if needed.
3. Tag and push:

```bash
git tag v0.1.0
git push origin v0.1.0
```

4. The [Release workflow](.github/workflows/release.yml) runs Goreleaser, which:
   - Builds `gitant` and `git-remote-gitant` for linux/darwin/windows (amd64 + arm64)
   - Uploads archives to GitHub Releases
   - Pushes `ghcr.io/graycodeai/gitant-daemon:<version>` and `:latest`

## Verify locally

```bash
make test
make release    # goreleaser release --snapshot --clean
./dist/gitant-daemon_linux_amd64_v1/gitant version
```

## Version metadata

Build injects version into `internal/api` via `-ldflags`:

- `gitant version` CLI
- `GET /api/v1/status` → `"version"`

Goreleaser uses `Dockerfile.goreleaser` (copies pre-built binaries). Local/CI builds use `Dockerfile` (compiles from source).
