package smartlog

import (
	"fmt"
	"strings"
)

// Node represents a commit node in the smartlog
type Node struct {
	Hash      string   `json:"hash"`
	ShortHash string   `json:"short_hash"`
	Message   string   `json:"message"`
	Author    string   `json:"author"`
	Branch    string   `json:"branch"`
	Parents   []string `json:"parents"`
	Children  []string `json:"children"`
	Current   bool     `json:"current"`
	Age       string   `json:"age"`
}

// Graph represents the commit graph
type Graph struct {
	Nodes []*Node `json:"nodes"`
}

// RenderSmartlog renders a smartlog visualization
func RenderSmartlog(graph *Graph) string {
	if len(graph.Nodes) == 0 {
		return "No commits found."
	}

	var sb strings.Builder
	sb.WriteString("Smartlog:\n\n")

	// Build adjacency list
	children := make(map[string][]string)
	parents := make(map[string][]string)
	nodeMap := make(map[string]*Node)

	for _, node := range graph.Nodes {
		nodeMap[node.Hash] = node
		for _, p := range node.Parents {
			children[p] = append(children[p], node.Hash)
			parents[node.Hash] = append(parents[node.Hash], p)
		}
	}

	// Find root nodes (no parents)
	var roots []string
	for _, node := range graph.Nodes {
		if len(parents[node.Hash]) == 0 {
			roots = append(roots, node.Hash)
		}
	}

	// Render tree
	visited := make(map[string]bool)
	for _, root := range roots {
		renderNode(&sb, nodeMap, children, root, "", visited)
	}

	return sb.String()
}

func renderNode(sb *strings.Builder, nodeMap map[string]*Node, children map[string][]string, hash, prefix string, visited map[string]bool) {
	if visited[hash] {
		return
	}
	visited[hash] = true

	node, ok := nodeMap[hash]
	if !ok {
		return
	}

	// Render current node
	marker := "○"
	if node.Current {
		marker = "◉"
	}

	fmt.Fprintf(sb, "%s%s %s %s (%s)\n", prefix, marker, node.ShortHash, node.Message, node.Age)

	// Render children
	childHashes := children[hash]
	for i, childHash := range childHashes {
		childPrefix := prefix
		if i < len(childHashes)-1 {
			childPrefix += "│ "
		} else {
			childPrefix += "  "
		}

		fmt.Fprintf(sb, "%s│\n", prefix)
		renderNode(sb, nodeMap, children, childHash, childPrefix, visited)
	}
}

// BuildGraph builds a graph from commits
func BuildGraph(commits []struct {
	Hash    string
	Message string
	Author  string
	Branch  string
	Parents []string
}) *Graph {
	nodes := make([]*Node, len(commits))
	childMap := make(map[string][]string)

	for i, commit := range commits {
		shortHash := commit.Hash
		if len(shortHash) > 7 {
			shortHash = shortHash[:7]
		}

		nodes[i] = &Node{
			Hash:      commit.Hash,
			ShortHash: shortHash,
			Message:   commit.Message,
			Author:    commit.Author,
			Branch:    commit.Branch,
			Parents:   commit.Parents,
		}

		for _, p := range commit.Parents {
			childMap[p] = append(childMap[p], commit.Hash)
		}
	}

	// Add children references
	for _, node := range nodes {
		node.Children = childMap[node.Hash]
	}

	return &Graph{Nodes: nodes}
}
