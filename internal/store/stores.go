package store

// Stores holds all store instances
type Stores struct {
	Issues         IssueStore
	PRs            PullRequestStore
	Labels         LabelStore
	Tasks          TaskStore
	Releases       ReleaseStore
	Protections    ProtectionStore
	Users          UserStore
	Sessions       SessionStore
	ReviewComments ReviewCommentStore
	Auth           *AuthService
}
