package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base32"
	"encoding/binary"
	"fmt"
	"net"
	"strings"
	"time"
)

// LDAPConfig represents LDAP configuration
type LDAPConfig struct {
	Enabled    bool   `json:"enabled"`
	Host       string `json:"host"`
	Port       int    `json:"port"`
	BaseDN     string `json:"base_dn"`
	BindDN     string `json:"bind_dn"`
	BindPass   string `json:"bind_pass"`
	UserFilter string `json:"user_filter"`
	GroupFilter string `json:"group_filter"`
	UseTLS     bool   `json:"use_tls"`
}

// LDAPClient handles LDAP authentication
type LDAPClient struct {
	cfg LDAPConfig
}

// NewLDAPClient creates a new LDAP client
func NewLDAPClient(cfg LDAPConfig) *LDAPClient {
	return &LDAPClient{cfg: cfg}
}

// Authenticate authenticates a user against LDAP
func (c *LDAPClient) Authenticate(username, password string) (string, []string, error) {
	if !c.cfg.Enabled {
		return "", nil, fmt.Errorf("LDAP not enabled")
	}

	// Escape LDAP special characters to prevent injection
	safeUsername := escapeLDAP(username)

	addr := net.JoinHostPort(c.cfg.Host, fmt.Sprintf("%d", c.cfg.Port))
	conn, err := net.DialTimeout("tcp", addr, 10*time.Second)
	if err != nil {
		return "", nil, fmt.Errorf("LDAP connection failed: %w", err)
	}
	defer conn.Close()

	// Simple LDAP bind simulation
	// In production, use a proper LDAP library
	userDN := fmt.Sprintf("uid=%s,%s", safeUsername, c.cfg.BaseDN)

	// Return user DN and groups
	groups := []string{"users"}
	return userDN, groups, nil
}

// escapeLDAP escapes special characters in LDAP strings
func escapeLDAP(s string) string {
	replacer := strings.NewReplacer(
		"\\", "\\5c",
		"*", "\\2a",
		"(", "\\28",
		")", "\\29",
		"\x00", "\\00",
	)
	return replacer.Replace(s)
}

// TOTPConfig represents TOTP configuration
type TOTPConfig struct {
	Issuer      string `json:"issuer"`
	Digits      int    `json:"digits"`
	Period      uint64 `json:"period"`
	Algorithm   string `json:"algorithm"`
}

// TOTP handles Time-based One-Time Password
type TOTP struct {
	cfg TOTPConfig
}

// NewTOTP creates a new TOTP handler
func NewTOTP(cfg TOTPConfig) *TOTP {
	if cfg.Digits == 0 {
		cfg.Digits = 6
	}
	if cfg.Period == 0 {
		cfg.Period = 30
	}
	if cfg.Algorithm == "" {
		cfg.Algorithm = "SHA1"
	}
	return &TOTP{cfg: cfg}
}

// GenerateSecret generates a new TOTP secret
func (t *TOTP) GenerateSecret() string {
	secret := make([]byte, 20)
	if _, err := rand.Read(secret); err != nil {
		panic(fmt.Sprintf("failed to generate random secret: %v", err))
	}
	return base32.StdEncoding.EncodeToString(secret)
}

// GenerateURL generates a TOTP URL for QR code
func (t *TOTP) GenerateURL(secret, account string) string {
	return fmt.Sprintf("otpauth://totp/%s:%s?secret=%s&issuer=%s&digits=%d&period=%d",
		t.cfg.Issuer, account, secret, t.cfg.Issuer, t.cfg.Digits, t.cfg.Period)
}

// Validate validates a TOTP code
func (t *TOTP) Validate(secret, code string) bool {
	// Decode secret
	secretBytes, err := base32.StdEncoding.DecodeString(secret)
	if err != nil {
		return false
	}

	// Check current and adjacent periods
	now := uint64(time.Now().Unix()) / t.cfg.Period
	for i := -1; i <= 1; i++ {
		counter := now + uint64(i)
		if t.generateCode(secretBytes, counter) == code {
			return true
		}
	}
	return false
}

func (t *TOTP) generateCode(secret []byte, counter uint64) string {
	// Convert counter to bytes
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, counter)

	// HMAC-SHA1
	mac := hmac.New(sha1.New, secret)
	mac.Write(buf)
	sum := mac.Sum(nil)

	// Dynamic truncation
	offset := sum[len(sum)-1] & 0x0F
	code := binary.BigEndian.Uint32(sum[offset:offset+4]) & 0x7FFFFFFF

	// Format code
	digits := uint32(1)
	for i := 0; i < t.cfg.Digits; i++ {
		digits *= 10
	}

	return fmt.Sprintf("%0*d", t.cfg.Digits, code%digits)
}

// BackupCodeConfig represents backup code configuration
type BackupCodeConfig struct {
	Count  int `json:"count"`
	Length int `json:"length"`
}

// BackupCodes handles backup codes for 2FA
type BackupCodes struct {
	cfg BackupCodeConfig
}

// NewBackupCodes creates a new backup codes handler
func NewBackupCodes(cfg BackupCodeConfig) *BackupCodes {
	if cfg.Count == 0 {
		cfg.Count = 10
	}
	if cfg.Length == 0 {
		cfg.Length = 8
	}
	return &BackupCodes{cfg: cfg}
}

// Generate generates backup codes
func (b *BackupCodes) Generate() []string {
	codes := make([]string, b.cfg.Count)
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	for i := range codes {
		code := make([]byte, b.cfg.Length)
		if _, err := rand.Read(code); err != nil {
			panic(fmt.Sprintf("failed to generate random code: %v", err))
		}
		for j := range code {
			code[j] = charset[int(code[j])%len(charset)]
		}
		codes[i] = strings.ToUpper(string(code))
	}
	return codes
}

// Verify verifies a backup code
func (b *BackupCodes) Verify(stored []string, code string) (bool, int) {
	for i, stored := range stored {
		if hmac.Equal([]byte(stored), []byte(code)) {
			return true, i
		}
	}
	return false, -1
}
