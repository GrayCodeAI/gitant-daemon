package security

import (
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"
)

var (
	// UsernamePattern defines valid usernames
	UsernamePattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]{1,38}[a-zA-Z0-9]$`)
	
	// EmailPattern defines valid emails
	EmailPattern = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	
	// RepoNamePattern defines valid repository names
	RepoNamePattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]{0,98}[a-zA-Z0-9]$`)
	
	// BranchPattern defines valid branch names
	BranchPattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._/-]{0,253}[a-zA-Z0-9]$`)
	
	// TagPattern defines valid tag names
	TagPattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]{0,253}[a-zA-Z0-9]$`)
	
	// SafeStringPattern allows only safe characters
	SafeStringPattern = regexp.MustCompile(`^[a-zA-Z0-9\s._-]+$`)
)

// ValidationError represents a validation error
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// ValidateUsername validates a username
func ValidateUsername(username string) error {
	if username == "" {
		return &ValidationError{Field: "username", Message: "is required"}
	}
	if utf8.RuneCountInString(username) < 3 {
		return &ValidationError{Field: "username", Message: "must be at least 3 characters"}
	}
	if utf8.RuneCountInString(username) > 39 {
		return &ValidationError{Field: "username", Message: "must be 39 characters or less"}
	}
	if !UsernamePattern.MatchString(username) {
		return &ValidationError{Field: "username", Message: "contains invalid characters"}
	}
	return nil
}

// ValidateEmail validates an email
func ValidateEmail(email string) error {
	if email == "" {
		return &ValidationError{Field: "email", Message: "is required"}
	}
	if utf8.RuneCountInString(email) > 255 {
		return &ValidationError{Field: "email", Message: "must be 255 characters or less"}
	}
	if !EmailPattern.MatchString(email) {
		return &ValidationError{Field: "email", Message: "is invalid"}
	}
	return nil
}

// ValidatePassword validates a password
func ValidatePassword(password string) error {
	if password == "" {
		return &ValidationError{Field: "password", Message: "is required"}
	}
	if utf8.RuneCountInString(password) < 8 {
		return &ValidationError{Field: "password", Message: "must be at least 8 characters"}
	}
	if utf8.RuneCountInString(password) > 128 {
		return &ValidationError{Field: "password", Message: "must be 128 characters or less"}
	}
	return nil
}

// ValidateRepoName validates a repository name
func ValidateRepoName(name string) error {
	if name == "" {
		return &ValidationError{Field: "name", Message: "is required"}
	}
	if utf8.RuneCountInString(name) > 100 {
		return &ValidationError{Field: "name", Message: "must be 100 characters or less"}
	}
	if !RepoNamePattern.MatchString(name) {
		return &ValidationError{Field: "name", Message: "contains invalid characters"}
	}
	return nil
}

// ValidateBranchName validates a branch name
func ValidateBranchName(name string) error {
	if name == "" {
		return &ValidationError{Field: "branch", Message: "is required"}
	}
	if utf8.RuneCountInString(name) > 255 {
		return &ValidationError{Field: "branch", Message: "must be 255 characters or less"}
	}
	if !BranchPattern.MatchString(name) {
		return &ValidationError{Field: "branch", Message: "contains invalid characters"}
	}
	return nil
}

// ValidateTagName validates a tag name
func ValidateTagName(name string) error {
	if name == "" {
		return &ValidationError{Field: "tag", Message: "is required"}
	}
	if utf8.RuneCountInString(name) > 255 {
		return &ValidationError{Field: "tag", Message: "must be 255 characters or less"}
	}
	if !TagPattern.MatchString(name) {
		return &ValidationError{Field: "tag", Message: "contains invalid characters"}
	}
	return nil
}

// SanitizeString sanitizes a string input
func SanitizeString(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "\x00", "") // Remove null bytes
	return s
}

// ValidateStringLength validates string length
func ValidateStringLength(field, s string, min, max int) error {
	length := utf8.RuneCountInString(s)
	if length < min {
		return &ValidationError{Field: field, Message: fmt.Sprintf("must be at least %d characters", min)}
	}
	if length > max {
		return &ValidationError{Field: field, Message: fmt.Sprintf("must be %d characters or less", max)}
	}
	return nil
}

// SanitizeHTML removes potentially dangerous HTML
func SanitizeHTML(s string) string {
	// Simple sanitization - remove script tags and event handlers
	s = regexp.MustCompile(`(?i)<script[^>]*>.*?</script>`).ReplaceAllString(s, "")
	s = regexp.MustCompile(`(?i)<[^>]*on\w+="[^"]*"[^>]*>`).ReplaceAllString(s, "")
	s = regexp.MustCompile(`(?i)<[^>]*on\w+='[^']*'[^>]*>`).ReplaceAllString(s, "")
	return s
}

// ValidateJSON validates that a string is valid JSON
func ValidateJSON(s string) error {
	if !strings.HasPrefix(s, "{") && !strings.HasPrefix(s, "[") {
		return &ValidationError{Field: "json", Message: "must be valid JSON"}
	}
	return nil
}
