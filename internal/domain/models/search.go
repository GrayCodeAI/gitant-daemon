package models

// SearchResult represents a single code search match.
type SearchResult struct {
	File    string `json:"file"`
	Line    int    `json:"line"`
	Context string `json:"context"`
}
