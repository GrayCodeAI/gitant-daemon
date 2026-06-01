-- Initial SQLite schema for Gitant daemon

-- Users table
CREATE TABLE users (
    id TEXT PRIMARY KEY,
    username TEXT UNIQUE NOT NULL,
    email TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    display_name TEXT,
    avatar_url TEXT,
    role TEXT NOT NULL DEFAULT 'developer',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Sessions table (in-memory is fine, but we'll support persistence too)
CREATE TABLE sessions (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    token TEXT UNIQUE NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX idx_sessions_token ON sessions(token);
CREATE INDEX idx_sessions_user_id ON sessions(user_id);
CREATE INDEX idx_sessions_expires_at ON sessions(expires_at);

-- Issues table (CRDT with operation log stored as JSON)
CREATE TABLE issues (
    id TEXT NOT NULL,
    repo_id TEXT NOT NULL,
    title TEXT NOT NULL,
    body TEXT,
    status TEXT NOT NULL DEFAULT 'open',
    author TEXT NOT NULL,
    labels TEXT,  -- JSON array
    assignee TEXT,
    tombstoned INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id, repo_id)
);

CREATE INDEX idx_issues_repo_id ON issues(repo_id);
CREATE INDEX idx_issues_status ON issues(status);

-- Pull Requests table (CRDT with operation log stored as JSON)
CREATE TABLE pull_requests (
    id TEXT NOT NULL,
    repo_id TEXT NOT NULL,
    title TEXT NOT NULL,
    body TEXT,
    status TEXT NOT NULL DEFAULT 'open',
    author TEXT NOT NULL,
    source_branch TEXT NOT NULL,
    target_branch TEXT NOT NULL,
    labels TEXT,  -- JSON array
    assignee TEXT,
    reviewers TEXT,  -- JSON array
    tombstoned INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id, repo_id)
);

CREATE INDEX idx_pull_requests_repo_id ON pull_requests(repo_id);
CREATE INDEX idx_pull_requests_status ON pull_requests(status);

-- Labels table
CREATE TABLE labels (
    repo_id TEXT NOT NULL,
    name TEXT NOT NULL,
    color TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (repo_id, name)
);

CREATE INDEX idx_labels_repo_id ON labels(repo_id);

-- Tasks table
CREATE TABLE tasks (
    id TEXT NOT NULL,
    repo_id TEXT NOT NULL,
    title TEXT NOT NULL,
    description TEXT,
    status TEXT NOT NULL DEFAULT 'open',
    claimed_by TEXT,
    created_by TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    claimed_at TIMESTAMP,
    completed_at TIMESTAMP,
    result TEXT,
    PRIMARY KEY (id, repo_id)
);

CREATE INDEX idx_tasks_repo_id ON tasks(repo_id);
CREATE INDEX idx_tasks_status ON tasks(status);
CREATE INDEX idx_tasks_claimed_by ON tasks(claimed_by);

-- Releases table
CREATE TABLE releases (
    id TEXT NOT NULL,
    repo_id TEXT NOT NULL,
    tag TEXT NOT NULL,
    title TEXT NOT NULL,
    body TEXT,
    author TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id, repo_id)
);

CREATE INDEX idx_releases_repo_id ON releases(repo_id);
CREATE INDEX idx_releases_tag ON releases(tag);

-- Branch protection rules table
CREATE TABLE branch_protections (
    repo_id TEXT NOT NULL,
    branch TEXT NOT NULL,
    require_pr INTEGER NOT NULL DEFAULT 0,
    require_approval INTEGER NOT NULL DEFAULT 0,
    no_force_push INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (repo_id, branch)
);

CREATE INDEX idx_branch_protections_repo_id ON branch_protections(repo_id);

-- Review comments table
CREATE TABLE review_comments (
    id TEXT PRIMARY KEY,
    pr_id TEXT NOT NULL,
    repo_id TEXT NOT NULL,
    file_path TEXT NOT NULL,
    line_number INTEGER NOT NULL,
    author_id TEXT NOT NULL,
    body TEXT NOT NULL,
    parent_id TEXT,
    status TEXT NOT NULL DEFAULT 'open',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_review_comments_pr_id ON review_comments(pr_id);
CREATE INDEX idx_review_comments_status ON review_comments(status);

-- OAuth providers table (for SSO)
CREATE TABLE oauth_providers (
    id TEXT PRIMARY KEY,
    provider TEXT NOT NULL,  -- 'github', 'gitlab', 'google', etc.
    user_id TEXT NOT NULL,
    provider_user_id TEXT NOT NULL,
    access_token TEXT,
    refresh_token TEXT,
    expires_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(provider, provider_user_id),
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX idx_oauth_user_id ON oauth_providers(user_id);

-- Repository metadata (for future use)
CREATE TABLE repos (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT,
    is_private INTEGER NOT NULL DEFAULT 0,
    author TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_repos_author ON repos(author);
