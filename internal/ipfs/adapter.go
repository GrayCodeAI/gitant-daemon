package ipfs

import "context"

// PinningAdapter wraps PinningStore for network.ObjectPinner.
type PinningAdapter struct {
	store *PinningStore
}

// NewPinningAdapter creates an ObjectPinner backed by the in-process pinning store.
func NewPinningAdapter(store *PinningStore) *PinningAdapter {
	return &PinningAdapter{store: store}
}

func (a *PinningAdapter) PinGitObject(ctx context.Context, repoID, hash string, data []byte) (string, error) {
	if a == nil || a.store == nil {
		return "", nil
	}
	return a.store.PinGitObject(ctx, repoID, hash, data)
}

func (a *PinningAdapter) IsPinned(repoID, hash string) bool {
	if a == nil || a.store == nil {
		return false
	}
	return a.store.IsPinned(repoID, hash)
}

// PinCount returns the number of pinned git objects.
func (a *PinningAdapter) PinCount() int {
	if a == nil || a.store == nil {
		return 0
	}
	return a.store.PinCount()
}

// Store returns the underlying pinning store.
func (a *PinningAdapter) Store() *PinningStore {
	if a == nil {
		return nil
	}
	return a.store
}
