package handlers

import (
	"encoding/json"
	"net/http"
)

// OpenAPISpec represents the OpenAPI specification
type OpenAPISpec struct {
	OpenAPI string                 `json:"openapi"`
	Info    map[string]interface{} `json:"info"`
	Paths   map[string]interface{} `json:"paths"`
	Servers []map[string]string    `json:"servers,omitempty"`
}

// GenerateOpenAPISpec generates the OpenAPI specification
func GenerateOpenAPISpec(baseURL string) *OpenAPISpec {
	return &OpenAPISpec{
		OpenAPI: "3.0.3",
		Info: map[string]interface{}{
			"title":       "Gitant API",
			"description": "Decentralized Git hosting platform for solo developers and AI agents",
			"version":     "0.2.0",
			"contact": map[string]string{
				"name":  "Gitant",
				"url":   "https://github.com/GrayCodeAI/gitant-daemon",
				"email": "support@gitant.dev",
			},
			"license": map[string]string{
				"name": "MIT",
				"url":  "https://opensource.org/licenses/MIT",
			},
		},
		Servers: []map[string]string{
			{"url": baseURL, "description": "Local development server"},
		},
		Paths: map[string]interface{}{
			"/api/v1/status": map[string]interface{}{
				"get": map[string]interface{}{
					"summary":     "Get daemon status",
					"description": "Returns the current status of the gitant daemon",
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Status information",
						},
					},
				},
			},
			"/api/v1/repos": map[string]interface{}{
				"get": map[string]interface{}{
					"summary":     "List repositories",
					"description": "Returns a list of all repositories",
					"parameters": []map[string]interface{}{
						{"name": "offset", "in": "query", "schema": map[string]string{"type": "integer"}},
						{"name": "limit", "in": "query", "schema": map[string]string{"type": "integer"}},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "List of repositories",
						},
					},
				},
				"post": map[string]interface{}{
					"summary":     "Create repository",
					"description": "Creates a new repository for an authenticated UCAN/HTTP-signature identity or session user. Session-created repositories record the session user as owner.",
					"security":    []map[string][]string{{"bearerAuth": []string{}}},
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"type": "object",
									"properties": map[string]interface{}{
										"name":        map[string]string{"type": "string"},
										"description": map[string]string{"type": "string"},
										"private":     map[string]string{"type": "boolean"},
									},
									"required": []string{"name"},
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"201": map[string]interface{}{
							"description": "Repository created",
						},
					},
				},
			},
			"/api/v1/repos/{id}/info/refs": map[string]interface{}{
				"get": map[string]interface{}{
					"summary":     "Advertise git smart HTTP refs",
					"description": "Advertises refs for git-upload-pack or git-receive-pack. Public repositories are readable anonymously; private repositories require read access.",
					"parameters": []map[string]interface{}{
						{"name": "id", "in": "path", "required": true, "schema": map[string]string{"type": "string"}},
						{"name": "service", "in": "query", "required": true, "schema": map[string]interface{}{"type": "string", "enum": []string{"git-upload-pack", "git-receive-pack"}}},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{"description": "Git smart HTTP ref advertisement"},
						"400": map[string]interface{}{"description": "Unsupported service"},
						"404": map[string]interface{}{"description": "Repository not found or private"},
					},
				},
			},
			"/api/v1/repos/{id}/git-upload-pack": map[string]interface{}{
				"post": map[string]interface{}{
					"summary":     "Run git-upload-pack",
					"description": "Streams a packfile for fetch/clone. Public repositories are readable anonymously; private repositories require read access, not write capability.",
					"parameters": []map[string]interface{}{
						{"name": "id", "in": "path", "required": true, "schema": map[string]string{"type": "string"}},
					},
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/x-git-upload-pack-request": map[string]interface{}{"schema": map[string]string{"type": "string", "format": "binary"}},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{"description": "Git upload-pack result stream"},
						"400": map[string]interface{}{"description": "Malformed upload-pack request"},
						"404": map[string]interface{}{"description": "Repository not found or private"},
					},
				},
			},
			"/api/v1/repos/{id}/git-receive-pack": map[string]interface{}{
				"post": map[string]interface{}{
					"summary":     "Run git-receive-pack",
					"description": "Accepts push packfiles and updates refs. Requires repository write capability via a verified UCAN or session owner/collaborator membership.",
					"security":    []map[string][]string{{"bearerAuth": []string{"repo:write"}}},
					"parameters": []map[string]interface{}{
						{"name": "id", "in": "path", "required": true, "schema": map[string]string{"type": "string"}},
					},
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/x-git-receive-pack-request": map[string]interface{}{"schema": map[string]string{"type": "string", "format": "binary"}},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{"description": "Git receive-pack result stream"},
						"401": map[string]interface{}{"description": "Authentication required"},
						"403": map[string]interface{}{"description": "Write capability or session owner/collaborator membership required"},
						"404": map[string]interface{}{"description": "Repository not found or private"},
					},
				},
			},
			"/api/v1/repos/{id}/issues": map[string]interface{}{
				"get": map[string]interface{}{
					"summary":     "List issues",
					"description": "Returns a list of issues for a repository",
					"parameters": []map[string]interface{}{
						{"name": "id", "in": "path", "required": true, "schema": map[string]string{"type": "string"}},
						{"name": "status", "in": "query", "schema": map[string]string{"type": "string"}},
						{"name": "labels", "in": "query", "schema": map[string]string{"type": "string"}},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "List of issues",
						},
					},
				},
				"post": map[string]interface{}{
					"summary":     "Create issue",
					"description": "Creates a new issue in a repository. Requires repository write capability via a verified UCAN or session owner/collaborator membership.",
					"security":    []map[string][]string{{"bearerAuth": []string{"repo:write"}}},
					"parameters": []map[string]interface{}{
						{"name": "id", "in": "path", "required": true, "schema": map[string]string{"type": "string"}},
					},
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"type": "object",
									"properties": map[string]interface{}{
										"title":  map[string]string{"type": "string"},
										"body":   map[string]string{"type": "string"},
										"labels": map[string]interface{}{"type": "array", "items": map[string]string{"type": "string"}},
									},
									"required": []string{"title"},
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"201": map[string]interface{}{
							"description": "Issue created",
						},
					},
				},
			},
			"/api/v1/repos/{id}/prs": map[string]interface{}{
				"get": map[string]interface{}{
					"summary":     "List pull requests",
					"description": "Returns a list of pull requests for a repository",
					"parameters": []map[string]interface{}{
						{"name": "id", "in": "path", "required": true, "schema": map[string]string{"type": "string"}},
						{"name": "status", "in": "query", "schema": map[string]string{"type": "string"}},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "List of pull requests",
						},
					},
				},
			},
			"/api/v1/auth/register": map[string]interface{}{
				"post": map[string]interface{}{
					"summary":     "Register user",
					"description": "Registers a new user account",
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"type": "object",
									"properties": map[string]interface{}{
										"username": map[string]string{"type": "string"},
										"email":    map[string]string{"type": "string"},
										"password": map[string]string{"type": "string"},
									},
									"required": []string{"username", "email", "password"},
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"201": map[string]interface{}{
							"description": "User registered",
						},
					},
				},
			},
			"/api/v1/auth/login": map[string]interface{}{
				"post": map[string]interface{}{
					"summary":     "Login",
					"description": "Authenticates a user and returns a session token",
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"type": "object",
									"properties": map[string]interface{}{
										"username": map[string]string{"type": "string"},
										"password": map[string]string{"type": "string"},
									},
									"required": []string{"username", "password"},
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Login successful",
						},
					},
				},
			},
			"/health": map[string]interface{}{
				"get": map[string]interface{}{
					"summary":     "Health check",
					"description": "Returns the health status of the daemon",
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Health status",
						},
					},
				},
			},
		},
	}
}

// HandleOpenAPI serves the OpenAPI specification
func HandleOpenAPI(w http.ResponseWriter, r *http.Request) {
	baseURL := "http://localhost:7777"
	spec := GenerateOpenAPISpec(baseURL)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(spec)
}
