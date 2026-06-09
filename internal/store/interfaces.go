package store

import (
	"context"
	"time"
)

// IssueStore defines the interface for issue storage
type IssueStore interface {
	Create(ctx context.Context, repoID, id, author, title, body string) (*Issue, error)
	Get(ctx context.Context, repoID, issueID string) (*Issue, error)
	List(ctx context.Context, repoID string, filters IssueFilters) ([]*Issue, error)
	Update(ctx context.Context, repoID, issueID string, fn func(*Issue) error) error
	Delete(ctx context.Context, repoID, issueID string) error
	Save() error
}

// IssueFilters for listing issues
type IssueFilters struct {
	Status string
	Labels []string
}

// Issue represents an issue
type Issue struct {
	ID        string
	Title     string
	Body      string
	Status    string
	Author    string
	Labels    []string
	Assignee  string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// PullRequestStore defines the interface for PR storage
type PullRequestStore interface {
	Create(ctx context.Context, repoID, id, author, title, body, sourceBranch, targetBranch string) (*PullRequest, error)
	Get(ctx context.Context, repoID, prID string) (*PullRequest, error)
	List(ctx context.Context, repoID string, filters PRFilters) ([]*PullRequest, error)
	Update(ctx context.Context, repoID, prID string, fn func(*PullRequest) error) error
	Delete(ctx context.Context, repoID, prID string) error
	Save() error
}

// PRFilters for listing PRs
type PRFilters struct {
	Status string
}

// PullRequest represents a pull request
type PullRequest struct {
	ID           string
	Title        string
	Body         string
	Status       string
	Author       string
	SourceBranch string
	TargetBranch string
	Labels       []string
	Assignee     string
	Reviewers    []string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// LabelStore defines the interface for label storage
type LabelStore interface {
	List(ctx context.Context, repoID string) ([]Label, error)
	Add(ctx context.Context, repoID, name, color string) error
	Remove(ctx context.Context, repoID, name string) error
	Save() error
}

// Label represents a repository label
type Label struct {
	Name  string
	Color string
}

// TaskStore defines the interface for task storage
type TaskStore interface {
	Create(ctx context.Context, repoID, id, createdBy, title, description string) (*Task, error)
	List(ctx context.Context, repoID string, status string) ([]*Task, error)
	Claim(ctx context.Context, repoID, taskID, claimedBy string) error
	Complete(ctx context.Context, repoID, taskID, result string) error
	Save() error
}

// Task represents an agent task
type Task struct {
	ID          string
	RepoID      string
	Title       string
	Description string
	Status      string
	ClaimedBy   string
	CreatedBy   string
	CreatedAt   time.Time
	ClaimedAt   *time.Time
	CompletedAt *time.Time
	Result      string
}

// ReleaseStore defines the interface for release storage
type ReleaseStore interface {
	Create(ctx context.Context, repoID, tag, title, body, author string) (*Release, error)
	Get(ctx context.Context, repoID, releaseID string) (*Release, error)
	List(ctx context.Context, repoID string) ([]*Release, error)
	Delete(ctx context.Context, repoID, releaseID string) error
	Save() error
}

// Release represents a release
type Release struct {
	ID        string
	RepoID    string
	Tag       string
	Title     string
	Body      string
	Author    string
	CreatedAt time.Time
}

// ProtectionStore defines the interface for branch protection storage
type ProtectionStore interface {
	Get(ctx context.Context, repoID, branch string) (*BranchProtection, error)
	List(ctx context.Context, repoID string) ([]BranchProtection, error)
	Set(ctx context.Context, repoID string, protection BranchProtection) error
	Remove(ctx context.Context, repoID, branch string) error
	Save() error
}

// BranchProtection defines rules for a protected branch
type BranchProtection struct {
	Branch          string
	RequirePR       bool
	RequireApproval bool
	NoForcePush     bool
}

// UserStore defines the interface for user storage
type UserStore interface {
	Create(ctx context.Context, user *User) error
	Get(ctx context.Context, id string) (*User, error)
	GetByUsername(ctx context.Context, username string) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	Update(ctx context.Context, user *User) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context) ([]*User, error)
	AddSSHKey(ctx context.Context, key *SSHKey) error
	DeleteSSHKey(ctx context.Context, userID, keyID string) error
	ListSSHKeys(ctx context.Context, userID string) ([]SSHKey, error)
	FindByFingerprint(ctx context.Context, fingerprint string) (*User, *SSHKey, error)
}

// User represents a user
type User struct {
	ID           string
	Username     string
	Email        string
	PasswordHash string
	DisplayName  string
	AvatarURL    string
	Role         string
	SSHKeys      []SSHKey
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// SSHKey represents a user's SSH public key
type SSHKey struct {
	ID          string
	UserID      string
	Name        string
	Fingerprint string
	PublicKey   string
	CreatedAt   time.Time
}

// SessionStore defines the interface for session storage
type SessionStore interface {
	Create(ctx context.Context, session *Session) error
	Get(ctx context.Context, token string) (*Session, error)
	Delete(ctx context.Context, token string) error
	DeleteExpired(ctx context.Context) error
}

// Session represents a user session
type Session struct {
	ID        string
	UserID    string
	Token     string
	ExpiresAt time.Time
	CreatedAt time.Time
}

const (
	// RepoRoleOwner identifies the session user who created or owns a repository.
	RepoRoleOwner = "owner"
	// RepoRoleCollaborator identifies a session user allowed to write to a repository.
	RepoRoleCollaborator = "collaborator"
)

// RepoCollaborator represents a user's repo ownership or collaborator membership.
type RepoCollaborator struct {
	RepoID    string
	UserID    string
	Role      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// CanWrite reports whether this membership grants repository write access.
func (c *RepoCollaborator) CanWrite() bool {
	if c == nil {
		return false
	}
	return c.Role == RepoRoleOwner || c.Role == RepoRoleCollaborator
}

// RepoCollaboratorStore defines the interface for repository ownership and collaborator storage.
type RepoCollaboratorStore interface {
	Add(ctx context.Context, collaborator *RepoCollaborator) error
	Get(ctx context.Context, repoID, userID string) (*RepoCollaborator, error)
	ListByRepo(ctx context.Context, repoID string) ([]*RepoCollaborator, error)
	Remove(ctx context.Context, repoID, userID string) error
	IsWriter(ctx context.Context, repoID, userID string) (bool, error)
}

// ReviewCommentStore defines the interface for PR review comments
type ReviewCommentStore interface {
	Create(ctx context.Context, comment *ReviewComment) error
	Get(ctx context.Context, id string) (*ReviewComment, error)
	ListByPR(ctx context.Context, prID string) ([]*ReviewComment, error)
	Update(ctx context.Context, comment *ReviewComment) error
	Resolve(ctx context.Context, id string) error
	Delete(ctx context.Context, id string) error
}

// ReviewComment represents an inline code review comment
type ReviewComment struct {
	ID         string
	PRID       string
	FilePath   string
	LineNumber int
	AuthorID   string
	Body       string
	ParentID   string
	Status     string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}
