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
					"description": "Creates a new repository",
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
					"description": "Creates a new issue in a repository",
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
			"/api/v1/notifications": map[string]interface{}{
				"get": map[string]interface{}{
					"summary":     "List notifications",
					"description": "Returns notifications for the authenticated user",
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "List of notifications",
						},
					},
				},
			},
			"/api/v1/packages": map[string]interface{}{
				"get": map[string]interface{}{
					"summary":     "List packages",
					"description": "Returns a list of all packages",
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "List of packages",
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
