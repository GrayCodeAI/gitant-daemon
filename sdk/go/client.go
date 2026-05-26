package gitant

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client is the Gitant API client
type Client struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

// NewClient creates a new Gitant API client
func NewClient(baseURL string, token string) *Client {
	return &Client{
		baseURL: baseURL,
		token:   token,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Status represents the daemon status
type Status struct {
	Version  string `json:"version"`
	Peers    int    `json:"peers"`
	Repos    int    `json:"repos"`
	Agents   int    `json:"agents"`
	Uptime   string `json:"uptime"`
	Identity string `json:"identity"`
}

// User represents a user
type User struct {
	ID          string `json:"id"`
	Username    string `json:"username"`
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
	AvatarURL   string `json:"avatar_url"`
	Role        string `json:"role"`
	CreatedAt   string `json:"created_at"`
}

// Repo represents a repository
type Repo struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Private     bool   `json:"private"`
	CreatedAt   string `json:"created_at"`
}

// Issue represents an issue
type Issue struct {
	ID        string   `json:"id"`
	Title     string   `json:"title"`
	Body      string   `json:"body"`
	Status    string   `json:"status"`
	Author    string   `json:"author"`
	Labels    []string `json:"labels"`
	Assignee  string   `json:"assignee"`
	CreatedAt string   `json:"created_at"`
	UpdatedAt string   `json:"updated_at"`
}

// PullRequest represents a pull request
type PullRequest struct {
	ID           string   `json:"id"`
	Title        string   `json:"title"`
	Body         string   `json:"body"`
	Status       string   `json:"status"`
	Author       string   `json:"author"`
	SourceBranch string   `json:"source_branch"`
	TargetBranch string   `json:"target_branch"`
	Labels       []string `json:"labels"`
	Assignee     string   `json:"assignee"`
	Reviewers    []string `json:"reviewers"`
	CreatedAt    string   `json:"created_at"`
	UpdatedAt    string   `json:"updated_at"`
}

// GetStatus gets the daemon status
func (c *Client) GetStatus() (*Status, error) {
	var status Status
	if err := c.get("/api/v1/status", &status); err != nil {
		return nil, err
	}
	return &status, nil
}

// Register registers a new user
func (c *Client) Register(username, email, password string) (*User, string, error) {
	body := map[string]string{
		"username": username,
		"email":    email,
		"password": password,
	}

	var result struct {
		User  User   `json:"user"`
		Token string `json:"token"`
	}
	if err := c.post("/api/v1/auth/register", body, &result); err != nil {
		return nil, "", err
	}
	return &result.User, result.Token, nil
}

// Login logs in a user
func (c *Client) Login(username, password string) (*User, string, error) {
	body := map[string]string{
		"username": username,
		"password": password,
	}

	var result struct {
		User  User   `json:"user"`
		Token string `json:"token"`
	}
	if err := c.post("/api/v1/auth/login", body, &result); err != nil {
		return nil, "", err
	}
	return &result.User, result.Token, nil
}

// ListRepos lists all repositories
func (c *Client) ListRepos() ([]Repo, error) {
	var result struct {
		Repos []Repo `json:"repos"`
	}
	if err := c.get("/api/v1/repos", &result); err != nil {
		return nil, err
	}
	return result.Repos, nil
}

// CreateRepo creates a new repository
func (c *Client) CreateRepo(name, description string, private bool) (*Repo, error) {
	body := map[string]interface{}{
		"name":        name,
		"description": description,
		"private":     private,
	}

	var repo Repo
	if err := c.post("/api/v1/repos", body, &repo); err != nil {
		return nil, err
	}
	return &repo, nil
}

// GetRepo gets a repository by ID
func (c *Client) GetRepo(id string) (*Repo, error) {
	var repo Repo
	if err := c.get(fmt.Sprintf("/api/v1/repos/%s", id), &repo); err != nil {
		return nil, err
	}
	return &repo, nil
}

// ListIssues lists issues for a repository
func (c *Client) ListIssues(repoID string, status string) ([]Issue, error) {
	url := fmt.Sprintf("/api/v1/repos/%s/issues", repoID)
	if status != "" {
		url += "?status=" + status
	}

	var result struct {
		Issues []Issue `json:"issues"`
	}
	if err := c.get(url, &result); err != nil {
		return nil, err
	}
	return result.Issues, nil
}

// CreateIssue creates a new issue
func (c *Client) CreateIssue(repoID, title, body string, labels []string) (*Issue, error) {
	reqBody := map[string]interface{}{
		"title":  title,
		"body":   body,
		"labels": labels,
	}

	var issue Issue
	if err := c.post(fmt.Sprintf("/api/v1/repos/%s/issues", repoID), reqBody, &issue); err != nil {
		return nil, err
	}
	return &issue, nil
}

// CloseIssue closes an issue
func (c *Client) CloseIssue(repoID, issueID string) error {
	return c.post(fmt.Sprintf("/api/v1/repos/%s/issues/%s/close", repoID, issueID), nil, nil)
}

// ListPRs lists pull requests for a repository
func (c *Client) ListPRs(repoID string, status string) ([]PullRequest, error) {
	url := fmt.Sprintf("/api/v1/repos/%s/prs", repoID)
	if status != "" {
		url += "?status=" + status
	}

	var result struct {
		PRs []PullRequest `json:"prs"`
	}
	if err := c.get(url, &result); err != nil {
		return nil, err
	}
	return result.PRs, nil
}

// CreatePR creates a new pull request
func (c *Client) CreatePR(repoID, title, body, sourceBranch, targetBranch string) (*PullRequest, error) {
	reqBody := map[string]string{
		"title":         title,
		"body":          body,
		"source_branch": sourceBranch,
		"target_branch": targetBranch,
	}

	var pr PullRequest
	if err := c.post(fmt.Sprintf("/api/v1/repos/%s/prs", repoID), reqBody, &pr); err != nil {
		return nil, err
	}
	return &pr, nil
}

// MergePR merges a pull request
func (c *Client) MergePR(repoID, prID string) error {
	return c.post(fmt.Sprintf("/api/v1/repos/%s/prs/%s/merge", repoID, prID), nil, nil)
}

// HealthCheck checks the daemon health
func (c *Client) HealthCheck() (map[string]interface{}, error) {
	var result map[string]interface{}
	if err := c.get("/health", &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) get(path string, result interface{}) error {
	req, err := http.NewRequest("GET", c.baseURL+path, nil)
	if err != nil {
		return err
	}

	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error (%d): %s", resp.StatusCode, string(body))
	}

	if result != nil {
		return json.NewDecoder(resp.Body).Decode(result)
	}
	return nil
}

func (c *Client) post(path string, body interface{}, result interface{}) error {
	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reqBody = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequest("POST", c.baseURL+path, reqBody)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error (%d): %s", resp.StatusCode, string(body))
	}

	if result != nil {
		return json.NewDecoder(resp.Body).Decode(result)
	}
	return nil
}
