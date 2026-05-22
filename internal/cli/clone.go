package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/plumbing"
)

// Clone clones a repository from the daemon to a local directory
func Clone(daemonURL, repoID, localPath string) error {
	client := NewClient(daemonURL)

	// Get repo info with refs
	var repoInfo struct {
		ID   string `json:"id"`
		Name string `json:"name"`
		Refs []struct {
			Name string `json:"name"`
			Hash string `json:"hash"`
		} `json:"refs"`
	}
	if err := client.Get(fmt.Sprintf("/api/v1/repos/%s/clone", repoID), &repoInfo); err != nil {
		return fmt.Errorf("fetching repo info: %w", err)
	}

	// Init local repo
	repo, err := git.PlainInit(localPath, false)
	if err != nil {
		return fmt.Errorf("initializing repo: %w", err)
	}

	// For each ref, fetch the commit and all reachable objects
	seen := make(map[string]bool)
	for _, ref := range repoInfo.Refs {
		if err := fetchObjectRecursive(client, repoID, repo, ref.Hash, seen); err != nil {
			fmt.Fprintf(Stderr(), "warning: failed to fetch %s: %v\n", ref.Hash, err)
		}
	}

	// Set refs
	for _, ref := range repoInfo.Refs {
		hash := plumbing.NewHash(ref.Hash)
		refName := plumbing.ReferenceName(ref.Name)
		if err := repo.Storer.SetReference(plumbing.NewHashReference(refName, hash)); err != nil {
			fmt.Fprintf(Stderr(), "warning: failed to set ref %s: %v\n", ref.Name, err)
		}
	}

	fmt.Fprintf(Stderr(), "Cloned %s to %s (%d refs, %d objects)\n", repoID, localPath, len(repoInfo.Refs), len(seen))
	return nil
}

// fetchObjectRecursive fetches an object and all its children
func fetchObjectRecursive(client *Client, repoID string, repo *git.Repository, hashStr string, seen map[string]bool) error {
	if seen[hashStr] {
		return nil
	}
	seen[hashStr] = true

	// Fetch object from daemon
	var obj struct {
		Hash    string `json:"hash"`
		Type    string `json:"type"`
		Content []byte `json:"content"`
	}
	if err := client.Get(fmt.Sprintf("/api/v1/repos/%s/objects/%s", repoID, hashStr), &obj); err != nil {
		return err
	}

	// Write object to local storer
	objType := plumbing.AnyObject
	switch obj.Type {
	case "commit":
		objType = plumbing.CommitObject
	case "tree":
		objType = plumbing.TreeObject
	case "blob":
		objType = plumbing.BlobObject
	case "tag":
		objType = plumbing.TagObject
	}

	enc := repo.Storer.NewEncodedObject()
	enc.SetType(objType)
	enc.SetSize(int64(len(obj.Content)))
	writer, err := enc.Writer()
	if err != nil {
		return err
	}
	writer.Write(obj.Content)
	if _, err := repo.Storer.SetEncodedObject(enc); err != nil {
		return err
	}

	// Walk children based on object type
	if objType == plumbing.CommitObject {
		// Parse commit to find tree and parent hashes
		contentStr := string(obj.Content)
		for _, line := range strings.Split(contentStr, "\n") {
			if strings.HasPrefix(line, "tree ") {
				treeHash := strings.TrimPrefix(line, "tree ")
				fetchObjectRecursive(client, repoID, repo, treeHash, seen)
			} else if strings.HasPrefix(line, "parent ") {
				parentHash := strings.TrimPrefix(line, "parent ")
				fetchObjectRecursive(client, repoID, repo, parentHash, seen)
			}
		}
	} else if objType == plumbing.TreeObject {
		// Parse tree entries: "<mode> <name>\0<20-byte-hash>" repeated
		content := obj.Content
		i := 0
		for i < len(content) {
			nullIdx := -1
			for j := i; j < len(content); j++ {
				if content[j] == 0 {
					nullIdx = j
					break
				}
			}
			if nullIdx == -1 || nullIdx+21 > len(content) {
				break
			}
			entryHash := fmt.Sprintf("%x", content[nullIdx+1:nullIdx+21])
			fetchObjectRecursive(client, repoID, repo, entryHash, seen)
			i = nullIdx + 21
		}
	}

	return nil
}

// Pull fetches latest changes from the daemon
func Pull(repoPath, daemonURL, repoID string) error {
	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return fmt.Errorf("opening git repo: %w", err)
	}

	client := NewClient(daemonURL)

	var repoInfo struct {
		Refs []struct {
			Name string `json:"name"`
			Hash string `json:"hash"`
		} `json:"refs"`
	}
	if err := client.Get(fmt.Sprintf("/api/v1/repos/%s/clone", repoID), &repoInfo); err != nil {
		return fmt.Errorf("fetching remote refs: %w", err)
	}

	// Fetch objects for new refs
	seen := make(map[string]bool)
	for _, ref := range repoInfo.Refs {
		fetchObjectRecursive(client, repoID, repo, ref.Hash, seen)
	}

	updated := 0
	for _, ref := range repoInfo.Refs {
		hash := plumbing.NewHash(ref.Hash)
		refName := plumbing.ReferenceName(ref.Name)
		if err := repo.Storer.SetReference(plumbing.NewHashReference(refName, hash)); err != nil {
			fmt.Fprintf(Stderr(), "warning: failed to update ref %s: %v\n", ref.Name, err)
		} else {
			updated++
		}
	}

	fmt.Fprintf(Stderr(), "Updated %d ref(s), fetched %d object(s)\n", updated, len(seen))
	return nil
}

// Stderr returns stderr for output (allows testing override)
var Stderr = func() *os.File { return os.Stderr }
