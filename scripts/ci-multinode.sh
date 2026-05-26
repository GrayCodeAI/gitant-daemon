#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
BIN="$ROOT/gitant"
NODE1_DIR="$(mktemp -d)"
NODE2_DIR="$(mktemp -d)"
NODE1_PID=""
NODE2_PID=""

cleanup() {
  if [[ -n "$NODE1_PID" ]]; then kill "$NODE1_PID" 2>/dev/null || true; fi
  if [[ -n "$NODE2_PID" ]]; then kill "$NODE2_PID" 2>/dev/null || true; fi
  rm -rf "$NODE1_DIR" "$NODE2_DIR"
}
trap cleanup EXIT

wait_for() {
  local url="$1"
  for _ in $(seq 1 30); do
    if curl -fsS "$url" >/dev/null 2>&1; then
      return 0
    fi
    sleep 1
  done
  echo "timeout waiting for $url" >&2
  return 1
}

echo "==> Building gitant"
(cd "$ROOT" && go build -o "$BIN" ./cmd/gitant/)

echo "==> Starting node 1"
"$BIN" serve --p2p --p2p-mdns=false --port 9777 --data-dir "$NODE1_DIR" >/tmp/gitant-node1.log 2>&1 &
NODE1_PID=$!
wait_for "http://127.0.0.1:9777/health"

BOOTSTRAP="$(curl -fsS http://127.0.0.1:9777/api/v1/status | python3 -c 'import json,sys; d=json.load(sys.stdin); print(d["p2p"]["addrs"][0])')"
echo "Node 1 bootstrap: $BOOTSTRAP"

echo "==> Starting node 2"
"$BIN" serve --p2p --p2p-mdns=false --port 9778 --data-dir "$NODE2_DIR" --bootstrap-peers "$BOOTSTRAP" >/tmp/gitant-node2.log 2>&1 &
NODE2_PID=$!
wait_for "http://127.0.0.1:9778/health"

echo "==> Waiting for peer mesh"
for _ in $(seq 1 30); do
  PEERS="$(curl -fsS http://127.0.0.1:9778/api/v1/status | python3 -c 'import json,sys; print(json.load(sys.stdin)["peers"])')"
  if [[ "$PEERS" -ge 1 ]]; then
    echo "Node 2 connected to $PEERS peer(s)"
    exit 0
  fi
  sleep 1
done

echo "Peer mesh failed — logs:" >&2
tail -30 /tmp/gitant-node1.log >&2 || true
tail -30 /tmp/gitant-node2.log >&2 || true
exit 1
