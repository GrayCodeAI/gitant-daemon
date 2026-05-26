package sqlite

import (
	"context"
	"fmt"
	"time"

	"github.com/lakshmanpatel/gitant/internal/store"
)

// UserStore implements store.UserStore for SQLite
type UserStore struct {
	db *DB
}

// NewUserStore creates a new SQLite user store
func NewUserStore(db *DB) *UserStore {
	return &UserStore{db: db}
}

// Create creates a new user
func (s *UserStore) Create(ctx context.Context, user *store.User) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO users (id, username, email, password_hash, display_name, avatar_url, role, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		user.ID, user.Username, user.Email, user.PasswordHash,
		user.DisplayName, user.AvatarURL, user.Role,
		user.CreatedAt, user.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("creating user: %w", err)
	}
	return nil
}

// Get gets a user by ID
func (s *UserStore) Get(ctx context.Context, id string) (*store.User, error) {
	user := &store.User{}
	err := s.db.QueryRowContext(ctx,
		`SELECT id, username, email, password_hash, display_name, avatar_url, role, created_at, updated_at
		 FROM users WHERE id = ?`, id,
	).Scan(&user.ID, &user.Username, &user.Email, &user.PasswordHash,
		&user.DisplayName, &user.AvatarURL, &user.Role,
		&user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("getting user: %w", err)
	}
	return user, nil
}

// GetByUsername gets a user by username
func (s *UserStore) GetByUsername(ctx context.Context, username string) (*store.User, error) {
	user := &store.User{}
	err := s.db.QueryRowContext(ctx,
		`SELECT id, username, email, password_hash, display_name, avatar_url, role, created_at, updated_at
		 FROM users WHERE username = ?`, username,
	).Scan(&user.ID, &user.Username, &user.Email, &user.PasswordHash,
		&user.DisplayName, &user.AvatarURL, &user.Role,
		&user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("getting user by username: %w", err)
	}
	return user, nil
}

// GetByEmail gets a user by email
func (s *UserStore) GetByEmail(ctx context.Context, email string) (*store.User, error) {
	user := &store.User{}
	err := s.db.QueryRowContext(ctx,
		`SELECT id, username, email, password_hash, display_name, avatar_url, role, created_at, updated_at
		 FROM users WHERE email = ?`, email,
	).Scan(&user.ID, &user.Username, &user.Email, &user.PasswordHash,
		&user.DisplayName, &user.AvatarURL, &user.Role,
		&user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("getting user by email: %w", err)
	}
	return user, nil
}

// Update updates a user
func (s *UserStore) Update(ctx context.Context, user *store.User) error {
	user.UpdatedAt = time.Now()
	_, err := s.db.ExecContext(ctx,
		`UPDATE users SET username = ?, email = ?, password_hash = ?, display_name = ?,
		 avatar_url = ?, role = ?, updated_at = ? WHERE id = ?`,
		user.Username, user.Email, user.PasswordHash,
		user.DisplayName, user.AvatarURL, user.Role,
		user.UpdatedAt, user.ID,
	)
	if err != nil {
		return fmt.Errorf("updating user: %w", err)
	}
	return nil
}

// Delete deletes a user
func (s *UserStore) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM users WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("deleting user: %w", err)
	}
	return nil
}

// List lists all users
func (s *UserStore) List(ctx context.Context) ([]*store.User, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, username, email, password_hash, display_name, avatar_url, role, created_at, updated_at
		 FROM users ORDER BY username`)
	if err != nil {
		return nil, fmt.Errorf("listing users: %w", err)
	}
	defer rows.Close()

	var users []*store.User
	for rows.Next() {
		user := &store.User{}
		if err := rows.Scan(&user.ID, &user.Username, &user.Email, &user.PasswordHash,
			&user.DisplayName, &user.AvatarURL, &user.Role,
			&user.CreatedAt, &user.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scanning user: %w", err)
		}
		users = append(users, user)
	}
	return users, nil
}
