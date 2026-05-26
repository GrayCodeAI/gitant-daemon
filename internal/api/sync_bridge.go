package api

import (
	"fmt"

	"github.com/go-git/go-git/v6/plumbing"
	"github.com/lakshmanpatel/gitant/internal/crdt"
	"github.com/lakshmanpatel/gitant/internal/network"
	"github.com/lakshmanpatel/gitant/internal/storage"
	"github.com/lakshmanpatel/gitant/internal/api/handlers"
)

type repoObjectStore struct {
	repos *storage.RepositoryRegistry
}

func newRepoObjectStore(repos *storage.RepositoryRegistry) network.ObjectStore {
	return &repoObjectStore{repos: repos}
}

func (s *repoObjectStore) HasObject(repoID, hash string) bool {
	repo, err := s.repos.Open(repoID)
	if err != nil {
		return false
	}
	return repo.HasObject(plumbing.NewHash(hash))
}

func (s *repoObjectStore) GetObject(repoID, hash string) (string, []byte, error) {
	repo, err := s.repos.Open(repoID)
	if err != nil {
		return "", nil, err
	}
	objType, data, err := repo.GetObject(plumbing.NewHash(hash))
	if err != nil {
		return "", nil, err
	}
	return objectTypeToString(objType), data, nil
}

func (s *repoObjectStore) PutObject(repoID, hash, objType string, data []byte) error {
	repo, err := s.repos.Open(repoID)
	if err != nil {
		return err
	}
	plumbingType, err := stringToObjectType(objType)
	if err != nil {
		return err
	}
	return repo.StoreObject(plumbing.NewHash(hash), plumbingType, data)
}

type crdtSyncStore struct {
	issues *crdt.IssueStore
	prs    *crdt.PullRequestStore
}

func newCRDTSyncStore(issues *crdt.IssueStore, prs *crdt.PullRequestStore) network.CRDTStore {
	return &crdtSyncStore{issues: issues, prs: prs}
}

type agentTrustStore struct {
	agents *handlers.AgentRegistry
}

func newAgentTrustStore(agents *handlers.AgentRegistry) network.TrustStore {
	return &agentTrustStore{agents: agents}
}

func (s *agentTrustStore) ApplyAttestation(sourceDID, targetDID string, score float64) error {
	return s.agents.ApplyAttestation(sourceDID, targetDID, score)
}

func (s *crdtSyncStore) MergeIssue(repoID string, issue *crdt.Issue) error {
	return s.issues.MergeRemote(repoID, issue)
}

func (s *crdtSyncStore) MergePR(repoID string, pr *crdt.PullRequest) error {
	return s.prs.MergeRemote(repoID, pr)
}

func objectTypeToString(objType plumbing.ObjectType) string {
	switch objType {
	case plumbing.BlobObject:
		return "blob"
	case plumbing.TreeObject:
		return "tree"
	case plumbing.CommitObject:
		return "commit"
	case plumbing.TagObject:
		return "tag"
	default:
		return "blob"
	}
}

func stringToObjectType(value string) (plumbing.ObjectType, error) {
	switch value {
	case "blob", "":
		return plumbing.BlobObject, nil
	case "tree":
		return plumbing.TreeObject, nil
	case "commit":
		return plumbing.CommitObject, nil
	case "tag":
		return plumbing.TagObject, nil
	default:
		return 0, fmt.Errorf("unsupported object type: %s", value)
	}
}
