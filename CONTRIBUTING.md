# Contributing to gitant

## Development Setup

```bash
# Clone
git clone https://github.com/lakshmanpatel/gitant.git
cd gitant/gitant-daemon

# Build
go build ./...

# Test
go test ./...

# Lint
go vet ./...
```

## Code Style

- Use `gofmt` for formatting
- Use `go vet` for static analysis
- Follow standard Go conventions (exported/unexported, package naming)
- Add tests for new functionality

## Pull Requests

1. Fork the repo
2. Create a feature branch
3. Make your changes
4. Run `go test ./...` and `go vet ./...`
5. Submit a PR with a clear description

## Issues

- Use the issue tracker for bugs and feature requests
- Include reproduction steps for bugs
- Label issues appropriately
