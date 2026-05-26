package api

import (
	"encoding/json"
	"net/http"
)

func handleOpenAPI(w http.ResponseWriter, r *http.Request) {
	spec := map[string]interface{}{
		"openapi": "3.0.3",
		"info": map[string]interface{}{
			"title":       "gitant API",
			"version":     Version,
			"description": "Decentralized git hosting platform API",
		},
		"servers": []map[string]string{
			{"url": "/", "description": "Current server"},
		},
		"paths": map[string]interface{}{
			"/health": map[string]interface{}{
				"get": map[string]interface{}{
					"summary":     "Health check",
					"description": "Returns health status with dependency checks",
					"tags":        []string{"System"},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Healthy",
							"content":     jsonContent(map[string]interface{}{"status": "healthy", "checks": map[string]string{"identity": "ok", "storage": "ok"}}),
						},
						"503": map[string]interface{}{"description": "Degraded"},
					},
				},
			},
			"/api/v1/status": map[string]interface{}{
				"get": map[string]interface{}{
					"summary":     "Server status",
					"description": "Returns version, uptime, peer count, repo count, identity DID",
					"tags":        []string{"System"},
					"responses": successResponse(map[string]interface{}{
						"version": "0.1.0", "peers": 0, "repos": 1,
						"agents": 1, "uptime": "1h2m3s", "identity": "did:key:z6Mk...",
						"p2p": map[string]interface{}{"enabled": false},
					}),
				},
			},
			"/api/v1/network/peers": map[string]interface{}{
				"get": map[string]interface{}{
					"summary":     "P2P network status",
					"description": "Returns libp2p peer ID, listen addresses, and connected peers",
					"tags":        []string{"System"},
					"responses":   successResponse(map[string]interface{}{"enabled": true, "peers": 0}),
				},
			},
			"/api/v1/federation/discover": map[string]interface{}{
				"get": map[string]interface{}{
					"summary":     "Federation discovery",
					"description": "Announces this node and returns known federation records",
					"tags":        []string{"System"},
					"parameters": []map[string]interface{}{
						{"name": "did", "in": "query", "schema": map[string]string{"type": "string"}},
					},
					"responses": successResponse(map[string]interface{}{"nodes": []interface{}{}}),
				},
			},
			"/api/v1/repos": map[string]interface{}{
				"get": map[string]interface{}{
					"summary":     "List repositories",
					"description": "Returns paginated list of all repositories",
					"tags":        []string{"Repos"},
					"parameters":  paginationParams(),
					"responses":   statusResponse(),
				},
				"post": map[string]interface{}{
					"summary":     "Create repository",
					"description": "Creates a new repository (requires auth)",
					"tags":        []string{"Repos"},
					"security":    bearerAuth(),
					"requestBody": jsonBody(map[string]interface{}{
						"name":        "my-repo",
						"description": "Optional description",
						"private":     false,
					}),
					"responses": statusResponse(),
				},
			},
			"/api/v1/repos/{id}": map[string]interface{}{
				"get": map[string]interface{}{
					"summary":     "Get repository",
					"description": "Returns repository metadata, refs, and latest commit",
					"tags":        []string{"Repos"},
					"parameters":  []map[string]interface{}{pathParam("id", "Repository ID")},
					"responses":   statusResponse(),
				},
				"delete": map[string]interface{}{
					"summary":     "Delete repository",
					"description": "Deletes a repository (requires auth)",
					"tags":        []string{"Repos"},
					"security":    bearerAuth(),
					"parameters":  []map[string]interface{}{pathParam("id", "Repository ID")},
					"responses":   statusResponse(),
				},
			},
			"/api/v1/repos/{id}/stars": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Get star count", "tags": []string{"Repos"},
					"parameters": []map[string]interface{}{pathParam("id", "Repository ID")},
					"responses":  statusResponse(),
				},
			},
			"/api/v1/repos/{id}/clone": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Clone repository", "tags": []string{"Repos"},
					"parameters": []map[string]interface{}{pathParam("id", "Repository ID")},
					"responses":  statusResponse(),
				},
			},
			"/api/v1/repos/{id}/refs": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "List refs", "tags": []string{"Repos"},
					"parameters": []map[string]interface{}{pathParam("id", "Repository ID")},
					"responses":  statusResponse(),
				},
			},
			"/api/v1/repos/{id}/objects/{hash}": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Get git object", "tags": []string{"Repos"},
					"parameters": []map[string]interface{}{
						pathParam("id", "Repository ID"),
						pathParam("hash", "Object hash"),
					},
					"responses": statusResponse(),
				},
			},
			"/api/v1/repos/{id}/info/refs": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Git info/refs discovery", "tags": []string{"Git"},
					"parameters": []map[string]interface{}{
						pathParam("id", "Repository ID"),
						queryParam("service", "Git service (git-upload-pack or git-receive-pack)", false),
					},
					"responses": statusResponse(),
				},
			},
			"/api/v1/repos/{id}/issues": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "List issues", "tags": []string{"Issues"},
					"parameters": []map[string]interface{}{
						pathParam("id", "Repository ID"),
						queryParam("status", "Filter by status (open, closed, all)", false),
						queryParam("labels", "Comma-separated labels (issue must have all)", false),
						queryParam("offset", "Pagination offset", false),
						queryParam("limit", "Page size", false),
					},
					"responses": statusResponse(),
				},
				"post": map[string]interface{}{
					"summary":     "Create issue",
					"description": "Creates a new issue (max title 256 chars, max body 65536 chars, max 20 labels)",
					"tags":        []string{"Issues"},
					"security":    bearerAuth(),
					"parameters":  []map[string]interface{}{pathParam("id", "Repository ID")},
					"requestBody": jsonBody(map[string]interface{}{
						"title":  "Bug report",
						"body":   "Description of the issue",
						"labels": []string{"bug"},
					}),
					"responses": statusResponse(),
				},
			},
			"/api/v1/repos/{id}/issues/{issueId}": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Get issue", "tags": []string{"Issues"},
					"parameters": []map[string]interface{}{
						pathParam("id", "Repository ID"),
						pathParam("issueId", "Issue ID"),
					},
					"responses": statusResponse(),
				},
			},
			"/api/v1/repos/{id}/issues/{issueId}/comments": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "List issue comments", "tags": []string{"Issues"},
					"parameters": []map[string]interface{}{
						pathParam("id", "Repository ID"),
						pathParam("issueId", "Issue ID"),
					},
					"responses": statusResponse(),
				},
			},
			"/api/v1/repos/{id}/issues/{issueId}/comment": map[string]interface{}{
				"post": map[string]interface{}{
					"summary": "Comment on issue", "tags": []string{"Issues"},
					"security":   bearerAuth(),
					"parameters": []map[string]interface{}{pathParam("id", "Repository ID"), pathParam("issueId", "Issue ID")},
					"requestBody": jsonBody(map[string]interface{}{"body": "Comment text"}),
					"responses":   statusResponse(),
				},
			},
			"/api/v1/repos/{id}/issues/{issueId}/close": map[string]interface{}{
				"post": map[string]interface{}{
					"summary": "Close issue", "tags": []string{"Issues"},
					"security":   bearerAuth(),
					"parameters": []map[string]interface{}{pathParam("id", "Repository ID"), pathParam("issueId", "Issue ID")},
					"responses":   statusResponse(),
				},
			},
			"/api/v1/repos/{id}/prs": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "List pull requests", "tags": []string{"PRs"},
					"parameters": []map[string]interface{}{
						pathParam("id", "Repository ID"),
						queryParam("status", "Filter by status (open, closed, merged, all)", false),
						queryParam("offset", "Pagination offset", false),
						queryParam("limit", "Page size", false),
					},
					"responses": statusResponse(),
				},
				"post": map[string]interface{}{
					"summary":     "Open pull request",
					"description": "Creates a new pull request (max title 256 chars, max body 65536 chars)",
					"tags":        []string{"PRs"},
					"security":    bearerAuth(),
					"parameters":  []map[string]interface{}{pathParam("id", "Repository ID")},
					"requestBody": jsonBody(map[string]interface{}{
					"title":         "Fix bug",
					"body":          "Description of changes",
					"source_branch": "feature-branch",
					"target_branch": "main",
				}),
					"responses": statusResponse(),
				},
			},
			"/api/v1/repos/{id}/prs/{prId}": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Get pull request", "tags": []string{"PRs"},
					"parameters": []map[string]interface{}{
						pathParam("id", "Repository ID"),
						pathParam("prId", "PR ID"),
					},
					"responses": statusResponse(),
				},
			},
			"/api/v1/repos/{id}/prs/{prId}/comments": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "List PR comments", "tags": []string{"PRs"},
					"parameters": []map[string]interface{}{
						pathParam("id", "Repository ID"),
						pathParam("prId", "PR ID"),
					},
					"responses": statusResponse(),
				},
			},
			"/api/v1/repos/{id}/prs/{prId}/review": map[string]interface{}{
				"post": map[string]interface{}{
					"summary":     "Review pull request",
					"description": "Submit a review (verdict: approve, request_changes, comment)",
					"tags":        []string{"PRs"},
					"security":    bearerAuth(),
					"parameters":  []map[string]interface{}{pathParam("id", "Repository ID"), pathParam("prId", "PR ID")},
					"requestBody": jsonBody(map[string]interface{}{"verdict": "approve", "body": "Looks good"}),
					"responses":   statusResponse(),
				},
			},
			"/api/v1/repos/{id}/prs/{prId}/merge": map[string]interface{}{
				"post": map[string]interface{}{
					"summary": "Merge pull request", "tags": []string{"PRs"},
					"security":   bearerAuth(),
					"parameters": []map[string]interface{}{pathParam("id", "Repository ID"), pathParam("prId", "PR ID")},
					"responses":   statusResponse(),
				},
			},
			"/api/v1/repos/{id}/files": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "List files", "tags": []string{"Files"},
					"parameters": []map[string]interface{}{
						pathParam("id", "Repository ID"),
						queryParam("path", "Directory path", false),
						queryParam("ref", "Git ref (branch/tag/commit)", false),
					},
					"responses": statusResponse(),
				},
			},
			"/api/v1/repos/{id}/files/{path}": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Get file contents", "tags": []string{"Files"},
					"parameters": []map[string]interface{}{
						pathParam("id", "Repository ID"),
						pathParam("path", "File path"),
						queryParam("ref", "Git ref", false),
					},
					"responses": statusResponse(),
				},
			},
			"/api/v1/repos/{id}/search": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Search code", "tags": []string{"Files"},
					"parameters": []map[string]interface{}{
						pathParam("id", "Repository ID"),
						queryParam("q", "Search query", true),
						queryParam("ref", "Git ref", false),
					},
					"responses": statusResponse(),
				},
			},
			"/api/v1/repos/{id}/commits": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Get commit log", "tags": []string{"Commits"},
					"parameters": []map[string]interface{}{
						pathParam("id", "Repository ID"),
						queryParam("ref", "Branch or tag", false),
						queryParam("limit", "Max commits to return", false),
					},
					"responses": statusResponse(),
				},
			},
			"/api/v1/repos/{id}/diff": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Diff commits", "tags": []string{"Commits"},
					"parameters": []map[string]interface{}{
						pathParam("id", "Repository ID"),
						queryParam("from", "From commit hash", true),
						queryParam("to", "To commit hash", true),
					},
					"responses": statusResponse(),
				},
			},
			"/api/v1/repos/{id}/commits/{hash}/parents": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Get commit parents", "tags": []string{"Commits"},
					"parameters": []map[string]interface{}{
						pathParam("id", "Repository ID"),
						pathParam("hash", "Commit hash"),
					},
					"responses": statusResponse(),
				},
			},
			"/api/v1/repos/{id}/labels": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "List labels", "tags": []string{"Labels"},
					"parameters":  []map[string]interface{}{pathParam("id", "Repository ID")},
					"responses":   statusResponse(),
				},
				"post": map[string]interface{}{
					"summary": "Create label", "tags": []string{"Labels"},
					"security":   bearerAuth(),
					"parameters": []map[string]interface{}{pathParam("id", "Repository ID")},
					"requestBody": jsonBody(map[string]interface{}{"name": "bug", "color": "#ff0000"}),
					"responses":   statusResponse(),
				},
			},
			"/api/v1/repos/{id}/labels/{name}": map[string]interface{}{
				"delete": map[string]interface{}{
					"summary": "Delete label", "tags": []string{"Labels"},
					"security":   bearerAuth(),
					"parameters": []map[string]interface{}{pathParam("id", "Repository ID"), pathParam("name", "Label name")},
					"responses":   statusResponse(),
				},
			},
			"/api/v1/repos/{id}/tasks": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "List tasks", "tags": []string{"Tasks"},
					"parameters": []map[string]interface{}{
						pathParam("id", "Repository ID"),
						queryParam("status", "Filter by status (open, claimed, completed)", false),
					},
					"responses": statusResponse(),
				},
				"post": map[string]interface{}{
					"summary": "Create task", "tags": []string{"Tasks"},
					"security":   bearerAuth(),
					"parameters": []map[string]interface{}{pathParam("id", "Repository ID")},
					"requestBody": jsonBody(map[string]interface{}{"title": "Implement feature", "description": "Details"}),
					"responses":   statusResponse(),
				},
			},
			"/api/v1/repos/{id}/tasks/{taskId}/claim": map[string]interface{}{
				"post": map[string]interface{}{
					"summary": "Claim task", "tags": []string{"Tasks"},
					"security":   bearerAuth(),
					"parameters": []map[string]interface{}{pathParam("id", "Repository ID"), pathParam("taskId", "Task ID")},
					"responses":   statusResponse(),
				},
			},
			"/api/v1/repos/{id}/tasks/{taskId}/complete": map[string]interface{}{
				"post": map[string]interface{}{
					"summary": "Complete task", "tags": []string{"Tasks"},
					"security":   bearerAuth(),
					"parameters": []map[string]interface{}{pathParam("id", "Repository ID"), pathParam("taskId", "Task ID")},
					"requestBody": jsonBody(map[string]interface{}{"result": "Completed successfully"}),
					"responses":   statusResponse(),
				},
			},
			"/api/v1/repos/{id}/releases": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "List releases", "tags": []string{"Releases"},
					"parameters": []map[string]interface{}{pathParam("id", "Repository ID")},
					"responses":  statusResponse(),
				},
				"post": map[string]interface{}{
					"summary":     "Create release",
					"description": "Create a new tagged release (requires auth)",
					"tags":        []string{"Releases"},
					"security":    bearerAuth(),
					"parameters":  []map[string]interface{}{pathParam("id", "Repository ID")},
					"requestBody": jsonBody(map[string]interface{}{"tag": "v1.0.0", "title": "Release title", "body": "Release notes"}),
					"responses":   statusResponse(),
				},
			},
			"/api/v1/repos/{id}/releases/{releaseId}": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Get release", "tags": []string{"Releases"},
					"parameters": []map[string]interface{}{pathParam("id", "Repository ID"), pathParam("releaseId", "Release ID")},
					"responses":  statusResponse(),
				},
				"delete": map[string]interface{}{
					"summary": "Delete release", "tags": []string{"Releases"},
					"security":   bearerAuth(),
					"parameters": []map[string]interface{}{pathParam("id", "Repository ID"), pathParam("releaseId", "Release ID")},
					"responses":   statusResponse(),
				},
			},
			"/api/v1/repos/{id}/protections": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "List branch protections", "tags": []string{"Protections"},
					"parameters": []map[string]interface{}{pathParam("id", "Repository ID")},
					"responses":  statusResponse(),
				},
			},
			"/api/v1/repos/{id}/protections/{branch}": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Get branch protection", "tags": []string{"Protections"},
					"parameters": []map[string]interface{}{pathParam("id", "Repository ID"), pathParam("branch", "Branch name")},
					"responses":  statusResponse(),
				},
				"put": map[string]interface{}{
					"summary": "Set branch protection", "tags": []string{"Protections"},
					"security":   bearerAuth(),
					"parameters": []map[string]interface{}{pathParam("id", "Repository ID"), pathParam("branch", "Branch name")},
					"requestBody": jsonBody(map[string]interface{}{"require_pr": true, "require_approval": true, "no_force_push": true}),
					"responses":   statusResponse(),
				},
				"delete": map[string]interface{}{
					"summary": "Remove branch protection", "tags": []string{"Protections"},
					"security":   bearerAuth(),
					"parameters": []map[string]interface{}{pathParam("id", "Repository ID"), pathParam("branch", "Branch name")},
					"responses":   statusResponse(),
				},
			},
			"/api/v1/repos/{id}/star": map[string]interface{}{
				"post": map[string]interface{}{
					"summary": "Star repository", "tags": []string{"Repos"},
					"security":   bearerAuth(),
					"parameters": []map[string]interface{}{pathParam("id", "Repository ID")},
					"responses":   statusResponse(),
				},
			},
			"/api/v1/repos/{id}/unstar": map[string]interface{}{
				"post": map[string]interface{}{
					"summary": "Unstar repository", "tags": []string{"Repos"},
					"security":   bearerAuth(),
					"parameters": []map[string]interface{}{pathParam("id", "Repository ID")},
					"responses":   statusResponse(),
				},
			},
			"/api/v1/repos/{id}/push": map[string]interface{}{
				"post": map[string]interface{}{
					"summary":     "Push objects",
					"description": "Push ref updates with packfile (requires auth)",
					"tags":        []string{"Git"},
					"security":    bearerAuth(),
					"parameters":  []map[string]interface{}{pathParam("id", "Repository ID")},
					"requestBody": jsonBody(map[string]interface{}{
						"packfile":    "<binary>",
						"ref_updates": []map[string]interface{}{{"name": "refs/heads/main", "old_hash": "0000000000000000000000000000000000000000", "new_hash": "abc123"}},
					}),
					"responses": statusResponse(),
				},
			},
			"/api/v1/repos/{id}/fork": map[string]interface{}{
				"post": map[string]interface{}{
					"summary": "Fork repository", "tags": []string{"Repos"},
					"security":   bearerAuth(),
					"parameters": []map[string]interface{}{pathParam("id", "Repository ID")},
					"requestBody": jsonBody(map[string]interface{}{"name": "my-fork"}),
					"responses":   statusResponse(),
				},
			},
			"/api/v1/repos/{id}/branches": map[string]interface{}{
				"post": map[string]interface{}{
					"summary": "Create branch", "tags": []string{"Repos"},
					"security":   bearerAuth(),
					"parameters": []map[string]interface{}{pathParam("id", "Repository ID")},
					"requestBody": jsonBody(map[string]interface{}{"name": "feature-branch", "commit": "abc123"}),
					"responses":   statusResponse(),
				},
			},
			"/api/v1/activity": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Get activity feed", "tags": []string{"Activity"},
					"parameters": []map[string]interface{}{
						queryParam("limit", "Max events to return", false),
					},
					"responses": statusResponse(),
				},
			},
			"/api/v1/agents": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "List agents", "tags": []string{"Agents"},
					"responses": statusResponse(),
				},
			},
			"/api/v1/agents/generate-did": map[string]interface{}{
				"post": map[string]interface{}{
					"summary": "Generate DID", "tags": []string{"Agents"},
					"security":  bearerAuth(),
					"responses": statusResponse(),
				},
			},
			"/api/v1/agents/resolve/{did}": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Resolve DID", "tags": []string{"Agents"},
					"parameters": []map[string]interface{}{pathParam("did", "DID to resolve")},
					"responses":  statusResponse(),
				},
			},
			"/api/v1/agents/{did}": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Get agent", "tags": []string{"Agents"},
					"parameters": []map[string]interface{}{pathParam("did", "Agent DID")},
					"responses":  statusResponse(),
				},
			},
			"/api/v1/agents/{did}/delegate": map[string]interface{}{
				"post": map[string]interface{}{
					"summary":     "Delegate capability",
					"description": "Delegate UCAN capability to another agent",
					"tags":        []string{"Agents"},
					"security":    bearerAuth(),
					"parameters":  []map[string]interface{}{pathParam("did", "Audience DID")},
					"requestBody": jsonBody(map[string]interface{}{
						"resource": "repo:*",
						"actions":  []string{"read", "write"},
					}),
					"responses": statusResponse(),
				},
			},
			"/api/v1/webhooks": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "List webhooks", "tags": []string{"Webhooks"},
					"responses": statusResponse(),
				},
				"post": map[string]interface{}{
					"summary":     "Register webhook",
					"description": "Register a webhook with HMAC-SHA256 signature verification",
					"tags":        []string{"Webhooks"},
					"security":    bearerAuth(),
					"requestBody": jsonBody(map[string]interface{}{
						"url":    "https://example.com/webhook",
						"events": []string{"issue.created", "pr.merged"},
						"secret": "optional-secret-for-hmac",
					}),
					"responses": statusResponse(),
				},
			},
			"/api/v1/webhooks/{id}": map[string]interface{}{
				"delete": map[string]interface{}{
					"summary": "Delete webhook", "tags": []string{"Webhooks"},
					"security":   bearerAuth(),
					"parameters": []map[string]interface{}{pathParam("id", "Webhook ID")},
					"responses":   statusResponse(),
				},
			},
			"/api/v1/ucan/revoke": map[string]interface{}{
				"post": map[string]interface{}{
					"summary": "Revoke UCAN", "tags": []string{"UCAN"},
					"security":   bearerAuth(),
					"requestBody": jsonBody(map[string]interface{}{"nonce": "ucan-nonce-to-revoke"}),
					"responses":   statusResponse(),
				},
			},
			"/api/v1/ucan/revocations": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "List revocations", "tags": []string{"UCAN"},
					"responses": statusResponse(),
				},
			},
		},
		"components": map[string]interface{}{
			"securitySchemes": map[string]interface{}{
				"bearerAuth": map[string]interface{}{
				"type":         "http",
					"scheme":       "bearer",
					"bearerFormat": "UCAN",
					"description":  "UCAN token obtained via /api/v1/agents/{did}/delegate",
				},
			},
		},
		"tags": []map[string]string{
			{"name": "System", "description": "Health, status, and metrics"},
			{"name": "Repos", "description": "Repository CRUD and git operations"},
			{"name": "Issues", "description": "Issue tracking"},
			{"name": "PRs", "description": "Pull request workflow"},
			{"name": "Files", "description": "File browsing and search"},
			{"name": "Commits", "description": "Commit history and diffs"},
			{"name": "Labels", "description": "Issue/PR labeling"},
			{"name": "Tasks", "description": "Task board"},
			{"name": "Releases", "description": "Tagged releases"},
			{"name": "Protections", "description": "Branch protection rules"},
			{"name": "Agents", "description": "Agent/DID management"},
			{"name": "Webhooks", "description": "Webhook registration"},
			{"name": "UCAN", "description": "UCAN token management"},
			{"name": "Git", "description": "Git smart HTTP protocol"},
			{"name": "Activity", "description": "Activity feed"},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	json.NewEncoder(w).Encode(spec)
}

func jsonContent(example interface{}) map[string]interface{} {
	return map[string]interface{}{
		"application/json": map[string]interface{}{
			"example": example,
		},
	}
}

// successResponse returns a 200 response with an example JSON body.
func successResponse(example interface{}) map[string]interface{} {
	return map[string]interface{}{
		"200": map[string]interface{}{
			"description": "Success",
			"content":     jsonContent(example),
		},
		"400": map[string]interface{}{"description": "Bad request / validation error"},
		"401": map[string]interface{}{"description": "Unauthorized (missing or invalid token)"},
		"404": map[string]interface{}{"description": "Not found"},
		"500": map[string]interface{}{"description": "Internal server error"},
	}
}


func statusResponse() map[string]interface{} {
	return successResponse(map[string]interface{}{})
}

func bearerAuth() []map[string]interface{} {
	return []map[string]interface{}{{"bearerAuth": []string{}}}
}

func pathParam(name, desc string) map[string]interface{} {
	return map[string]interface{}{
		"name":     name,
		"in":       "path",
		"required": true,
		"schema":   map[string]string{"type": "string"},
		"description": desc,
	}
}

func queryParam(name, desc string, required bool) map[string]interface{} {
	return map[string]interface{}{
		"name":        name,
		"in":          "query",
		"required":    required,
		"schema":      map[string]string{"type": "string"},
		"description": desc,
	}
}

func paginationParams() []map[string]interface{} {
	return []map[string]interface{}{
		queryParam("offset", "Pagination offset", false),
		queryParam("limit", "Page size (default 50, max 200)", false),
	}
}

func jsonBody(example interface{}) map[string]interface{} {
	return map[string]interface{}{
		"required": true,
		"content": map[string]interface{}{
			"application/json": map[string]interface{}{
				"example": example,
			},
		},
	}
}
