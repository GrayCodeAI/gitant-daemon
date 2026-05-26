package arweave

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Anchor represents an Arweave anchor
type Anchor struct {
	ID         string    `json:"id"`
	TxID       string    `json:"tx_id"`
	RepoID     string    `json:"repo_id"`
	CommitHash string    `json:"commit_hash"`
	DataHash   string    `json:"data_hash"`
	Timestamp  time.Time `json:"timestamp"`
	URL        string    `json:"url"`
}

// Client provides Arweave operations via Irys
type Client struct {
	endpoint   string
	httpClient *http.Client
}

// NewClient creates a new Arweave client
func NewClient(endpoint string) *Client {
	return &Client{
		endpoint: endpoint,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Upload uploads data to Arweave via Irys
func (c *Client) Upload(data []byte, tags map[string]string) (*Anchor, error) {
	body := map[string]interface{}{
		"data": data,
		"tags": tags,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshaling body: %w", err)
	}

	resp, err := c.httpClient.Post(
		fmt.Sprintf("%s/tx", c.endpoint),
		"application/json",
		bytes.NewReader(jsonBody),
	)
	if err != nil {
		return nil, fmt.Errorf("uploading to arweave: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("arweave error (%d): %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		ID  string `json:"id"`
		URL string `json:"url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	anchor := &Anchor{
		ID:        fmt.Sprintf("anchor-%d", time.Now().UnixNano()),
		TxID:      result.ID,
		Timestamp: time.Now(),
		URL:       result.URL,
	}

	return anchor, nil
}

// Get retrieves data from Arweave
func (c *Client) Get(txID string) ([]byte, error) {
	resp, err := c.httpClient.Get(fmt.Sprintf("%s/tx/%s/data", c.endpoint, txID))
	if err != nil {
		return nil, fmt.Errorf("getting from arweave: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("arweave error (%d)", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

// GetStatus gets the status of a transaction
func (c *Client) GetStatus(txID string) (string, error) {
	resp, err := c.httpClient.Get(fmt.Sprintf("%s/tx/%s/status", c.endpoint, txID))
	if err != nil {
		return "", fmt.Errorf("getting status: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return "pending", nil
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("arweave error (%d)", resp.StatusCode)
	}

	var result struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decoding response: %w", err)
	}

	return result.Status, nil
}
