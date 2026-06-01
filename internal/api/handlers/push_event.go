package handlers

import (
	"github.com/lakshmanpatel/gitant/internal/network"
	"github.com/lakshmanpatel/gitant/internal/webhooks"
)

// OnRepoChanged, if set, is invoked whenever a repository's contents change
// (e.g. on push). The server wires this to the search index's Invalidate so the
// next search rebuilds the cache. It is a single package-level hook rather than
// webhooks.SetEventHook (which has only one slot, already used for P2P sync, and
// is only registered when P2P is enabled).
var OnRepoChanged func(repoID string)

func dispatchPushEvent(wm *webhooks.Manager, repoID string, objectHashes []string, refHeads map[string]string) {
	if OnRepoChanged != nil {
		OnRepoChanged(repoID)
	}
	wm.Dispatch(webhooks.Event{
		Type: webhooks.EventPush,
		Repo: repoID,
		Data: network.PushEventData(objectHashes, refHeads),
	})
}
