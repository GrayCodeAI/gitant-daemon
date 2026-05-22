package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

// Client is an HTTP client for the gitant daemon API
type Client struct {
	BaseURL string
}

// NewClient creates a new daemon client
// Priority: --daemon-url flag > GITANT_DAEMON_URL env > default localhost:7777
func NewClient(url string) *Client {
	if url == "" {
		url = os.Getenv("GITANT_DAEMON_URL")
	}
	if url == "" {
		url = "http://localhost:7777"
	}
	return &Client{BaseURL: url}
}

// Get performs a GET request and decodes the JSON response
func (c *Client) Get(path string, result interface{}) error {
	resp, err := http.Get(c.BaseURL + path)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	if result != nil {
		if err := json.Unmarshal(body, result); err != nil {
			return fmt.Errorf("decoding response: %w", err)
		}
	}
	return nil
}

// Post performs a POST request with a JSON body
func (c *Client) Post(path string, body interface{}, result interface{}) error {
	data, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("encoding request: %w", err)
	}

	resp, err := http.Post(c.BaseURL+path, "application/json", bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return fmt.Errorf("API error %d: %s", resp.StatusCode, string(respBody))
	}

	if result != nil {
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("decoding response: %w", err)
		}
	}
	return nil
}

// Put performs a PUT request with a JSON body
func (c *Client) Put(path string, body interface{}, result interface{}) error {
	data, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("encoding request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPut, c.BaseURL+path, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return fmt.Errorf("API error %d: %s", resp.StatusCode, string(respBody))
	}

	if result != nil {
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("decoding response: %w", err)
		}
	}
	return nil
}

// Delete performs a DELETE request
func (c *Client) Delete(path string) error {
	req, err := http.NewRequest(http.MethodDelete, c.BaseURL+path, nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}
	return nil
}
