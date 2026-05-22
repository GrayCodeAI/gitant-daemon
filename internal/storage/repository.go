package storage

import (
	"fmt"
	"time"

	"github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/plumbing"
	"github.com/go-git/go-git/v6/plumbing/filemode"
	"github.com/go-git/go-git/v6/plumbing/object"
)

type Repository struct {
	repo *git.Repository
	path string
}

// OpenRepository opens an existing git repository
func OpenRepository(path string) (*Repository, error) {
	repo, err := git.PlainOpen(path)
	if err != nil {
		return nil, fmt.Errorf("opening repository: %w", err)
	}

	return &Repository{
		repo: repo,
		path: path,
	}, nil
}

// InitRepository creates a new git repository
func InitRepository(path string) (*Repository, error) {
	repo, err := git.PlainInit(path, false)
	if err != nil {
		return nil, fmt.Errorf("initializing repository: %w", err)
	}

	return &Repository{
		repo: repo,
		path: path,
	}, nil
}

// CreateBlob creates a new blob object from content
func (r *Repository) CreateBlob(content []byte) (plumbing.Hash, error) {
	obj := r.repo.Storer
	enc := obj.NewEncodedObject()
	enc.SetType(plumbing.BlobObject)
	enc.SetSize(int64(len(content)))

	writer, err := enc.Writer()
	if err != nil {
		return plumbing.ZeroHash, fmt.Errorf("getting writer: %w", err)
	}

	_, err = writer.Write(content)
	if err != nil {
		return plumbing.ZeroHash, fmt.Errorf("writing blob content: %w", err)
	}

	blobHash, err := obj.SetEncodedObject(enc)
	if err != nil {
		return plumbing.ZeroHash, fmt.Errorf("storing blob: %w", err)
	}

	return blobHash, nil
}

// GetBlob retrieves a blob by hash
func (r *Repository) GetBlob(hash plumbing.Hash) ([]byte, error) {
	blob, err := r.repo.BlobObject(hash)
	if err != nil {
		return nil, fmt.Errorf("getting blob: %w", err)
	}

	reader, err := blob.Reader()
	if err != nil {
		return nil, fmt.Errorf("reading blob: %w", err)
	}
	defer reader.Close()

	buf := make([]byte, blob.Size)
	_, err = reader.Read(buf)
	if err != nil {
		return nil, fmt.Errorf("reading blob content: %w", err)
	}

	return buf, nil
}

// CreateTree creates a new tree object from entries
func (r *Repository) CreateTree(entries []TreeEntry) (plumbing.Hash, error) {
	obj := r.repo.Storer
	enc := obj.NewEncodedObject()
	enc.SetType(plumbing.TreeObject)

	// Encode tree entries: "<mode> <name>\0<20-byte-hash>"
	var content []byte
	for _, entry := range entries {
		mode := entry.Mode.String()
		line := fmt.Sprintf("%s %s\x00", mode, entry.Name)
		content = append(content, []byte(line)...)
		// ObjectID.Bytes() returns the raw 20-byte hash
		hashBytes := entry.Hash.Bytes()
		content = append(content, hashBytes...)
	}

	enc.SetSize(int64(len(content)))
	writer, err := enc.Writer()
	if err != nil {
		return plumbing.ZeroHash, fmt.Errorf("getting writer: %w", err)
	}

	if _, err := writer.Write(content); err != nil {
		return plumbing.ZeroHash, fmt.Errorf("writing tree content: %w", err)
	}

	hash, err := obj.SetEncodedObject(enc)
	if err != nil {
		return plumbing.ZeroHash, fmt.Errorf("storing tree: %w", err)
	}

	return hash, nil
}

// GetTree retrieves a tree by hash
func (r *Repository) GetTree(hash plumbing.Hash) (*object.Tree, error) {
	return r.repo.TreeObject(hash)
}

// CreateCommit creates a new commit object
func (r *Repository) CreateCommit(treeHash plumbing.Hash, parents []plumbing.Hash, author, message string) (plumbing.Hash, error) {
	obj := r.repo.Storer
	enc := obj.NewEncodedObject()
	enc.SetType(plumbing.CommitObject)

	// Build commit body
	var body string
	body += fmt.Sprintf("tree %s\n", treeHash.String())
	for _, parent := range parents {
		body += fmt.Sprintf("parent %s\n", parent.String())
	}
	timestamp := time.Now().Unix()
	timezone := "+0000"
	body += fmt.Sprintf("author %s <%s@localhost> %d %s\n", author, author, timestamp, timezone)
	body += fmt.Sprintf("committer %s <%s@localhost> %d %s\n", author, author, timestamp, timezone)
	body += "\n" + message + "\n"

	enc.SetSize(int64(len(body)))
	writer, err := enc.Writer()
	if err != nil {
		return plumbing.ZeroHash, fmt.Errorf("getting writer: %w", err)
	}

	if _, err := writer.Write([]byte(body)); err != nil {
		return plumbing.ZeroHash, fmt.Errorf("writing commit content: %w", err)
	}

	hash, err := obj.SetEncodedObject(enc)
	if err != nil {
		return plumbing.ZeroHash, fmt.Errorf("storing commit: %w", err)
	}

	return hash, nil
}

// GetCommit retrieves a commit by hash
func (r *Repository) GetCommit(hash plumbing.Hash) (*object.Commit, error) {
	return r.repo.CommitObject(hash)
}

// ListRefs returns all references in the repository
func (r *Repository) ListRefs() ([]plumbing.Hash, error) {
	refs, err := r.repo.References()
	if err != nil {
		return nil, fmt.Errorf("listing references: %w", err)
	}

	var hashes []plumbing.Hash
	refs.ForEach(func(ref *plumbing.Reference) error {
		hashes = append(hashes, ref.Hash())
		return nil
	})

	return hashes, nil
}

// CreateBranch creates a new branch pointing to a commit
func (r *Repository) CreateBranch(name string, commitHash plumbing.Hash) error {
	refName := plumbing.NewBranchReferenceName(name)
	ref := plumbing.NewHashReference(refName, commitHash)
	return r.repo.Storer.SetReference(ref)
}

// DeleteBranch deletes a branch
func (r *Repository) DeleteBranch(name string) error {
	refName := plumbing.NewBranchReferenceName(name)
	return r.repo.Storer.RemoveReference(refName)
}

// GetBranch returns the commit hash a branch points to
func (r *Repository) GetBranch(name string) (plumbing.Hash, error) {
	refName := plumbing.NewBranchReferenceName(name)
	ref, err := r.repo.Reference(refName, true)
	if err != nil {
		return plumbing.ZeroHash, fmt.Errorf("getting branch: %w", err)
	}
	return ref.Hash(), nil
}

// TreeEntry represents an entry in a tree object
type TreeEntry struct {
	Name string
	Mode filemode.FileMode
	Hash plumbing.Hash
}

// CommitInfo represents commit metadata for API responses
type CommitInfo struct {
	Hash      string `json:"hash"`
	Author    string `json:"author"`
	Message   string `json:"message"`
	TreeHash  string `json:"tree_hash"`
	ParentHashes []string `json:"parent_hashes"`
}

// WalkCommits walks the commit history starting from a hash
func (r *Repository) WalkCommits(startHash plumbing.Hash, limit int) ([]CommitInfo, error) {
	commits := make([]CommitInfo, 0, limit)
	current := startHash

	for i := 0; i < limit; i++ {
		commit, err := r.repo.CommitObject(current)
		if err != nil {
			break
		}

		info := CommitInfo{
			Hash:    commit.Hash.String(),
			Author:  commit.Author.String(),
			Message: commit.Message,
			TreeHash: commit.TreeHash.String(),
		}
		for _, parent := range commit.ParentHashes {
			info.ParentHashes = append(info.ParentHashes, parent.String())
		}
		commits = append(commits, info)

		if len(commit.ParentHashes) == 0 {
			break
		}
		current = commit.ParentHashes[0]
	}

	return commits, nil
}

// GetFileFromTree retrieves a file's content from a tree by path
func (r *Repository) GetFileFromTree(treeHash plumbing.Hash, path string) ([]byte, error) {
	tree, err := r.repo.TreeObject(treeHash)
	if err != nil {
		return nil, fmt.Errorf("getting tree: %w", err)
	}

	entry, err := tree.FindEntry(path)
	if err != nil {
		return nil, fmt.Errorf("file not found: %s", path)
	}

	blob, err := r.repo.BlobObject(entry.Hash)
	if err != nil {
		return nil, fmt.Errorf("getting blob: %w", err)
	}

	reader, err := blob.Reader()
	if err != nil {
		return nil, fmt.Errorf("reading blob: %w", err)
	}
	defer reader.Close()

	buf := make([]byte, blob.Size)
	if _, err := reader.Read(buf); err != nil {
		return nil, fmt.Errorf("reading content: %w", err)
	}

	return buf, nil
}

// ListTreeEntries lists entries in a tree at an optional subpath
func (r *Repository) ListTreeEntries(treeHash plumbing.Hash, path string) ([]TreeEntry, error) {
	tree, err := r.repo.TreeObject(treeHash)
	if err != nil {
		return nil, fmt.Errorf("getting tree: %w", err)
	}

	// If path is specified, navigate into subdirectory
	if path != "" {
		subtree, err := tree.Tree(path)
		if err != nil {
			return nil, fmt.Errorf("subtree not found: %s", path)
		}
		tree = subtree
	}

	entries := make([]TreeEntry, 0)
	for _, entry := range tree.Entries {
		entries = append(entries, TreeEntry{
			Name: entry.Name,
			Mode: entry.Mode,
			Hash: entry.Hash,
		})
	}

	return entries, nil
}

// StoreObject stores a raw git object by hash
func (r *Repository) StoreObject(hash plumbing.Hash, objType plumbing.ObjectType, content []byte) error {
	obj := r.repo.Storer
	enc := obj.NewEncodedObject()
	enc.SetType(objType)
	enc.SetSize(int64(len(content)))

	writer, err := enc.Writer()
	if err != nil {
		return fmt.Errorf("getting writer: %w", err)
	}
	if _, err := writer.Write(content); err != nil {
		return fmt.Errorf("writing content: %w", err)
	}

	if _, err := obj.SetEncodedObject(enc); err != nil {
		return fmt.Errorf("storing object: %w", err)
	}
	return nil
}

// GetObject retrieves a raw git object by hash
func (r *Repository) GetObject(hash plumbing.Hash) (plumbing.ObjectType, []byte, error) {
	obj, err := r.repo.Storer.EncodedObject(plumbing.AnyObject, hash)
	if err != nil {
		return 0, nil, fmt.Errorf("getting object: %w", err)
	}

	reader, err := obj.Reader()
	if err != nil {
		return 0, nil, fmt.Errorf("getting reader: %w", err)
	}
	defer reader.Close()

	buf := make([]byte, obj.Size())
	if _, err := reader.Read(buf); err != nil {
		return 0, nil, fmt.Errorf("reading content: %w", err)
	}

	return obj.Type(), buf, nil
}

// ListAllRefs returns all references with their names and hashes
type RefInfo struct {
	Name string `json:"name"`
	Hash string `json:"hash"`
}

func (r *Repository) ListAllRefs() ([]RefInfo, error) {
	refs, err := r.repo.References()
	if err != nil {
		return nil, fmt.Errorf("listing references: %w", err)
	}

	var result []RefInfo
	refs.ForEach(func(ref *plumbing.Reference) error {
		result = append(result, RefInfo{
			Name: ref.Name().String(),
			Hash: ref.Hash().String(),
		})
		return nil
	})

	return result, nil
}
