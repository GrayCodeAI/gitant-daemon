package store

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// AuthService handles authentication operations
type AuthService struct {
	Users    UserStore
	sessions SessionStore
}

// NewAuthService creates a new auth service
func NewAuthService(users UserStore, sessions SessionStore) *AuthService {
	return &AuthService{
		Users:    users,
		sessions: sessions,
	}
}

// RegisterInput contains registration parameters
type RegisterInput struct {
	Username    string
	Email       string
	Password    string
	DisplayName string
}

// Register registers a new user
func (s *AuthService) Register(ctx context.Context, input RegisterInput) (*User, error) {
	if input.Username == "" {
		return nil, fmt.Errorf("username is required")
	}
	if input.Email == "" {
		return nil, fmt.Errorf("email is required")
	}
	if len(input.Password) < 8 {
		return nil, fmt.Errorf("password must be at least 8 characters")
	}

	// Check if username exists
	if _, err := s.Users.GetByUsername(ctx, input.Username); err == nil {
		return nil, fmt.Errorf("username already exists: %s", input.Username)
	}

	// Check if email exists
	if _, err := s.Users.GetByEmail(ctx, input.Email); err == nil {
		return nil, fmt.Errorf("email already exists: %s", input.Email)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hashing password: %w", err)
	}

	now := time.Now()
	user := &User{
		ID:           generateID(),
		Username:     input.Username,
		Email:        input.Email,
		PasswordHash: string(hash),
		DisplayName:  input.DisplayName,
		Role:         "developer",
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if user.DisplayName == "" {
		user.DisplayName = input.Username
	}

	if err := s.Users.Create(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

// LoginInput contains login parameters
type LoginInput struct {
	Username string
	Password string
}

// Login authenticates a user and creates a session
func (s *AuthService) Login(ctx context.Context, input LoginInput) (*Session, error) {
	user, err := s.Users.GetByUsername(ctx, input.Username)
	if err != nil {
		return nil, fmt.Errorf("invalid credentials")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(input.Password)); err != nil {
		return nil, fmt.Errorf("invalid credentials")
	}

	session := &Session{
		ID:        generateID(),
		UserID:    user.ID,
		Token:     generateToken(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
		CreatedAt: time.Now(),
	}

	if err := s.sessions.Create(ctx, session); err != nil {
		return nil, err
	}

	return session, nil
}

// ValidateSession validates a session token and returns the user
func (s *AuthService) ValidateSession(ctx context.Context, token string) (*User, error) {
	session, err := s.sessions.Get(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("invalid session")
	}

	if session.ExpiresAt.Before(time.Now()) {
		s.sessions.Delete(ctx, token)
		return nil, fmt.Errorf("session expired")
	}

	user, err := s.Users.Get(ctx, session.UserID)
	if err != nil {
		return nil, fmt.Errorf("user not found")
	}

	return user, nil
}

// Logout invalidates a session
func (s *AuthService) Logout(ctx context.Context, token string) error {
	return s.sessions.Delete(ctx, token)
}

// CleanupSessions removes expired sessions
func (s *AuthService) CleanupSessions(ctx context.Context) error {
	return s.sessions.DeleteExpired(ctx)
}

func generateID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic("crypto/rand.Read failed: " + err.Error())
	}
	return hex.EncodeToString(b)
}

func generateToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic("crypto/rand.Read failed: " + err.Error())
	}
	return hex.EncodeToString(b)
}
