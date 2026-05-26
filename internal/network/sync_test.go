package network

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/go-git/go-git/v6/plumbing"
)

type memoryObjectStore struct {
	mu      sync.RWMutex
	objects map[string]map[string]memoryObject
}

type memoryObject struct {
	objType string
	data    []byte
}

func newMemoryObjectStore() *memoryObjectStore {
	return &memoryObjectStore{objects: make(map[string]map[string]memoryObject)}
}

func (s *memoryObjectStore) HasObject(repoID, hash string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	repo, ok := s.objects[repoID]
	if !ok {
		return false
	}
	_, ok = repo[hash]
	return ok
}

func (s *memoryObjectStore) GetObject(repoID, hash string) (string, []byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	repo, ok := s.objects[repoID]
	if !ok {
		return "", nil, errObjectNotFound
	}
	obj, ok := repo[hash]
	if !ok {
		return "", nil, errObjectNotFound
	}
	return obj.objType, append([]byte(nil), obj.data...), nil
}

func (s *memoryObjectStore) PutObject(repoID, hash, objType string, data []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.objects[repoID]; !ok {
		s.objects[repoID] = make(map[string]memoryObject)
	}
	s.objects[repoID][hash] = memoryObject{objType: objType, data: append([]byte(nil), data...)}
	return nil
}

var errObjectNotFound = &objectNotFoundError{}

type objectNotFoundError struct{}

func (e *objectNotFoundError) Error() string { return "object not found" }

func TestBlockExchangeBetweenPeers(t *testing.T) {
	ctx := context.Background()

	providerStore := newMemoryObjectStore()
	hash := plumbing.NewHash("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa").String()
	if err := providerStore.PutObject("demo", hash, "blob", []byte("hello")); err != nil {
		t.Fatal(err)
	}

	node1, err := StartNode(ctx, NodeConfig{
		ListenAddr: "/ip4/127.0.0.1/tcp/0",
		EnableMDNS: false,
		ServerDID:  "did:key:provider",
	})
	if err != nil {
		t.Fatal(err)
	}
	defer node1.Close()
	_ = NewSyncCoordinator(node1, providerStore, nil, nil, nil)

	node2, err := StartNode(ctx, NodeConfig{
		ListenAddr: "/ip4/127.0.0.1/tcp/0",
		EnableMDNS: false,
		ServerDID:  "did:key:consumer",
	})
	if err != nil {
		t.Fatal(err)
	}
	defer node2.Close()

	consumerStore := newMemoryObjectStore()
	_ = NewSyncCoordinator(node2, consumerStore, nil, nil, nil)

	if err := node2.Host.Connect(ctx, node1.AdvertisedAddrs()[0]); err != nil {
		t.Fatal(err)
	}
	time.Sleep(200 * time.Millisecond)

	if err := node2.SyncObjects(ctx, consumerStore, node1.Host.ID(), "demo", []string{hash}); err != nil {
		t.Fatal(err)
	}
	if !consumerStore.HasObject("demo", hash) {
		t.Fatal("expected replicated object on consumer node")
	}
}

func TestMergeBootstrapPeers(t *testing.T) {
	t.Setenv("GITANT_SEED_PEERS", "/ip4/1.1.1.1/tcp/4001/p2p/abc")
	merged := MergeBootstrapPeers([]string{"/ip4/2.2.2.2/tcp/4001/p2p/def", "/ip4/1.1.1.1/tcp/4001/p2p/abc"})
	if len(merged) != 2 {
		t.Fatalf("expected 2 unique bootstrap peers, got %d", len(merged))
	}
}
