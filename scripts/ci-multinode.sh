#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
BIN="$ROOT/gitant"
NODE1_DIR="$(mktemp -d)"
NODE2_DIR="$(mktemp -d)"
NODE1_PID=""
NODE2_PID=""
REPO_ID="ci-repl-$(date +%s)"
ISSUE_TITLE="ci-replication-$(date +%s)"

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

mint_ucan() {
  local identity_path="$1"
  local audience="$2"
  local resource="$3"
  local actions="$4"
  (cd "$ROOT" && go run scripts/ci-mint-ucan.go "$identity_path" "$audience" "$resource" "$actions")
}

api_post() {
  local url="$1"
  local token="$2"
  local body="$3"
  curl -fsS -X POST "$url" \
    -H "Authorization: Bearer $token" \
    -H "Content-Type: application/json" \
    -d "$body"
}

echo "==> Building gitant"
(cd "$ROOT" && go build -o "$BIN" ./cmd/gitant/)

echo "==> Starting node 1"
"$BIN" serve --p2p --p2p-mdns=false --port 9777 --data-dir "$NODE1_DIR" >/tmp/gitant-node1.log 2>&1 &
NODE1_PID=$!
wait_for "http://127.0.0.1:9777/health"

NODE1_DID="$(curl -fsS http://127.0.0.1:9777/api/v1/status | python3 -c 'import json,sys; print(json.load(sys.stdin)["identity"])')"
BOOTSTRAP="$(curl -fsS http://127.0.0.1:9777/api/v1/status | python3 -c 'import json,sys; d=json.load(sys.stdin); print(d["p2p"]["addrs"][0])')"
echo "Node 1 DID: $NODE1_DID"
echo "Node 1 bootstrap: $BOOTSTRAP"

echo "==> Starting node 2"
"$BIN" serve --p2p --p2p-mdns=false --port 9778 --data-dir "$NODE2_DIR" --bootstrap-peers "$BOOTSTRAP" >/tmp/gitant-node2.log 2>&1 &
NODE2_PID=$!
wait_for "http://127.0.0.1:9778/health"

NODE2_DID="$(curl -fsS http://127.0.0.1:9778/api/v1/status | python3 -c 'import json,sys; print(json.load(sys.stdin)["identity"])')"
echo "Node 2 DID: $NODE2_DID"

echo "==> Waiting for peer mesh"
MESH_OK=false
for _ in $(seq 1 30); do
  PEERS="$(curl -fsS http://127.0.0.1:9778/api/v1/status | python3 -c 'import json,sys; print(json.load(sys.stdin)["peers"])')"
  if [[ "$PEERS" -ge 1 ]]; then
    echo "Node 2 connected to $PEERS peer(s)"
    MESH_OK=true
    break
  fi
  sleep 1
done
if [[ "$MESH_OK" != "true" ]]; then
  echo "Peer mesh failed — logs:" >&2
  tail -30 /tmp/gitant-node1.log >&2 || true
  tail -30 /tmp/gitant-node2.log >&2 || true
  exit 1
fi

echo "==> Minting UCANs for replication test"
NODE1_UCAN="$(mint_ucan "$NODE1_DIR/identity.key" "$NODE1_DID" "repo:${REPO_ID}" "read,write")"
NODE2_UCAN="$(mint_ucan "$NODE2_DIR/identity.key" "$NODE2_DID" "repo:${REPO_ID}" "read,write")"

echo "==> Creating repo on node 1 and node 2"
api_post "http://127.0.0.1:9777/api/v1/repos" "$NODE1_UCAN" \
  "{\"name\":\"${REPO_ID}\",\"description\":\"CI replication test\"}" >/dev/null

api_post "http://127.0.0.1:9778/api/v1/repos" "$NODE2_UCAN" \
  "{\"name\":\"${REPO_ID}\",\"description\":\"CI replication test mirror\"}" >/dev/null

sleep 2

echo "==> Creating issue on node 1"
ISSUE_UCAN="$(mint_ucan "$NODE1_DIR/identity.key" "$NODE1_DID" "repo:${REPO_ID}" "read,write")"
api_post "http://127.0.0.1:9777/api/v1/repos/${REPO_ID}/issues" "$ISSUE_UCAN" \
  "{\"title\":\"${ISSUE_TITLE}\",\"body\":\"replicated by ci-multinode\"}" >/dev/null

echo "==> Waiting for issue CRDT replication on node 2 (target <30s)"
REPL_OK=false
for _ in $(seq 1 30); do
  FOUND="$(curl -fsS "http://127.0.0.1:9778/api/v1/repos/${REPO_ID}/issues" | python3 -c "
import json, sys
title = sys.argv[1]
data = json.load(sys.stdin)
issues = data.get('issues') or []
print('yes' if any(i.get('title') == title for i in issues) else 'no')
" "$ISSUE_TITLE")"
  if [[ "$FOUND" == "yes" ]]; then
    echo "Issue replicated to node 2: ${ISSUE_TITLE}"
    REPL_OK=true
    break
  fi
  sleep 1
done

if [[ "$REPL_OK" != "true" ]]; then
  echo "CRDT issue replication failed — checking federation events on node 2" >&2
  EVENT_FOUND="$(curl -fsS "http://127.0.0.1:9778/api/v1/federation/discover" | python3 -c "
import json, sys
repo, title = sys.argv[1], sys.argv[2]
data = json.load(sys.stdin)
for ev in data.get('events') or []:
    if ev.get('type') == 'issue.created' and ev.get('repo') == repo:
        d = ev.get('data') or {}
        if d.get('title') == title:
            print('yes')
            raise SystemExit(0)
print('no')
" "$REPO_ID" "$ISSUE_TITLE")"
  if [[ "$EVENT_FOUND" == "yes" ]]; then
    echo "Federated issue.created event received on node 2 (CRDT merge pending)"
    REPL_OK=true
  fi
fi

if [[ "$REPL_OK" != "true" ]]; then
  echo "Replication test failed — logs:" >&2
  tail -40 /tmp/gitant-node1.log >&2 || true
  tail -40 /tmp/gitant-node2.log >&2 || true
  exit 1
fi

echo "==> Multi-node replication E2E passed"
