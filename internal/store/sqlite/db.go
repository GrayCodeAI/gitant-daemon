package sqlite

import (
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

// DB wraps the database connection
type DB struct {
	*sql.DB
	path string
}

// New creates a new SQLite database connection
func New(path string) (*DB, error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("creating database directory: %w", err)
	}

	db, err := sql.Open("sqlite", path+"?_journal_mode=WAL&_busy_timeout=5000&_foreign_keys=ON")
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}

	db.SetMaxOpenConns(1) // SQLite doesn't support concurrent writes well
	db.SetMaxIdleConns(1)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("pinging database: %w", err)
	}

	return &DB{DB: db, path: path}, nil
}

// RunMigrations runs all database migrations
func (db *DB) RunMigrations() error {
	slog.Info("running database migrations")

	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS migrations (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL UNIQUE,
		applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`); err != nil {
		return fmt.Errorf("creating migrations table: %w", err)
	}

	migrations := []struct {
		name string
		sql  string
	}{
		{
			name: "001_create_users",
			sql: `CREATE TABLE IF NOT EXISTS users (
				id TEXT PRIMARY KEY,
				username TEXT UNIQUE NOT NULL,
				email TEXT UNIQUE NOT NULL,
				password_hash TEXT NOT NULL,
				display_name TEXT DEFAULT '',
				avatar_url TEXT DEFAULT '',
				role TEXT DEFAULT 'developer',
				created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
				updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
			)`,
		},
		{
			name: "002_create_sessions",
			sql: `CREATE TABLE IF NOT EXISTS sessions (
				id TEXT PRIMARY KEY,
				user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
				token TEXT UNIQUE NOT NULL,
				expires_at TIMESTAMP NOT NULL,
				created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
			)`,
		},
		{
			name: "003_create_issues",
			sql: `CREATE TABLE IF NOT EXISTS issues (
				id TEXT PRIMARY KEY,
				repo_id TEXT NOT NULL,
				title TEXT NOT NULL,
				body TEXT DEFAULT '',
				status TEXT DEFAULT 'open',
				author TEXT NOT NULL,
				assignee TEXT DEFAULT '',
				created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
				updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
			)`,
		},
		{
			name: "004_create_issue_labels",
			sql: `CREATE TABLE IF NOT EXISTS issue_labels (
				issue_id TEXT NOT NULL REFERENCES issues(id) ON DELETE CASCADE,
				label TEXT NOT NULL,
				PRIMARY KEY (issue_id, label)
			)`,
		},
		{
			name: "005_create_pull_requests",
			sql: `CREATE TABLE IF NOT EXISTS pull_requests (
				id TEXT PRIMARY KEY,
				repo_id TEXT NOT NULL,
				title TEXT NOT NULL,
				body TEXT DEFAULT '',
				status TEXT DEFAULT 'open',
				author TEXT NOT NULL,
				source_branch TEXT NOT NULL,
				target_branch TEXT NOT NULL,
				assignee TEXT DEFAULT '',
				created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
				updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
			)`,
		},
		{
			name: "006_create_pr_labels",
			sql: `CREATE TABLE IF NOT EXISTS pr_labels (
				pr_id TEXT NOT NULL REFERENCES pull_requests(id) ON DELETE CASCADE,
				label TEXT NOT NULL,
				PRIMARY KEY (pr_id, label)
			)`,
		},
		{
			name: "007_create_pr_reviewers",
			sql: `CREATE TABLE IF NOT EXISTS pr_reviewers (
				pr_id TEXT NOT NULL REFERENCES pull_requests(id) ON DELETE CASCADE,
				reviewer TEXT NOT NULL,
				PRIMARY KEY (pr_id, reviewer)
			)`,
		},
		{
			name: "008_create_labels",
			sql: `CREATE TABLE IF NOT EXISTS labels (
				repo_id TEXT NOT NULL,
				name TEXT NOT NULL,
				color TEXT DEFAULT '#6b7280',
				PRIMARY KEY (repo_id, name)
			)`,
		},
		{
			name: "009_create_tasks",
			sql: `CREATE TABLE IF NOT EXISTS tasks (
				id TEXT PRIMARY KEY,
				repo_id TEXT NOT NULL,
				title TEXT NOT NULL,
				description TEXT DEFAULT '',
				status TEXT DEFAULT 'open',
				claimed_by TEXT DEFAULT '',
				created_by TEXT NOT NULL,
				created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
				claimed_at TIMESTAMP,
				completed_at TIMESTAMP,
				result TEXT DEFAULT ''
			)`,
		},
		{
			name: "010_create_releases",
			sql: `CREATE TABLE IF NOT EXISTS releases (
				id TEXT PRIMARY KEY,
				repo_id TEXT NOT NULL,
				tag TEXT NOT NULL,
				title TEXT NOT NULL,
				body TEXT DEFAULT '',
				author TEXT NOT NULL,
				created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
				UNIQUE(repo_id, tag)
			)`,
		},
		{
			name: "011_create_branch_protections",
			sql: `CREATE TABLE IF NOT EXISTS branch_protections (
				repo_id TEXT NOT NULL,
				branch TEXT NOT NULL,
				require_pr BOOLEAN DEFAULT FALSE,
				require_approval BOOLEAN DEFAULT FALSE,
				no_force_push BOOLEAN DEFAULT FALSE,
				PRIMARY KEY (repo_id, branch)
			)`,
		},
		{
			name: "012_create_review_comments",
			sql: `CREATE TABLE IF NOT EXISTS review_comments (
				id TEXT PRIMARY KEY,
				pr_id TEXT NOT NULL REFERENCES pull_requests(id) ON DELETE CASCADE,
				file_path TEXT NOT NULL,
				line_number INTEGER DEFAULT 0,
				author_id TEXT NOT NULL,
				body TEXT NOT NULL,
				parent_id TEXT REFERENCES review_comments(id),
				status TEXT DEFAULT 'open',
				created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
				updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
			)`,
		},
		{
			name: "013_create_issue_operations",
			sql: `CREATE TABLE IF NOT EXISTS issue_operations (
				id TEXT PRIMARY KEY,
				issue_id TEXT NOT NULL REFERENCES issues(id) ON DELETE CASCADE,
				op_type TEXT NOT NULL,
				author TEXT NOT NULL,
				timestamp TIMESTAMP NOT NULL,
				lamport INTEGER DEFAULT 0,
				data TEXT DEFAULT '{}',
				parent TEXT DEFAULT ''
			)`,
		},
		{
			name: "014_create_pr_operations",
			sql: `CREATE TABLE IF NOT EXISTS pr_operations (
				id TEXT PRIMARY KEY,
				pr_id TEXT NOT NULL REFERENCES pull_requests(id) ON DELETE CASCADE,
				op_type TEXT NOT NULL,
				author TEXT NOT NULL,
				timestamp TIMESTAMP NOT NULL,
				lamport INTEGER DEFAULT 0,
				data TEXT DEFAULT '{}',
				parent TEXT DEFAULT ''
			)`,
		},
		{
			name: "015_create_indexes",
			sql: `CREATE INDEX IF NOT EXISTS idx_issues_repo ON issues(repo_id);
				CREATE INDEX IF NOT EXISTS idx_issues_status ON issues(status);
				CREATE INDEX IF NOT EXISTS idx_prs_repo ON pull_requests(repo_id);
				CREATE INDEX IF NOT EXISTS idx_prs_status ON pull_requests(status);
				CREATE INDEX IF NOT EXISTS idx_tasks_repo ON tasks(repo_id);
				CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status);
				CREATE INDEX IF NOT EXISTS idx_releases_repo ON releases(repo_id);
				CREATE INDEX IF NOT EXISTS idx_sessions_token ON sessions(token);
				CREATE INDEX IF NOT EXISTS idx_sessions_expires ON sessions(expires_at);
				CREATE INDEX IF NOT EXISTS idx_review_comments_pr ON review_comments(pr_id);
				CREATE INDEX IF NOT EXISTS idx_issue_operations_issue ON issue_operations(issue_id);
				CREATE INDEX IF NOT EXISTS idx_pr_operations_pr ON pr_operations(pr_id)`,
		},
	}

	for _, m := range migrations {
		var count int
		err := db.QueryRow("SELECT COUNT(*) FROM migrations WHERE name = ?", m.name).Scan(&count)
		if err != nil {
			return fmt.Errorf("checking migration %s: %w", m.name, err)
		}

		if count > 0 {
			continue
		}

		slog.Info("applying migration", "name", m.name)
		if _, err := db.Exec(m.sql); err != nil {
			return fmt.Errorf("applying migration %s: %w", m.name, err)
		}

		if _, err := db.Exec("INSERT INTO migrations (name) VALUES (?)", m.name); err != nil {
			return fmt.Errorf("recording migration %s: %w", m.name, err)
		}
	}

	slog.Info("migrations complete")
	return nil
}

// Close closes the database connection
func (db *DB) Close() error {
	return db.DB.Close()
}
