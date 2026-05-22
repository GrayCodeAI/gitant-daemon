package cli

import (
	"encoding/base64"
	"fmt"

	"github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/plumbing"
	"github.com/go-git/go-git/v6/plumbing/filemode"
	"github.com/go-git/go-git/v6/plumbing/object"
	"github.com/go-git/go-git/v6/storage/memory"
)

// RefUpdate represents a ref change to push
type RefUpdate struct {
	Name    string `json:"name"`
	OldHash string `json:"old_hash"`
	NewHash string `json:"new_hash"`
}

// GitObject represents an object to transfer
type GitObject struct {
	Hash    string `json:"hash"`
	Type    string `json:"type"`
	Content string `json:"content"` // base64
}

// Push performs a push from a local repo to the daemon
func Push(repoPath, daemonURL, repoID string) error {
	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return fmt.Errorf("opening git repo: %w", err)
	}

	// Get all local refs
	refs, err := repo.References()
	if err != nil {
		return fmt.Errorf("listing refs: %w", err)
	}

	var updates []RefUpdate
	var commitHashes []plumbing.Hash
	refs.ForEach(func(ref *plumbing.Reference) error {
		if ref.Name().IsBranch() || ref.Name().IsTag() {
			updates = append(updates, RefUpdate{
				Name:    ref.Name().String(),
				NewHash: ref.Hash().String(),
			})
			commitHashes = append(commitHashes, ref.Hash())
		}
		return nil
	})

	if len(updates) == 0 {
		fmt.Println("Nothing to push")
		return nil
	}

	// Collect all reachable objects
	objects, err := collectObjects(repo, commitHashes)
	if err != nil {
		return fmt.Errorf("collecting objects: %w", err)
	}

	client := NewClient(daemonURL)
	var result struct {
		Success bool     `json:"success"`
		Repo    string   `json:"repo"`
		Errors  []string `json:"errors"`
	}

	err = client.Post(fmt.Sprintf("/api/v1/repos/%s/push", repoID), map[string]interface{}{
		"objects":     objects,
		"ref_updates": updates,
	}, &result)
	if err != nil {
		return fmt.Errorf("push failed: %w", err)
	}

	if len(result.Errors) > 0 {
		for _, e := range result.Errors {
			fmt.Fprintf(Stderr(), "warning: %s\n", e)
		}
	}

	fmt.Fprintf(Stderr(), "Pushed %d ref(s), %d object(s) to %s\n", len(updates), len(objects), repoID)
	return nil
}

// collectObjects walks the git graph from the given commits and collects all reachable objects
func collectObjects(repo *git.Repository, hashes []plumbing.Hash) ([]GitObject, error) {
	seen := make(map[string]bool)
	var objects []GitObject

	for _, hash := range hashes {
		// Walk commits
		commitIter := object.NewCommitPreorderIter(&object.Commit{Hash: hash}, nil, nil)
		commitIter.ForEach(func(c *object.Commit) error {
			if seen[c.Hash.String()] {
				return nil
			}
			seen[c.Hash.String()] = true

			// Add commit object
			obj, err := encodeObject(c.Hash, "commit", repo)
			if err == nil {
				objects = append(objects, obj)
			}

			// Add tree and blobs
			tree, err := repo.TreeObject(c.TreeHash)
			if err == nil {
				treeObjs, err := collectTreeObjects(repo, tree, seen)
				if err == nil {
					objects = append(objects, treeObjs...)
				}
			}
			return nil
		})
	}

	return objects, nil
}

// collectTreeObjects recursively collects tree and blob objects
func collectTreeObjects(repo *git.Repository, tree *object.Tree, seen map[string]bool) ([]GitObject, error) {
	var objects []GitObject

	if seen[tree.Hash.String()] {
		return objects, nil
	}
	seen[tree.Hash.String()] = true

	// Add tree object
	obj, err := encodeObject(tree.Hash, "tree", repo)
	if err == nil {
		objects = append(objects, obj)
	}

	// Walk entries
	for _, entry := range tree.Entries {
		if entry.Mode == filemode.Regular || entry.Mode == filemode.Executable {
			if !seen[entry.Hash.String()] {
				seen[entry.Hash.String()] = true
				blobObj, err := encodeObject(entry.Hash, "blob", repo)
				if err == nil {
					objects = append(objects, blobObj)
				}
			}
		} else if entry.Mode == filemode.Dir {
			subtree, err := repo.TreeObject(entry.Hash)
			if err == nil {
				subObjs, err := collectTreeObjects(repo, subtree, seen)
				if err == nil {
					objects = append(objects, subObjs...)
				}
			}
		}
	}

	return objects, nil
}

// encodeObject reads a git object and encodes it for transfer
func encodeObject(hash plumbing.Hash, objType string, repo *git.Repository) (GitObject, error) {
	obj, err := repo.Storer.EncodedObject(plumbing.AnyObject, hash)
	if err != nil {
		return GitObject{}, err
	}

	reader, err := obj.Reader()
	if err != nil {
		return GitObject{}, err
	}
	defer reader.Close()

	buf := make([]byte, obj.Size())
	if _, err := reader.Read(buf); err != nil {
		return GitObject{}, err
	}

	return GitObject{
		Hash:    hash.String(),
		Type:    objType,
		Content: base64.StdEncoding.EncodeToString(buf),
	}, nil
}

// Ensure memory import is used (for go-git dependency)
var _ = memory.NewStorage()
