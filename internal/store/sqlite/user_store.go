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

// AddSSHKey adds an SSH public key for a user
func (s *SQLUserStore) AddSSHKey(ctx context.Context, key *store.SSHKey) error {
	query := `
		INSERT INTO ssh_keys (id, user_id, name, public_key, fingerprint, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	_, err := s.db.ExecContext(ctx, query,
		key.ID,
		key.UserID,
		key.Name,
		key.PublicKey,
		key.Fingerprint,
		key.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("adding SSH key: %w", err)
	}

	return nil
}

// DeleteSSHKey removes an SSH key
func (s *SQLUserStore) DeleteSSHKey(ctx context.Context, userID, keyID string) error {
	query := `DELETE FROM ssh_keys WHERE id = ? AND user_id = ?`

	result, err := s.db.ExecContext(ctx, query, keyID, userID)
	if err != nil {
		return fmt.Errorf("deleting SSH key: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("SSH key not found: %s", keyID)
	}

	return nil
}

// ListSSHKeys lists all SSH keys for a user
func (s *SQLUserStore) ListSSHKeys(ctx context.Context, userID string) ([]store.SSHKey, error) {
	query := `
		SELECT id, user_id, name, public_key, fingerprint, created_at
		FROM ssh_keys
		WHERE user_id = ?
		ORDER BY created_at DESC
	`

	rows, err := s.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("listing SSH keys: %w", err)
	}
	defer rows.Close()

	var keys []store.SSHKey
	for rows.Next() {
		key := store.SSHKey{}
		err := rows.Scan(
			&key.ID,
			&key.UserID,
			&key.Name,
			&key.PublicKey,
			&key.Fingerprint,
			&key.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scanning SSH key: %w", err)
		}
		keys = append(keys, key)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating SSH keys: %w", err)
	}

	return keys, nil
}

// FindByFingerprint finds a user by their SSH key fingerprint
func (s *SQLUserStore) FindByFingerprint(ctx context.Context, fingerprint string) (*store.User, *store.SSHKey, error) {
	query := `
		SELECT u.id, u.username, u.email, u.password_hash, u.display_name, u.avatar_url, u.role, u.created_at, u.updated_at,
		       k.id, k.user_id, k.name, k.public_key, k.fingerprint, k.created_at
		FROM users u
		INNER JOIN ssh_keys k ON u.id = k.user_id
		WHERE k.fingerprint = ?
	`

	user := &store.User{}
	key := &store.SSHKey{}
	err := s.db.QueryRowContext(ctx, query, fingerprint).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.PasswordHash,
		&user.DisplayName,
		&user.AvatarURL,
		&user.Role,
		&user.CreatedAt,
		&user.UpdatedAt,
		&key.ID,
		&key.UserID,
		&key.Name,
		&key.PublicKey,
		&key.Fingerprint,
		&key.CreatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil, fmt.Errorf("no user found for fingerprint: %s", fingerprint)
		}
		return nil, nil, fmt.Errorf("finding user by fingerprint: %w", err)
	}

	return user, key, nil
}
