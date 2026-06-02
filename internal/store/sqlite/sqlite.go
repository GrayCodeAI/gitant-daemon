package sqlite

import (
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite"
	"github.com/golang-migrate/migrate/v4/source/file"
)

// Store represents a SQLite-backed store
type Store struct {
	db      *sql.DB
	dataDir string
}

// NewStore creates a new SQLite store
func NewStore(dataDir string) (*Store, error) {
	dbPath := filepath.Join(dataDir, "gitant.db")

	// Ensure directory exists
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("creating data directory: %w", err)
	}

	// Open SQLite database with proper settings for concurrent access
	// Using DSN parameters that enable WAL mode and proper locking
	dsn := fmt.Sprintf("%s?_journal_mode=WAL&_timeout=5000", dbPath)

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("opening SQLite database: %w", err)
	}

	// Set connection pooling
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(0)

	// Test connection
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("pinging SQLite database: %w", err)
	}

	s := &Store{
		db:      db,
		dataDir: dataDir,
	}

	// Run migrations
	if err := s.runMigrations(); err != nil {
		db.Close()
		return nil, fmt.Errorf("running migrations: %w", err)
	}

	return s, nil
}

// runMigrations runs database migrations
func (s *Store) runMigrations() error {
	driver, err := sqlite.WithInstance(s.db, &sqlite.Config{})
	if err != nil {
		return fmt.Errorf("creating SQLite migration driver: %w", err)
	}

	// Use file:// source for migrations
	// The path is relative to the module root
	migrationsPath := "file://internal/store/sqlite/migrations"

	fileDriver := &file.File{}
	src, err := fileDriver.Open(migrationsPath)
	if err != nil {
		return fmt.Errorf("creating migration source: %w", err)
	}

	m, err := migrate.NewWithInstance("file", src, "sqlite", driver)
	if err != nil {
		return fmt.Errorf("creating migrate instance: %w", err)
	}

	// Run migrations
	err = m.Up()
	if err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("applying migrations: %w", err)
	}

	if err == migrate.ErrNoChange {
		slog.Info("Database schema is up to date")
	} else {
		slog.Info("Database migrations applied successfully")
	}

	return nil
}

// Close closes the database connection
func (s *Store) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

// DB returns the underlying sql.DB connection
func (s *Store) DB() *sql.DB {
	return s.db
}

// NewUserStore creates a SQLite-backed user store
func (s *Store) NewUserStore() *SQLUserStore {
	return &SQLUserStore{db: s.db}
}

// NewSessionStore creates a SQLite-backed session store
func (s *Store) NewSessionStore() *SQLSessionStore {
	return &SQLSessionStore{db: s.db}
}

// NewIssueStore creates a SQLite-backed issue store
func (s *Store) NewIssueStore() *SQLIssueStore {
	return &SQLIssueStore{db: s.db}
}

// NewPullRequestStore creates a SQLite-backed pull request store
func (s *Store) NewPullRequestStore() *SQLPullRequestStore {
	return &SQLPullRequestStore{db: s.db}
}

// NewLabelStore creates a SQLite-backed label store
func (s *Store) NewLabelStore() *SQLLabelStore {
	return &SQLLabelStore{db: s.db}
}

// NewTaskStore creates a SQLite-backed task store
func (s *Store) NewTaskStore() *SQLTaskStore {
	return &SQLTaskStore{db: s.db}
}

// NewReleaseStore creates a SQLite-backed release store
func (s *Store) NewReleaseStore() *SQLReleaseStore {
	return &SQLReleaseStore{db: s.db}
}

// NewProtectionStore creates a SQLite-backed protection store
func (s *Store) NewProtectionStore() *SQLProtectionStore {
	return &SQLProtectionStore{db: s.db}
}

// NewReviewCommentStore creates a SQLite-backed review comment store
func (s *Store) NewReviewCommentStore() *SQLReviewCommentStore {
	return &SQLReviewCommentStore{db: s.db}
}
