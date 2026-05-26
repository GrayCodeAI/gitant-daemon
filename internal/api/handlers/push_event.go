package handlers

import (
	"github.com/lakshmanpatel/gitant/internal/network"
	"github.com/lakshmanpatel/gitant/internal/webhooks"
)

func dispatchPushEvent(wm *webhooks.Manager, repoID string, objectHashes []string, refHeads map[string]string) {
	wm.Dispatch(webhooks.Event{
		Type: webhooks.EventPush,
		Repo: repoID,
		Data: network.PushEventData(objectHashes, refHeads),
	})
}
