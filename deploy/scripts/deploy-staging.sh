#!/bin/bash
set -euo pipefail

# Gitant Staging Deployment Script
# Usage: ./deploy-staging.sh [command]

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log() { echo -e "${GREEN}[INFO]${NC} $1"; }
warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
error() { echo -e "${RED}[ERROR]${NC} $1"; exit 1; }

# Check prerequisites
check_prereqs() {
    log "Checking prerequisites..."
    command -v docker >/dev/null 2>&1 || error "Docker not installed"
    command -v docker-compose >/dev/null 2>&1 || error "Docker Compose not installed"
    log "Prerequisites OK"
}

# Build images
build() {
    log "Building daemon image..."
    cd "$PROJECT_ROOT"
    docker build -t gitant-daemon:staging .

    log "Building web image..."
    cd "$PROJECT_ROOT/../gitant-web"
    docker build -t gitant-web:staging .

    log "Build complete"
}

# Deploy to Docker Compose staging
deploy_compose() {
    log "Deploying to Docker Compose staging..."
    cd "$PROJECT_ROOT/deploy/staging"
    docker-compose down
    docker-compose up -d
    log "Staging deployed at http://localhost:7777"
}

# Deploy to Kubernetes staging
deploy_k8s() {
    log "Deploying to Kubernetes staging..."
    kubectl apply -f "$PROJECT_ROOT/deploy/k8s/staging.yaml"
    log "Kubernetes deployment applied"
    kubectl rollout status deployment/gitant-daemon -n gitant --timeout=120s
    kubectl rollout status deployment/gitant-web -n gitant --timeout=120s
    log "Kubernetes deployment complete"
}

# Run tests
test() {
    log "Running tests..."
    cd "$PROJECT_ROOT"
    make test
    log "Tests passed"
}

# Health check
health() {
    log "Running health check..."
    curl -sf http://localhost:7777/health || error "Health check failed"
    log "Health check passed"
}

# Logs
logs() {
    cd "$PROJECT_ROOT/deploy/staging"
    docker-compose logs -f
}

# Status
status() {
    log "Staging status:"
    cd "$PROJECT_ROOT/deploy/staging"
    docker-compose ps
}

# Main
case "${1:-help}" in
    build)      check_prereqs; build ;;
    deploy)     check_prereqs; build; deploy_compose ;;
    deploy-k8s) check_prereqs; build; deploy_k8s ;;
    test)       test ;;
    health)     health ;;
    logs)       logs ;;
    status)     status ;;
    *)
        echo "Usage: $0 {build|deploy|deploy-k8s|test|health|logs|status}"
        exit 1
        ;;
esac
