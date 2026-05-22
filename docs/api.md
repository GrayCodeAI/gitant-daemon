# gitant API Reference

Base URL: `http://localhost:7777`

## Health

### `GET /health`
```json
{"status": "ok"}
```

### `GET /api/v1/status`
```json
{
  "version": "0.1.0",
  "peers": 0,
  "repos": 1,
  "agents": 0,
  "uptime": "5m30s",
  "identity": "did:key:z6Mk..."
}
```

## Repositories

### `POST /api/v1/repos`
Create a repository.
```bash
curl -X POST http://localhost:7777/api/v1/repos \
  -H 'Content-Type: application/json' \
  -d '{"name":"my-repo","description":"A test","private":false}'
```

### `GET /api/v1/repos`
List all repositories.

### `GET /api/v1/repos/{id}`
Get repository details.

### `DELETE /api/v1/repos/{id}`
Delete a repository.

### `POST /api/v1/repos/{id}/push`
Push ref updates.
```json
{
  "ref_updates": [
    {"name": "refs/heads/main", "new_hash": "abc123..."}
  ]
}
```

### `GET /api/v1/repos/{id}/clone`
Get repo info and refs for cloning.

### `GET /api/v1/repos/{id}/refs`
List all refs.

### `POST /api/v1/repos/{id}/branches`
Create a branch.
```json
{"name": "feature", "commit": "abc123..."}
```

## Files

### `GET /api/v1/repos/{id}/files?path=&ref=`
List files at path.

### `GET /api/v1/repos/{id}/files/{path}?ref=`
Get file content.

### `GET /api/v1/repos/{id}/search?q=`
Search code (stub).

## Commits

### `GET /api/v1/repos/{id}/commits?ref=&limit=20`
Commit history.

### `GET /api/v1/repos/{id}/diff?from=&to=`
Diff between commits.

## Issues

### `POST /api/v1/repos/{id}/issues`
```json
{"title": "Bug report", "body": "Details...", "labels": ["bug"]}
```

### `GET /api/v1/repos/{id}/issues`
List issues.

### `GET /api/v1/repos/{id}/issues/{issueId}`
Get issue.

### `POST /api/v1/repos/{id}/issues/{issueId}/comment`
```json
{"body": "A comment"}
```

### `POST /api/v1/repos/{id}/issues/{issueId}/close`
Close issue.

## Pull Requests

### `POST /api/v1/repos/{id}/prs`
```json
{
  "title": "Feature",
  "body": "Description",
  "source_branch": "feature",
  "target_branch": "main"
}
```

### `GET /api/v1/repos/{id}/prs`
List PRs.

### `GET /api/v1/repos/{id}/prs/{prId}`
Get PR.

### `POST /api/v1/repos/{id}/prs/{prId}/review`
```json
{"verdict": "approve", "body": "LGTM"}
```

### `POST /api/v1/repos/{id}/prs/{prId}/merge`
Merge PR.

## Agents

### `GET /api/v1/agents`
List agents (stub).

### `GET /api/v1/agents/{did}`
Get agent profile.

### `POST /api/v1/agents/{did}/delegate`
Delegate UCAN capability.

### `POST /api/v1/agents/generate-did`
Generate new DID.

### `GET /api/v1/agents/resolve/{did}`
Resolve DID to document.
