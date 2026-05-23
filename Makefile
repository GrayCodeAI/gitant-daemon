.PHONY: build test lint clean install docker run vet fmt

# Build variables
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME ?= $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')

LDFLAGS := -ldflags "-s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.buildTime=$(BUILD_TIME)"

## build: Compile all binaries
build:
	CGO_ENABLED=0 go build $(LDFLAGS) -o bin/gitant ./cmd/gitant/
	CGO_ENABLED=0 go build $(LDFLAGS) -o bin/git-remote-gitant ./cmd/git-remote-gitant/

## install: Install binaries to $GOPATH/bin
install:
	go install $(LDFLAGS) ./cmd/gitant/
	go install $(LDFLAGS) ./cmd/git-remote-gitant/

## test: Run all tests with race detector
test:
	go test ./... -race -count=1 -timeout=120s

## test-cover: Run tests with coverage report
test-cover:
	go test ./... -race -coverprofile=coverage.out -covermode=atomic
	go tool cover -html=coverage.out -o coverage.html

## vet: Run go vet
vet:
	go vet ./...

## fmt: Check formatting
fmt:
	gofmt -l .

## lint: Run golangci-lint
lint:
	golangci-lint run ./...

## clean: Remove build artifacts
clean:
	rm -rf bin/ coverage.out coverage.html

## docker: Build Docker image
docker:
	docker build -t gitant:$(VERSION) .

## docker-compose: Start services
docker-compose:
	docker-compose up -d

## run: Build and start the daemon
run: build
	./bin/gitant serve

## all: Run all checks (fmt, vet, lint, test)
all: fmt vet lint test
