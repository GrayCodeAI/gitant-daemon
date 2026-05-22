package identity

import (
	"os"
	"testing"
	"time"
)

func TestProofChainValidation(t *testing.T) {
	// Create identities: root -> intermediate -> leaf
	root, err := NewIdentity()
	if err != nil {
		t.Fatal(err)
	}
	intermediate, err := NewIdentity()
	if err != nil {
		t.Fatal(err)
	}
	leaf, err := NewIdentity()
	if err != nil {
		t.Fatal(err)
	}

	// Root delegates to intermediate (signed proof)
	rootUCAN := NewUCAN(root.DID, intermediate.DID, []Capability{
		{Resource: "repo:test", Actions: []string{"read", "write"}},
	}, 1*time.Hour)
	rootProof, err := rootUCAN.Sign(root)
	if err != nil {
		t.Fatal(err)
	}

	// Intermediate delegates to leaf, with root's signed UCAN as proof
	leafUCAN := &UCAN{
		Issuer:    intermediate.DID,
		Audience:  leaf.DID,
		NotBefore: time.Now().Unix(),
		Expires:   time.Now().Add(30 * time.Minute).Unix(),
		Caps: []Capability{
			{Resource: "repo:test", Actions: []string{"read"}},
		},
		Proofs: []string{rootProof},
		Nonce:  generateNonce(),
	}
	leafToken, err := leafUCAN.Sign(intermediate)
	if err != nil {
		t.Fatal(err)
	}

	// Verify proof chain
	if err := VerifyProofChain(leafToken, 10); err != nil {
		t.Fatalf("proof chain should be valid: %v", err)
	}
}

func TestProofChainAudienceMismatch(t *testing.T) {
	alice, err := NewIdentity()
	if err != nil {
		t.Fatal(err)
	}
	bob, err := NewIdentity()
	if err != nil {
		t.Fatal(err)
	}
	charlie, err := NewIdentity()
	if err != nil {
		t.Fatal(err)
	}

	// Alice delegates to Charlie (not Bob)
	wrongProof := NewUCAN(alice.DID, charlie.DID, []Capability{
		{Resource: "repo:test", Actions: []string{"read"}},
	}, 1*time.Hour)
	proofToken, err := wrongProof.Sign(alice)
	if err != nil {
		t.Fatal(err)
	}

	// Bob delegates to Charlie with Alice's proof (audience mismatch)
	ucan := &UCAN{
		Issuer:    bob.DID,
		Audience:  charlie.DID,
		NotBefore: time.Now().Unix(),
		Expires:   time.Now().Add(30 * time.Minute).Unix(),
		Caps: []Capability{
			{Resource: "repo:test", Actions: []string{"read"}},
		},
		Proofs: []string{proofToken},
		Nonce:  generateNonce(),
	}
	token, err := ucan.Sign(bob)
	if err != nil {
		t.Fatal(err)
	}

	// Should fail: proof audience (Charlie) != issuer (Bob)
	if err := VerifyProofChain(token, 10); err == nil {
		t.Fatal("expected audience mismatch error")
	}
}

func TestProofChainMaxDepth(t *testing.T) {
	// Chain: A -> B -> C -> D (3 delegation hops)
	a, _ := NewIdentity()
	b, _ := NewIdentity()
	c, _ := NewIdentity()
	d, _ := NewIdentity()

	// A -> B (root proof, signed)
	aUCAN := NewUCAN(a.DID, b.DID, []Capability{
		{Resource: "repo:test", Actions: []string{"read"}},
	}, 1*time.Hour)
	aProof, err := aUCAN.Sign(a)
	if err != nil {
		t.Fatal(err)
	}

	// B -> C with A's proof (signed)
	bUCAN := NewUCAN(b.DID, c.DID, []Capability{
		{Resource: "repo:test", Actions: []string{"read"}},
	}, 1*time.Hour)
	bUCAN.Proofs = []string{aProof}
	bProof, err := bUCAN.Sign(b)
	if err != nil {
		t.Fatal(err)
	}

	// C -> D with B's proof (signed)
	cUCAN := NewUCAN(c.DID, d.DID, []Capability{
		{Resource: "repo:test", Actions: []string{"read"}},
	}, 1*time.Hour)
	cUCAN.Proofs = []string{bProof}
	cToken, err := cUCAN.Sign(c)
	if err != nil {
		t.Fatal(err)
	}

	// Should fail with max depth 2 (chain depth is 3: C->B->A)
	if err := VerifyProofChain(cToken, 2); err == nil {
		t.Fatal("expected max depth error")
	}

	// Should pass with max depth 10
	if err := VerifyProofChain(cToken, 10); err != nil {
		t.Fatalf("should pass with max depth 10: %v", err)
	}
}

func TestRevocationStore(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "gitant-revocation-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	store := NewRevocationStore(tmpDir)

	// Initially empty
	if store.IsRevoked("nonce1") {
		t.Fatal("nonce1 should not be revoked initially")
	}

	// Revoke
	store.Revoke("nonce1")
	if !store.IsRevoked("nonce1") {
		t.Fatal("nonce1 should be revoked")
	}
	if store.IsRevoked("nonce2") {
		t.Fatal("nonce2 should not be revoked")
	}

	// List
	list := store.List()
	if len(list) != 1 {
		t.Fatalf("expected 1 revocation, got %d", len(list))
	}
	if _, ok := list["nonce1"]; !ok {
		t.Fatal("expected nonce1 in list")
	}
}

func TestRevocationStorePersistence(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "gitant-revocation-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create and save
	store1 := NewRevocationStore(tmpDir)
	store1.Revoke("abc123")
	store1.Revoke("def456")
	if err := store1.Save(); err != nil {
		t.Fatal(err)
	}

	// Load into new store
	store2 := NewRevocationStore(tmpDir)
	if err := store2.Load(); err != nil {
		t.Fatal(err)
	}

	if !store2.IsRevoked("abc123") {
		t.Fatal("abc123 should be revoked after reload")
	}
	if !store2.IsRevoked("def456") {
		t.Fatal("def456 should be revoked after reload")
	}
	if store2.IsRevoked("xyz") {
		t.Fatal("xyz should not be revoked")
	}
}

func TestRevocationStoreLoadNonExistent(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "gitant-revocation-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	store := NewRevocationStore(tmpDir)
	// Loading a non-existent file should not error
	if err := store.Load(); err != nil {
		t.Fatalf("Load should not error on missing file: %v", err)
	}
}

func TestVerifySignedUCANWithChainNoRevocation(t *testing.T) {
	issuer, err := NewIdentity()
	if err != nil {
		t.Fatal(err)
	}
	audience, err := NewIdentity()
	if err != nil {
		t.Fatal(err)
	}

	ucan := NewUCAN(issuer.DID, audience.DID, []Capability{
		{Resource: "repo:test", Actions: []string{"read"}},
	}, 1*time.Hour)
	token, err := ucan.Sign(issuer)
	if err != nil {
		t.Fatal(err)
	}

	// Should pass with nil revocation store
	result, err := VerifySignedUCANWithChain(token, nil)
	if err != nil {
		t.Fatalf("verification should pass: %v", err)
	}
	if result.Issuer != issuer.DID {
		t.Fatal("issuer mismatch")
	}
}

func TestVerifySignedUCANWithChainRevoked(t *testing.T) {
	issuer, err := NewIdentity()
	if err != nil {
		t.Fatal(err)
	}
	audience, err := NewIdentity()
	if err != nil {
		t.Fatal(err)
	}

	ucan := NewUCAN(issuer.DID, audience.DID, []Capability{
		{Resource: "repo:test", Actions: []string{"read"}},
	}, 1*time.Hour)
	token, err := ucan.Sign(issuer)
	if err != nil {
		t.Fatal(err)
	}

	tmpDir, err := os.MkdirTemp("", "gitant-revocation-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	store := NewRevocationStore(tmpDir)
	store.Revoke(ucan.Nonce)

	// Should fail: UCAN is revoked
	_, err = VerifySignedUCANWithChain(token, store)
	if err == nil {
		t.Fatal("expected revocation error")
	}
}

func TestVerifySignedUCANWithChainAndProofs(t *testing.T) {
	root, err := NewIdentity()
	if err != nil {
		t.Fatal(err)
	}
	intermediate, err := NewIdentity()
	if err != nil {
		t.Fatal(err)
	}
	leaf, err := NewIdentity()
	if err != nil {
		t.Fatal(err)
	}

	// Root delegates to intermediate (signed proof)
	rootUCAN := NewUCAN(root.DID, intermediate.DID, []Capability{
		{Resource: "repo:test", Actions: []string{"read", "write"}},
	}, 1*time.Hour)
	rootProof, err := rootUCAN.Sign(root)
	if err != nil {
		t.Fatal(err)
	}

	// Intermediate delegates to leaf with root's proof
	leafUCAN := &UCAN{
		Issuer:    intermediate.DID,
		Audience:  leaf.DID,
		NotBefore: time.Now().Unix(),
		Expires:   time.Now().Add(30 * time.Minute).Unix(),
		Caps: []Capability{
			{Resource: "repo:test", Actions: []string{"read"}},
		},
		Proofs: []string{rootProof},
		Nonce:  generateNonce(),
	}
	token, err := leafUCAN.Sign(intermediate)
	if err != nil {
		t.Fatal(err)
	}

	result, err := VerifySignedUCANWithChain(token, nil)
	if err != nil {
		t.Fatalf("verification should pass: %v", err)
	}
	if len(result.Proofs) != 1 {
		t.Fatal("expected 1 proof")
	}
}
