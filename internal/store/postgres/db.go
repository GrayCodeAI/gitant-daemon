package postgres

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DB wraps the PostgreSQL connection pool
type DB struct {
	pool *pgxpool.Pool
}

// New creates a new PostgreSQL database connection
func New(connString string) (*DB, error) {
	config, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, fmt.Errorf("parsing connection string: %w", err)
	}

	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		return nil, fmt.Errorf("creating connection pool: %w", err)
	}

	if err := pool.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("pinging database: %w", err)
	}

	return &DB{pool: pool}, nil
}

// RunMigrations runs all database migrations
func (db *DB) RunMigrations() error {
	slog.Info("running database migrations")

	_, err := db.pool.Exec(context.Background(), `CREATE TABLE IF NOT EXISTS migrations (
		id SERIAL PRIMARY KEY,
		name TEXT NOT NULL UNIQUE,
		applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`)
	if err != nil {
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
				username VARCHAR(39) UNIQUE NOT NULL,
				email VARCHAR(255) UNIQUE NOT NULL,
				password_hash TEXT NOT NULL,
				display_name VARCHAR(100) DEFAULT '',
				avatar_url TEXT DEFAULT '',
				role VARCHAR(20) DEFAULT 'developer',
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
				title VARCHAR(255) NOT NULL,
				body TEXT DEFAULT '',
				status VARCHAR(20) DEFAULT 'open',
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
				label VARCHAR(64) NOT NULL,
				PRIMARY KEY (issue_id, label)
			)`,
		},
		{
			name: "005_create_pull_requests",
			sql: `CREATE TABLE IF NOT EXISTS pull_requests (
				id TEXT PRIMARY KEY,
				repo_id TEXT NOT NULL,
				title VARCHAR(255) NOT NULL,
				body TEXT DEFAULT '',
				status VARCHAR(20) DEFAULT 'open',
				author TEXT NOT NULL,
				source_branch VARCHAR(100) NOT NULL,
				target_branch VARCHAR(100) NOT NULL,
				assignee TEXT DEFAULT '',
				created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
				updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
			)`,
		},
		{
			name: "006_create_pr_labels",
			sql: `CREATE TABLE IF NOT EXISTS pr_labels (
				pr_id TEXT NOT NULL REFERENCES pull_requests(id) ON DELETE CASCADE,
				label VARCHAR(64) NOT NULL,
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
				name VARCHAR(64) NOT NULL,
				color VARCHAR(7) DEFAULT '#6b7280',
				PRIMARY KEY (repo_id, name)
			)`,
		},
		{
			name: "009_create_tasks",
			sql: `CREATE TABLE IF NOT EXISTS tasks (
				id TEXT PRIMARY KEY,
				repo_id TEXT NOT NULL,
				title VARCHAR(255) NOT NULL,
				description TEXT DEFAULT '',
				status VARCHAR(20) DEFAULT 'open',
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
				tag VARCHAR(100) NOT NULL,
				title VARCHAR(255) NOT NULL,
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
				branch VARCHAR(100) NOT NULL,
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
				status VARCHAR(20) DEFAULT 'open',
				created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
				updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
			)`,
		},
		{
			name: "013_create_discussions",
			sql: `CREATE TABLE IF NOT EXISTS discussions (
				id TEXT PRIMARY KEY,
				repo_id TEXT NOT NULL,
				title VARCHAR(255) NOT NULL,
				body TEXT DEFAULT '',
				author TEXT NOT NULL,
				category VARCHAR(20) DEFAULT 'general',
				status VARCHAR(20) DEFAULT 'open',
				upvotes INTEGER DEFAULT 0,
				views INTEGER DEFAULT 0,
				created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
				updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
			)`,
		},
		{
			name: "014_create_discussion_answers",
			sql: `CREATE TABLE IF NOT EXISTS discussion_answers (
				id TEXT PRIMARY KEY,
				discussion_id TEXT NOT NULL REFERENCES discussions(id) ON DELETE CASCADE,
				body TEXT NOT NULL,
				author TEXT NOT NULL,
				is_accepted BOOLEAN DEFAULT FALSE,
				upvotes INTEGER DEFAULT 0,
				created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
				updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
			)`,
		},
		{
			name: "015_create_projects",
			sql: `CREATE TABLE IF NOT EXISTS projects (
				id TEXT PRIMARY KEY,
				repo_id TEXT NOT NULL,
				name VARCHAR(100) NOT NULL,
				description TEXT DEFAULT '',
				status VARCHAR(20) DEFAULT 'active',
				created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
				updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
			)`,
		},
		{
			name: "016_create_project_columns",
			sql: `CREATE TABLE IF NOT EXISTS project_columns (
				id TEXT PRIMARY KEY,
				project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
				name VARCHAR(100) NOT NULL,
				order_index INTEGER DEFAULT 0
			)`,
		},
		{
			name: "017_create_project_cards",
			sql: `CREATE TABLE IF NOT EXISTS project_cards (
				id TEXT PRIMARY KEY,
				column_id TEXT NOT NULL REFERENCES project_columns(id) ON DELETE CASCADE,
				title VARCHAR(255) NOT NULL,
				description TEXT DEFAULT '',
				assignee TEXT DEFAULT '',
				issue_id TEXT,
				pr_id TEXT,
				order_index INTEGER DEFAULT 0,
				created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
				updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
			)`,
		},
		{
			name: "018_create_audit_events",
			sql: `CREATE TABLE IF NOT EXISTS audit_events (
				id TEXT PRIMARY KEY,
				type VARCHAR(50) NOT NULL,
				actor TEXT NOT NULL,
				repo_id TEXT,
				resource TEXT NOT NULL,
				action VARCHAR(100) NOT NULL,
				details JSONB,
				ip VARCHAR(45),
				user_agent TEXT,
				created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
			)`,
		},
		{
			name: "019_create_notifications",
			sql: `CREATE TABLE IF NOT EXISTS notifications (
				id TEXT PRIMARY KEY,
				user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
				type VARCHAR(50) NOT NULL,
				title VARCHAR(255) NOT NULL,
				body TEXT DEFAULT '',
				read BOOLEAN DEFAULT FALSE,
				metadata JSONB,
				created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
			)`,
		},
		{
			name: "020_create_indexes",
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
				CREATE INDEX IF NOT EXISTS idx_discussions_repo ON discussions(repo_id);
				CREATE INDEX IF NOT EXISTS idx_projects_repo ON projects(repo_id);
				CREATE INDEX IF NOT EXISTS idx_audit_events_actor ON audit_events(actor);
				CREATE INDEX IF NOT EXISTS idx_audit_events_repo ON audit_events(repo_id);
				CREATE INDEX IF NOT EXISTS idx_audit_events_type ON audit_events(type);
				CREATE INDEX IF NOT EXISTS idx_notifications_user ON notifications(user_id);
				CREATE INDEX IF NOT EXISTS idx_notifications_read ON notifications(read)`,
		},
	}

	for _, m := range migrations {
		var count int
		err := db.pool.QueryRow(context.Background(), "SELECT COUNT(*) FROM migrations WHERE name = ?", m.name).Scan(&count)
		if err != nil {
			// PostgreSQL uses $1 for parameters
			err = db.pool.QueryRow(context.Background(), "SELECT COUNT(*) FROM migrations WHERE name = $1", m.name).Scan(&count)
			if err != nil {
				return fmt.Errorf("checking migration %s: %w", m.name, err)
			}
		}

		if count > 0 {
			continue
		}

		slog.Info("applying migration", "name", m.name)
		if _, err := db.pool.Exec(context.Background(), m.sql); err != nil {
			return fmt.Errorf("applying migration %s: %w", m.name, err)
		}

		if _, err := db.pool.Exec(context.Background(), "INSERT INTO migrations (name) VALUES ($1)", m.name); err != nil {
			return fmt.Errorf("recording migration %s: %w", m.name, err)
		}
	}

	slog.Info("migrations complete")
	return nil
}

// Pool returns the connection pool
func (db *DB) Pool() *pgxpool.Pool {
	return db.pool
}

// Close closes the database connection
func (db *DB) Close() {
	db.pool.Close()
}

// Exec executes a query
func (db *DB) Exec(ctx context.Context, sql string, args ...interface{}) error {
	_, err := db.pool.Exec(ctx, sql, args...)
	return err
}

// Query executes a query and returns rows
func (db *DB) Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
	return db.pool.Query(ctx, sql, args...)
}

// QueryRow executes a query and returns a single row
func (db *DB) QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row {
	return db.pool.QueryRow(ctx, sql, args...)
}
