# Federation and bootstrap seeds

Gitant nodes federate over libp2p (DHT + GossipSub). WAN discovery uses bootstrap multiaddrs.

## Bootstrap configuration

| Source | Example |
|--------|---------|
| Embedded defaults | `internal/network/bootstrap_peers.json` (empty until you deploy seeds) |
| Environment | `GITANT_SEED_PEERS=/ip4/203.0.113.10/tcp/4001/p2p/Qm...` |
| CLI flag | `gitant serve --p2p --bootstrap-peers /ip4/.../p2p/...` |
| API | `GET /api/v1/network/bootstrap` |

## Running a seed node

```bash
gitant serve --p2p --p2p-listen /ip4/0.0.0.0/tcp/4001 --port 7777
curl -s http://localhost:7777/api/v1/status | jq .p2p.addrs
```

Publish the returned multiaddr in `bootstrap_peers.json` or `GITANT_SEED_PEERS`.

## LAN vs WAN

- **LAN**: `--p2p-mdns` discovers peers automatically.
- **WAN**: peers need `--bootstrap-peers` or seed env vars to join the mesh.

## IPFS warm pinning

Enable optional warm storage for replicated git objects:

```bash
gitant serve --p2p --ipfs-pin
```

Pinned object count appears in `GET /api/v1/status` as `ipfs_pins`.

## Agent trust attestations

Authenticated operators can attest agent trust:

```bash
curl -X POST http://localhost:7777/api/v1/agents/did:key:.../attest \
  -H 'Authorization: Bearer $UCAN' \
  -H 'Content-Type: application/json' \
  -d '{"score":0.9,"reason":"reliable automation agent"}'
```

Attestations gossip on `gitant/attestations` and blend into each node's local trust score (70/30 EMA).
