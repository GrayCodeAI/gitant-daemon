package identity

import (
	"crypto/ed25519"
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

// Encode encodes the UCAN to a signed string
func (u *UCAN) Encode() (string, error) {
	data, err := json.Marshal(u)
	if err != nil {
		return "", fmt.Errorf("marshaling UCAN: %w", err)
	}

	return base64Encode(data), nil
}

// Sign signs the UCAN with the given identity and returns a signed token
func (u *UCAN) Sign(id *Identity) (string, error) {
	data, err := json.Marshal(u)
	if err != nil {
		return "", fmt.Errorf("marshaling UCAN: %w", err)
	}

	sig := id.Sign(data)

	// Format: <base64-payload>.<base64-signature>
	return base64Encode(data) + "." + base64Encode(sig), nil
}

// VerifySignedUCAN verifies a signed UCAN token with a known issuer identity
func VerifySignedUCAN(token string, expectedIssuer *Identity) (*UCAN, error) {
	ucan, payload, sig, err := splitUCAN(token)
	if err != nil {
		return nil, err
	}
	if !expectedIssuer.Verify(payload, sig) {
		return nil, fmt.Errorf("invalid signature")
	}
	return ucan, nil
}

// VerifySignedUCANByKey verifies a signed UCAN using a raw public key (for did:key resolution)
func VerifySignedUCANByKey(token string, pubKey ed25519.PublicKey) (*UCAN, error) {
	ucan, payload, sig, err := splitUCAN(token)
	if err != nil {
		return nil, err
	}
	if !ed25519.Verify(pubKey, payload, sig) {
		return nil, fmt.Errorf("invalid signature")
	}
	return ucan, nil
}

// splitUCAN splits a signed UCAN token into its components
func splitUCAN(token string) (*UCAN, []byte, []byte, error) {
	dotIdx := -1
	for i, c := range token {
		if c == '.' {
			dotIdx = i
			break
		}
	}
	if dotIdx == -1 {
		return nil, nil, nil, fmt.Errorf("invalid token format")
	}

	payload, err := base64Decode(token[:dotIdx])
	if err != nil {
		return nil, nil, nil, fmt.Errorf("decoding payload: %w", err)
	}

	sig, err := base64Decode(token[dotIdx+1:])
	if err != nil {
		return nil, nil, nil, fmt.Errorf("decoding signature: %w", err)
	}

	var ucan UCAN
	if err := json.Unmarshal(payload, &ucan); err != nil {
		return nil, nil, nil, fmt.Errorf("unmarshaling UCAN: %w", err)
	}

	return &ucan, payload, sig, nil
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
