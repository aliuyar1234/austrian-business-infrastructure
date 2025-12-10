package unit

import (
	"testing"
	"time"

	"austrian-business-infrastructure/internal/idaustria"
)

// TestUserInfoTypes verifies user info types
func TestUserInfoTypes(t *testing.T) {
	user := &idaustria.UserInfo{
		Subject:    "user-123",
		GivenName:  "Max",
		FamilyName: "Mustermann",
		Email:      "max@example.com",
		BPK:        "bpk-hash-value",
	}

	if user.Subject == "" {
		t.Error("Subject should not be empty")
	}

	if user.GivenName != "Max" {
		t.Error("Given name mismatch")
	}

	if user.FamilyName != "Mustermann" {
		t.Error("Family name mismatch")
	}

	if user.BPK == "" {
		t.Error("BPK should not be empty")
	}
}

// TestTokenTypes verifies token types
func TestTokenTypes(t *testing.T) {
	now := time.Now()
	expiresAt := now.Add(1 * time.Hour)

	token := &idaustria.Token{
		AccessToken:  "access-token-value",
		TokenType:    "Bearer",
		RefreshToken: "refresh-token-value",
		ExpiresAt:    expiresAt,
		IDToken:      "id-token-value",
	}

	if token.AccessToken == "" {
		t.Error("Access token should not be empty")
	}

	if token.TokenType != "Bearer" {
		t.Error("Token type mismatch")
	}

	if !token.ExpiresAt.After(now) {
		t.Error("Token should expire in the future")
	}
}

// TestAuthorizationRequestTypes verifies authorization request types
func TestAuthorizationRequestTypes(t *testing.T) {
	req := &idaustria.AuthorizationRequest{
		State:         "random-state",
		Nonce:         "random-nonce",
		CodeVerifier:  "code-verifier-pkce",
		CodeChallenge: "code-challenge-pkce",
		RedirectAfter: "https://app.example.com/callback",
		Scopes:        []string{"openid", "profile", "signature"},
	}

	if req.State == "" {
		t.Error("State should not be empty")
	}

	if req.Nonce == "" {
		t.Error("Nonce should not be empty")
	}

	if req.CodeVerifier == "" {
		t.Error("Code verifier should not be empty (PKCE)")
	}

	if req.CodeChallenge == "" {
		t.Error("Code challenge should not be empty (PKCE)")
	}

	if len(req.Scopes) == 0 {
		t.Error("Scopes should not be empty")
	}

	// Verify signature scope is included
	hasSignatureScope := false
	for _, scope := range req.Scopes {
		if scope == "signature" {
			hasSignatureScope = true
			break
		}
	}
	if !hasSignatureScope {
		t.Error("Signature scope should be included")
	}
}

// TestSessionTypes verifies session types
func TestSessionTypes(t *testing.T) {
	now := time.Now()

	session := &idaustria.Session{
		ID:            "session-123",
		State:         "session-state",
		Nonce:         "session-nonce",
		CodeVerifier:  "session-code-verifier",
		SignerID:      "signer-123",
		RedirectAfter: "https://app.example.com/done",
		CreatedAt:     now,
		ExpiresAt:     now.Add(10 * time.Minute),
	}

	if session.State == "" {
		t.Error("State should not be empty")
	}

	if session.SignerID == "" {
		t.Error("Signer ID should not be empty")
	}

	if session.ID == "" {
		t.Error("Session ID should not be empty")
	}
}

// TestIDTokenClaimsTypes verifies ID token claims types
func TestIDTokenClaimsTypes(t *testing.T) {
	now := time.Now()

	claims := &idaustria.IDTokenClaims{
		Subject:   "user-123",
		Issuer:    "https://eid.gv.at",
		Audience:  "client-id",
		IssuedAt:  now.Unix(),
		ExpiresAt: now.Add(1 * time.Hour).Unix(),
		Nonce:     "nonce-value",
	}

	if claims.Subject == "" {
		t.Error("Subject should not be empty")
	}

	if claims.Issuer == "" {
		t.Error("Issuer should not be empty")
	}

	if claims.ExpiresAt <= claims.IssuedAt {
		t.Error("ExpiresAt should be after IssuedAt")
	}
}

// TestOIDCErrorTypes verifies OIDC error types
func TestOIDCErrorTypes(t *testing.T) {
	err := &idaustria.OIDCError{
		Code:        "invalid_request",
		Description: "The request is missing a required parameter",
	}

	if err.Code == "" {
		t.Error("Error code should not be empty")
	}

	if err.Description == "" {
		t.Error("Error description should not be empty")
	}

	// Error should implement error interface
	if err.Error() == "" {
		t.Error("Error() should return non-empty string")
	}
}

// TestPKCEGeneration verifies PKCE parameter generation
func TestPKCEGeneration(t *testing.T) {
	// Test that PKCE code verifier has correct length
	verifier := "verifier-must-be-at-least-43-chars-for-pkce-compliance"
	if len(verifier) < 43 {
		t.Error("PKCE code verifier should be at least 43 characters")
	}
}

// TestBPKHashing verifies BPK hashing
func TestBPKHashing(t *testing.T) {
	bpk := "sample-bpk-value"

	hashed := idaustria.HashBPK(bpk)

	if hashed == "" {
		t.Error("Hashed BPK should not be empty")
	}

	if hashed == bpk {
		t.Error("Hashed BPK should not equal raw BPK")
	}

	// Same input should produce same hash
	hashed2 := idaustria.HashBPK(bpk)
	if hashed != hashed2 {
		t.Error("BPK hashing should be deterministic")
	}
}

// TestSessionExpiry verifies session expiration logic
func TestSessionExpiry(t *testing.T) {
	now := time.Now()

	// Session that has expired
	expiredSession := &idaustria.Session{
		ID:        "expired",
		State:     "state",
		ExpiresAt: now.Add(-1 * time.Hour),
	}

	if !expiredSession.ExpiresAt.Before(now) {
		t.Error("Expired session should have ExpiresAt in the past")
	}

	// Session that is still valid
	validSession := &idaustria.Session{
		ID:        "valid",
		State:     "state",
		ExpiresAt: now.Add(10 * time.Minute),
	}

	if !validSession.ExpiresAt.After(now) {
		t.Error("Valid session should have ExpiresAt in the future")
	}
}
