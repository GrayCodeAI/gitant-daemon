# Contributing to Gitant

Thank you for your interest in contributing to Gitant! This document provides guidelines and instructions for contributing.

## Getting Started

1. Fork the repository
2. Clone your fork
3. Create a feature branch
4. Make your changes
5. Submit a pull request

## Development Setup

### Prerequisites

- Go 1.26+
- Node.js 20+
- Git

### Daemon

```bash
cd gitant-daemon
make build
make test
make lint
```

### CLI

```bash
cd gitant-cli
make build
make test
```

### MCP Server

```bash
cd gitant-mcp
npm install
npm run build
npm test
```

### Web UI

```bash
cd gitant-web
npm install
npm run dev
```

## Code Style

### Go

- Follow standard Go conventions
- Run `gofmt` before committing
- Run `golangci-lint` for linting
- Write tests for new features

### TypeScript

- Use ESLint with the provided config
- Run `npm run lint` before committing
- Write tests for new features

## Commit Messages

Follow conventional commits:

```
feat: add new feature
fix: fix bug
docs: update documentation
test: add tests
refactor: refactor code
chore: update dependencies
```

## Pull Request Process

1. Update documentation if needed
2. Add tests for new features
3. Ensure all tests pass
4. Ensure lint passes
5. Request review from maintainers

## Reporting Issues

- Use GitHub Issues
- Include reproduction steps
- Include system information
- Include logs if applicable

## Code of Conduct

Please read [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md).

## License

By contributing, you agree that your contributions will be licensed under the MIT License.
