# CRDT sync and conflict resolution

Gitant replicates issue and pull request metadata with **operation-based CRDTs** over libp2p GossipSub.

## What syncs

| Entity | Transport | Merge strategy |
|--------|-----------|----------------|
| Issues | `gitant/crdt` + `gitant/repo/{id}/crdt` | Lamport-ordered op log merge |
| Pull requests | same | Lamport-ordered op log merge |
| Git objects | `/gitant/block/1.0.0` stream + DHT announce | Content-addressed; missing hashes fetched from push peer |
| Tasks | not yet replicated | Single-node until task CRDT lands |

## Conflict policy

1. **Operations are immutable** — each CRDT mutation appends an operation with a unique ID.
2. **Lamport timestamps define order** — when rebuilding state, ops sort by `(lamport, timestamp)`.
3. **Concurrent edits merge by replay** — if two nodes edit the same issue, both op sets are unioned and replayed. No ops are discarded.
4. **Last replay wins for scalar fields** — title, body, status, labels follow the final op in Lamport order (not wall-clock time).
5. **Comments accumulate** — comment ops append; duplicates are prevented by op ID deduplication.
6. **PR merge state** — a `merged` status op from any authorized peer is replayed like other status changes; branch protection remains enforced locally on merge API calls.

## Guarantees

- **Eventual consistency** across connected peers on the same gossip mesh.
- **No silent data loss** for issue/PR metadata — ops are additive.
- **Git objects** replicate only when announced in push events; peers pull missing hashes from the pushing node.

## Non-goals (current Phase C)

- Byzantine fault tolerance
- Encrypted gossip payloads
- Cross-repo ACL enforcement on inbound CRDT messages (private repo writes still require UCAN on each node’s HTTP API)

## Verification

Run multi-node replication tests:

```bash
go test ./internal/network/... -run Sync -v
go test ./internal/crdt/... -run Merge -v
```
