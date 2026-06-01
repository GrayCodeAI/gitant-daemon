package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/lakshmanpatel/gitant/internal/store"
)

// SQLUserStore implements UserStore using SQLite
type SQLUserStore struct {
	db *sql.DB
}

// Create creates a new user
func (s *SQLUserStore) Create(ctx context.Context, user *store.User) error {
	query := `
		INSERT INTO users (id, username, email, password_hash, display_name, avatar_url, role, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := s.db.ExecContext(ctx, query,
		user.ID,
		user.Username,
		user.Email,
		user.PasswordHash,
		user.DisplayName,
		user.AvatarURL,
		user.Role,
		user.CreatedAt,
		user.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("creating user: %w", err)
	}

	return nil
}

// Get retrieves a user by ID
func (s *SQLUserStore) Get(ctx context.Context, id string) (*store.User, error) {
	query := `
		SELECT id, username, email, password_hash, display_name, avatar_url, role, created_at, updated_at
		FROM users
		WHERE id = ?
	`

	user := &store.User{}
	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.PasswordHash,
		&user.DisplayName,
		&user.AvatarURL,
		&user.Role,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found: %s", id)
		}
		return nil, fmt.Errorf("getting user: %w", err)
	}

	return user, nil
}

// GetByUsername retrieves a user by username
func (s *SQLUserStore) GetByUsername(ctx context.Context, username string) (*store.User, error) {
	query := `
		SELECT id, username, email, password_hash, display_name, avatar_url, role, created_at, updated_at
		FROM users
		WHERE username = ?
	`

	user := &store.User{}
	err := s.db.QueryRowContext(ctx, query, username).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.PasswordHash,
		&user.DisplayName,
		&user.AvatarURL,
		&user.Role,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found: %s", username)
		}
		return nil, fmt.Errorf("getting user by username: %w", err)
	}

	return user, nil
}

// GetByEmail retrieves a user by email
func (s *SQLUserStore) GetByEmail(ctx context.Context, email string) (*store.User, error) {
	query := `
		SELECT id, username, email, password_hash, display_name, avatar_url, role, created_at, updated_at
		FROM users
		WHERE email = ?
	`

	user := &store.User{}
	err := s.db.QueryRowContext(ctx, query, email).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.PasswordHash,
		&user.DisplayName,
		&user.AvatarURL,
		&user.Role,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found: %s", email)
		}
		return nil, fmt.Errorf("getting user by email: %w", err)
	}

	return user, nil
}

// Update updates a user
func (s *SQLUserStore) Update(ctx context.Context, user *store.User) error {
	query := `
		UPDATE users
		SET username = ?, email = ?, password_hash = ?, display_name = ?, avatar_url = ?, role = ?, updated_at = ?
		WHERE id = ?
	`

	result, err := s.db.ExecContext(ctx, query,
		user.Username,
		user.Email,
		user.PasswordHash,
		user.DisplayName,
		user.AvatarURL,
		user.Role,
		time.Now(),
		user.ID,
	)

	if err != nil {
		return fmt.Errorf("updating user: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("user not found: %s", user.ID)
	}

	return nil
}

// Delete removes a user
func (s *SQLUserStore) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM users WHERE id = ?`

	result, err := s.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("deleting user: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("user not found: %s", id)
	}

	return nil
}

// List retrieves all users
func (s *SQLUserStore) List(ctx context.Context) ([]*store.User, error) {
	query := `
		SELECT id, username, email, password_hash, display_name, avatar_url, role, created_at, updated_at
		FROM users
		ORDER BY created_at DESC
	`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("listing users: %w", err)
	}
	defer rows.Close()

	var users []*store.User
	for rows.Next() {
		user := &store.User{}
		err := rows.Scan(
			&user.ID,
			&user.Username,
			&user.Email,
			&user.PasswordHash,
			&user.DisplayName,
			&user.AvatarURL,
			&user.Role,
			&user.CreatedAt,
			&user.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scanning user: %w", err)
		}
		users = append(users, user)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating users: %w", err)
	}

	return users, nil
}
