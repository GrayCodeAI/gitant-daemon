package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

// generateID returns a cryptographically random identifier with the given prefix.
// Example output: "issue-a3f8b2c1d4e5f607"
func generateID(prefix string) string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%s-%s", prefix, hex.EncodeToString(b))
}
