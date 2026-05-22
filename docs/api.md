# gitant API Reference

Base URL: `http://localhost:7777/api/v1`

All list endpoints support pagination via `offset` and `limit` query parameters.
Default page size: 20, max: 100.

## Health & Status

### GET /health
Health check. Returns `{"status":"ok"}`.

### GET /api/v1/status
Returns node status: version, uptime, identity DID, peer/repo/agent counts.

---

## Repositories

### GET /api/v1/repos
List all repositories.

**Response:** `{ "repos": [...], "total": N, "offset": N, "limit": N }`

### POST /api/v1/repos
Create a repository. **Auth required.**

**Body:**
```json
{
  "name": "my-project",
  "description": "Optional description",
  "private": false
}
```

### GET /api/v1/repos/{id}
Get repository details.

### DELETE /api/v1/repos/{id}
Delete a repository. **Auth required.**

### POST /api/v1/repos/{id}/fork
Fork a repository. **Auth required.**

**Body:** `{ "name": "fork-name" }`

### POST /api/v1/repos/{id}/star
Star a repository. **Auth required.**

### POST /api/v1/repos/{id}/unstar
Unstar a repository. **Auth required.**

### GET /api/v1/repos/{id}/stars
Get star count and list of stargazers.

---

## Git Operations

### GET /api/v1/repos/{id}/refs
List all references (branches, tags).

### POST /api/v1/repos/{id}/branches
Create a branch. **Auth required.**

**Body:** `{ "name": "feature-x", "commit": "<hash>" }`

### POST /api/v1/repos/{id}/push
Push objects (JSON format). **Auth required.**

**Body:**
```json
{
  "objects": [
    { "hash": "...", "type": "commit", "content": "<base64>" }
  ],
  "ref_updates": [
    { "name": "refs/heads/main", "new_hash": "..." }
  ]
}
```

### POST /api/v1/repos/{id}/push-packfile
Push using a git packfile (more efficient). **Auth required.**

**Body:**
```json
{
  "packfile": "<base64-encoded-packfile>",
  "ref_updates": [
    { "name": "refs/heads/main", "new_hash": "..." }
  ]
}
```

### GET /api/v1/repos/{id}/clone
Clone repository (git smart-HTTP).

### GET /api/v1/repos/{id}/info/refs
Git smart-HTTP info/refs endpoint.

### POST /api/v1/repos/{id}/git-upload-pack
Git smart-HTTP upload-pack.

### POST /api/v1/repos/{id}/git-receive-pack
Git smart-HTTP receive-pack (accepts packfile).

---

## Files

### GET /api/v1/repos/{id}/files
List files in repository tree.

**Query:** `path` (subdirectory), `ref` (branch/tag), `offset`, `limit`

### GET /api/v1/repos/{id}/files/{path}
Get file contents.

**Query:** `ref` (branch/tag)

### GET /api/v1/repos/{id}/search
Search code in repository.

**Query:** `q` (required), `ref`, `offset`, `limit`

**Response:**
```json
{
  "query": "function",
  "results": [
    { "file": "src/main.go", "line": 42, "context": "src/main.go:42: func main() {" }
  ],
  "total": N,
  "offset": N,
  "limit": N
}
```

---

## Commits

### GET /api/v1/repos/{id}/commits
Get commit history.

**Query:** `ref` (starting point), `offset`, `limit`

### GET /api/v1/repos/{id}/diff
Diff two commits.

**Query:** `from` (hash), `to` (hash)

### GET /api/v1/repos/{id}/commits/{hash}/parents
Get diffs against all parent commits.

---

## Issues

### GET /api/v1/repos/{id}/issues
List issues. **Query:** `offset`, `limit`

### POST /api/v1/repos/{id}/issues
Create an issue. **Auth required.**

**Body:**
```json
{
  "title": "Bug report",
  "body": "Description of the issue",
  "labels": ["bug"]
}
```

### GET /api/v1/repos/{id}/issues/{issueId}
Get issue details.

### POST /api/v1/repos/{id}/issues/{issueId}/comment
Comment on an issue. **Auth required.**

**Body:** `{ "body": "Comment text" }`

### POST /api/v1/repos/{id}/issues/{issueId}/close
Close an issue. **Auth required.**

### GET /api/v1/repos/{id}/issues/{issueId}/comments
List issue comments. **Query:** `offset`, `limit`

---

## Pull Requests

### GET /api/v1/repos/{id}/prs
List pull requests. **Query:** `offset`, `limit`

### POST /api/v1/repos/{id}/prs
Open a pull request. **Auth required.**

**Body:**
```json
{
  "title": "Add feature",
  "body": "Description",
  "source_branch": "feature",
  "target_branch": "main"
}
```

### GET /api/v1/repos/{id}/prs/{prId}
Get PR details.

### POST /api/v1/repos/{id}/prs/{prId}/review
Review a PR. **Auth required.**

**Body:** `{ "verdict": "approve|reject", "body": "Comments" }`

### POST /api/v1/repos/{id}/prs/{prId}/merge
Merge a PR. **Auth required.**

### GET /api/v1/repos/{id}/prs/{prId}/comments
List PR comments. **Query:** `offset`, `limit`

---

## Labels

### GET /api/v1/repos/{id}/labels
List labels. **Query:** `offset`, `limit`

### POST /api/v1/repos/{id}/labels
Create a label. **Auth required.**

**Body:** `{ "name": "bug", "color": "#ff0000" }`

### DELETE /api/v1/repos/{id}/labels/{name}
Delete a label. **Auth required.**

---

## Tasks

### GET /api/v1/repos/{id}/tasks
List tasks. **Query:** `status` (open|claimed|completed|failed), `offset`, `limit`

### POST /api/v1/repos/{id}/tasks
Create a task. **Auth required.**

**Body:** `{ "title": "Fix bug", "description": "Details" }`

### POST /api/v1/repos/{id}/tasks/{taskId}/claim
Claim a task. **Auth required.**

### POST /api/v1/repos/{id}/tasks/{taskId}/complete
Complete a task. **Auth required.**

**Body:** `{ "result": "Optional result description" }`

---

## Branch Protection

### GET /api/v1/repos/{id}/protections
List all protection rules. **Query:** `offset`, `limit`

### GET /api/v1/repos/{id}/protections/{branch}
Get protection rule for a branch.

### PUT /api/v1/repos/{id}/protections/{branch}
Set protection rule. **Auth required.**

**Body:**
```json
{
  "require_pr": true,
  "require_approval": true,
  "no_force_push": true
}
```

### DELETE /api/v1/repos/{id}/protections/{branch}
Remove protection rule. **Auth required.**

---

## Activity

### GET /api/v1/activity
Get unified activity feed across all repos.

**Query:** `offset`, `limit`

---

## Agents & Identity

### GET /api/v1/agents
List known agents. **Query:** `offset`, `limit`

### GET /api/v1/agents/{did}
Get agent details.

### POST /api/v1/agents/generate-did
Generate a new DID:key identity. **Auth required.**

### POST /api/v1/agents/resolve/{did}
Resolve a DID to a DID document.

### POST /api/v1/agents/{did}/delegate
Delegate capabilities to another DID. **Auth required.**

**Body:**
```json
{
  "audience": "did:key:z...",
  "resource": "repo:my-project",
  "actions": ["read", "write"]
}
```

### POST /api/v1/agents/verify
Verify a UCAN token. **Auth required.**

**Body:** `{ "ucan": "<token>" }`

---

## Webhooks

### GET /api/v1/webhooks
List registered webhooks. **Query:** `offset`, `limit`

### POST /api/v1/webhooks
Register a webhook. **Auth required.**

**Body:**
```json
{
  "url": "https://example.com/webhook",
  "events": ["push", "issue.created", "pr.merged"],
  "secret": "optional-secret"
}
```

### DELETE /api/v1/webhooks/{id}
Delete a webhook. **Auth required.**

---

## UCAN Management

### POST /api/v1/ucan/revoke
Revoke a UCAN by nonce. **Auth required.**

**Body:** `{ "nonce": "...", "reason": "optional" }`

### GET /api/v1/ucan/revocations
List all revoked UCANs. **Query:** `offset`, `limit`

---

## Webhook Events

| Event | Description |
|-------|-------------|
| `push` | Code pushed to repository |
| `issue.created` | New issue created |
| `issue.closed` | Issue closed |
| `issue.commented` | Comment on issue |
| `pr.opened` | Pull request opened |
| `pr.merged` | Pull request merged |
| `pr.reviewed` | Pull request reviewed |
| `repo.created` | Repository created |
| `repo.deleted` | Repository deleted |
