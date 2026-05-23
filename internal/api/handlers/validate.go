package handlers

import (
	"net/http"
	"regexp"
	"strings"
)

var validIDPattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]{0,63}$`)

// ValidateID checks if a string is a safe identifier (repo names, branch names, etc.)
// Rejects path traversal attempts and special characters.
func ValidateID(id string) bool {
	if id == "" || len(id) > 64 {
		return false
	}
	if strings.Contains(id, "..") || strings.Contains(id, "/") || strings.Contains(id, "\\") || strings.Contains(id, "\x00") {
		return false
	}
	return validIDPattern.MatchString(id)
}

// ValidateRepoID validates and returns an error response if invalid
func ValidateRepoID(w http.ResponseWriter, id string) bool {
	if !ValidateID(id) {
		http.Error(w, "Invalid repository ID", http.StatusBadRequest)
		return false
	}
	return true
}

// SanitizeError returns a safe error message for API clients.
// Internal errors are logged server-side; only generic messages go to clients.
func SanitizeError(err error, fallback string) string {
	if err == nil {
		return fallback
	}
	errMsg := err.Error()
	// Map known error patterns to safe messages
	if strings.Contains(errMsg, "not found") {
		return errMsg // "not found" is safe to expose
	}
	if strings.Contains(errMsg, "already exists") {
		return errMsg // "already exists" is safe to expose
	}
	if strings.Contains(errMsg, "invalid") || strings.Contains(errMsg, "required") {
		return errMsg // validation errors are safe to expose
	}
	// For everything else, return generic message
	return fallback
}
