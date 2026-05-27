package identity

import (
	"crypto/ed25519"
	"fmt"
	"strings"
)

// VerifyProofChain recursively verifies a UCAN's proof chain.
// Each proof's audience must match the current UCAN's issuer (delegation chain).
// maxDepth limits recursion (default 10 if <= 0).
func VerifyProofChain(token string, maxDepth int) error {
	if maxDepth <= 0 {
		maxDepth = 10
	}
	return verifyProofChainRecursive(token, maxDepth, 0)
}

func verifyProofChainRecursive(token string, maxDepth, depth int) error {
	if depth >= maxDepth {
		return fmt.Errorf("proof chain exceeds max depth of %d", maxDepth)
	}

	ucan, payload, sig, err := splitUCAN(token)
	if err != nil {
		return fmt.Errorf("decoding token at depth %d: %w", depth, err)
	}

	// Extract issuer public key and verify signature
	pubKey, err := ExtractPublicKeyFromDID(ucan.Issuer)
	if err != nil {
		return fmt.Errorf("resolving issuer key at depth %d: %w", depth, err)
	}

	if !ed25519.Verify(pubKey, payload, sig) {
		return fmt.Errorf("invalid signature at depth %d", depth)
	}

	// Validate time bounds
	if err := ucan.Validate(); err != nil {
		return fmt.Errorf("time validation at depth %d: %w", depth, err)
	}

	// Recursively verify proofs
	for i, proofToken := range ucan.Proofs {
		proofUCAN, err := decodeAnyUCAN(proofToken)
		if err != nil {
			return fmt.Errorf("decoding proof[%d] at depth %d: %w", i, depth, err)
		}

		// The proof's audience must match this UCAN's issuer (delegation chain)
		if proofUCAN.Audience != ucan.Issuer {
			return fmt.Errorf("proof[%d] audience %q does not match issuer %q at depth %d",
				i, proofUCAN.Audience, ucan.Issuer, depth)
		}

		// Attenuation: child capabilities must be a subset of the proof's capabilities
		if !CapabilitiesSubset(ucan.Caps, proofUCAN.Caps) {
			return fmt.Errorf("proof[%d] attenuation violation at depth %d: child capabilities exceed parent", i, depth)
		}

		if err := verifyProofChainRecursive(proofToken, maxDepth, depth+1); err != nil {
			return err
		}
	}

	return nil
}

// decodeAnyUCAN decodes a UCAN token that may be signed (has dot) or unsigned (base64 only).
func decodeAnyUCAN(token string) (*UCAN, error) {
	if strings.Contains(token, ".") {
		ucan, _, _, err := splitUCAN(token)
		return ucan, err
	}
	return DecodeUCAN(token)
}

// VerifySignedUCANWithChain verifies a UCAN signature, checks revocation,
// validates proof chain, and optionally checks replay via NonceCache.
func VerifySignedUCANWithChain(token string, revocations *RevocationStore, nonces *NonceCache) (*UCAN, error) {
	ucan, payload, sig, err := splitUCAN(token)
	if err != nil {
		return nil, err
	}

	pubKey, err := ExtractPublicKeyFromDID(ucan.Issuer)
	if err != nil {
		return nil, fmt.Errorf("resolving issuer key: %w", err)
	}

	if !ed25519.Verify(pubKey, payload, sig) {
		return nil, fmt.Errorf("invalid signature")
	}

	if err := ucan.Validate(); err != nil {
		return nil, err
	}

	// Check revocation
	if revocations != nil && revocations.IsRevoked(ucan.Nonce) {
		return nil, fmt.Errorf("UCAN has been revoked")
	}

	// Replay protection: reject if this nonce was already used
	if nonces != nil && !nonces.Check(ucan.Nonce) {
		return nil, fmt.Errorf("UCAN replay detected: nonce already used")
	}

	// Verify proof chain
	if len(ucan.Proofs) > 0 {
		if err := VerifyProofChain(token, 10); err != nil {
			return nil, fmt.Errorf("proof chain validation failed: %w", err)
		}
	}

	return ucan, nil
}
