-- Repo ownership and collaborator ACLs for session-authenticated repo access.
CREATE TABLE repo_collaborators (
    repo_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    role TEXT NOT NULL DEFAULT 'collaborator',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (repo_id, user_id)
);

CREATE INDEX idx_repo_collaborators_repo_id ON repo_collaborators(repo_id);
CREATE INDEX idx_repo_collaborators_user_id ON repo_collaborators(user_id);
CREATE INDEX idx_repo_collaborators_role ON repo_collaborators(role);
