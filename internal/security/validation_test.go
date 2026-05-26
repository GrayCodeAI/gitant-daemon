package security

import (
	"testing"
	"time"
)

func TestValidateUsername(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid", "alice", false},
		{"valid with hyphen", "alice-bob", false},
		{"valid with underscore", "alice_bob", false},
		{"too short", "ab", true},
		{"too long", "a123456789012345678901234567890123456789", true},
		{"invalid chars", "alice@bob", true},
		{"empty", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateUsername(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateUsername(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestValidateEmail(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid", "alice@example.com", false},
		{"valid with subdomain", "alice@sub.example.com", false},
		{"invalid no @", "aliceexample.com", true},
		{"invalid no domain", "alice@", true},
		{"empty", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateEmail(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateEmail(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestValidatePassword(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid", "password123", false},
		{"too short", "pass", true},
		{"empty", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePassword(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePassword(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestValidateRepoName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid", "my-repo", false},
		{"valid with dots", "my.repo.name", false},
		{"invalid start hyphen", "-myrepo", true},
		{"invalid end hyphen", "myrepo-", true},
		{"empty", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRepoName(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateRepoName(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestSanitizeString(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"trim spaces", "  hello  ", "hello"},
		{"remove null bytes", "hello\x00world", "helloworld"},
		{"normal", "hello world", "hello world"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SanitizeString(tt.input)
			if got != tt.want {
				t.Errorf("SanitizeString(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestCSRFProtector(t *testing.T) {
	protector := NewCSRFProtector(time.Second)

	token := protector.GenerateToken()
	if token == "" {
		t.Fatal("expected non-empty token")
	}

	if !protector.ValidateToken(token) {
		t.Fatal("expected token to be valid")
	}

	protector.InvalidateToken(token)

	if protector.ValidateToken(token) {
		t.Fatal("expected token to be invalid after invalidation")
	}
}

func TestCSRFProtector_Expiration(t *testing.T) {
	protector := NewCSRFProtector(50 * time.Millisecond)

	token := protector.GenerateToken()

	time.Sleep(100 * time.Millisecond)

	if protector.ValidateToken(token) {
		t.Fatal("expected token to be expired")
	}
}
