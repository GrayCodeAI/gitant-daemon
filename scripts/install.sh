#!/usr/bin/env bash
# Gitant install script — https://get.gitant.dev/install.sh
set -euo pipefail

GITANT_VERSION="${GITANT_VERSION:-latest}"
GITANT_REPO="${GITANT_REPO:-GrayCodeAI/gitant-daemon}"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"
USE_DOCKER="${USE_DOCKER:-0}"

info() { printf '==> %s\n' "$*"; }
warn() { printf 'warning: %s\n' "$*" >&2; }

detect_os_arch() {
  local os arch
  os="$(uname -s | tr '[:upper:]' '[:lower:]')"
  arch="$(uname -m)"
  case "$arch" in
    x86_64|amd64) arch="amd64" ;;
    arm64|aarch64) arch="arm64" ;;
    *) warn "unsupported architecture: $arch"; exit 1 ;;
  esac
  case "$os" in
    linux|darwin) ;;
    *) warn "unsupported OS: $os"; exit 1 ;;
  esac
  printf '%s-%s' "$os" "$arch"
}

install_binary() {
  local platform asset url tmp
  platform="$(detect_os_arch)"
  if [[ "$GITANT_VERSION" == "latest" ]]; then
    url="https://github.com/${GITANT_REPO}/releases/latest/download/gitant_${platform}.tar.gz"
  else
    url="https://github.com/${GITANT_REPO}/releases/download/${GITANT_VERSION}/gitant_${platform}.tar.gz"
  fi

  info "Downloading gitant ${GITANT_VERSION} for ${platform}"
  tmp="$(mktemp -d)"
  trap 'rm -rf "$tmp"' EXIT
  curl -fsSL "$url" -o "${tmp}/gitant.tar.gz"
  tar -xzf "${tmp}/gitant.tar.gz" -C "$tmp"
  install -m 755 "${tmp}/gitant" "${INSTALL_DIR}/gitant"
  if [[ -f "${tmp}/git-remote-gitant" ]]; then
    install -m 755 "${tmp}/git-remote-gitant" "${INSTALL_DIR}/git-remote-gitant"
  fi
  info "Installed to ${INSTALL_DIR}/gitant"
}

install_docker_stack() {
  info "Docker mode: run daemon + web from gitant-daemon"
  cat <<'EOF'

Run the full stack (daemon + web):

  git clone https://github.com/GrayCodeAI/gitant-daemon.git
  git clone https://github.com/GrayCodeAI/gitant-web.git
  cd gitant-daemon
  docker compose -f docker-compose.stack.yml up -d

Daemon: http://localhost:7777
Web UI: http://localhost:3303

Install the CLI separately: https://github.com/GrayCodeAI/gitant-cli

EOF
}

main() {
  if [[ "${USE_DOCKER}" == "1" ]]; then
    install_docker_stack
    exit 0
  fi

  if ! command -v curl >/dev/null 2>&1; then
    warn "curl is required"
    exit 1
  fi

  install_binary
  info "Start the daemon: gitant serve --p2p"
  info "Enable LAN discovery: gitant serve --p2p --p2p-mdns"
}

main "$@"
