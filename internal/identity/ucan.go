package identity

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"
)

// Capability represents a UCAN capability
type Capability struct {
	Resource string   `json:"resource"`
	Actions  []string `json:"actions"`
}

// UCAN represents a UCAN token
type UCAN struct {
	Issuer    string       `json:"issuer"`
	Audience  string       `json:"audience"`
	NotBefore int64        `json:"nbf"`
	Expires   int64        `json:"exp"`
	Caps      []Capability `json:"caps"`
	Proofs    []string     `json:"proofs,omitempty"`
	Nonce     string       `json:"nonce"`
}

// NewUCAN creates a new UCAN token
func NewUCAN(issuer, audience string, caps []Capability, duration time.Duration) *UCAN {
	now := time.Now()

	return &UCAN{
		Issuer:    issuer,
		Audience:  audience,
		NotBefore: now.Unix(),
		Expires:   now.Add(duration).Unix(),
		Caps:      caps,
		Nonce:     generateNonce(),
	}
}

// Encode encodes the UCAN to a string
func (u *UCAN) Encode() (string, error) {
	data, err := json.Marshal(u)
	if err != nil {
		return "", fmt.Errorf("marshaling UCAN: %w", err)
	}

	// TODO: Sign the UCAN with the issuer's key
	// For now, return base64 encoded JSON
	return base64Encode(data), nil
}

// DecodeUCAN decodes a UCAN from a string
func DecodeUCAN(encoded string) (*UCAN, error) {
	data, err := base64Decode(encoded)
	if err != nil {
		return nil, fmt.Errorf("decoding UCAN: %w", err)
	}

	var ucan UCAN
	if err := json.Unmarshal(data, &ucan); err != nil {
		return nil, fmt.Errorf("unmarshaling UCAN: %w", err)
	}

	return &ucan, nil
}

// Validate validates the UCAN
func (u *UCAN) Validate() error {
	now := time.Now().Unix()

	if u.NotBefore > now {
		return fmt.Errorf("UCAN is not yet valid")
	}

	if u.Expires < now {
		return fmt.Errorf("UCAN has expired")
	}

	return nil
}

// HasCapability checks if the UCAN has a specific capability
func (u *UCAN) HasCapability(resource, action string) bool {
	for _, cap := range u.Caps {
		if cap.Resource == resource || cap.Resource == "*" {
			for _, a := range cap.Actions {
				if a == action || a == "*" {
					return true
				}
			}
		}
	}

	return false
}

// generateNonce generates a random nonce
func generateNonce() string {
	b := make([]byte, 16)
	rand.Read(b)
	return fmt.Sprintf("%x", b)
}

func base64Encode(data []byte) string {
	return base64.RawURLEncoding.EncodeToString(data)
}

func base64Decode(s string) ([]byte, error) {
	return base64.RawURLEncoding.DecodeString(s)
}
