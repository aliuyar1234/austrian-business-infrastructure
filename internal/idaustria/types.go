package idaustria

import (
	"time"
)

// UserInfo contains the user information from ID Austria
type UserInfo struct {
	// Subject is the unique identifier (sub claim)
	Subject string `json:"sub"`
	// Name is the full name
	Name string `json:"name"`
	// GivenName is the first name
	GivenName string `json:"given_name,omitempty"`
	// FamilyName is the last name
	FamilyName string `json:"family_name,omitempty"`
	// Email is the email address (if requested and provided)
	Email string `json:"email,omitempty"`
	// EmailVerified indicates if the email is verified
	EmailVerified bool `json:"email_verified,omitempty"`
	// BPK is the bereichsspezifisches Personenkennzeichen (sector-specific ID)
	// This is sensitive and should be handled carefully
	BPK string `json:"bpk,omitempty"`
	// BPKType indicates the BPK sector
	BPKType string `json:"bpk_type,omitempty"`
	// DateOfBirth is the user's date of birth
	DateOfBirth string `json:"date_of_birth,omitempty"`
}

// Token represents OAuth2 tokens from ID Austria
type Token struct {
	// AccessToken is the access token for API calls
	AccessToken string `json:"access_token"`
	// TokenType is the token type (usually "Bearer")
	TokenType string `json:"token_type"`
	// ExpiresIn is the number of seconds until the access token expires
	ExpiresIn int `json:"expires_in"`
	// RefreshToken is the refresh token for obtaining new access tokens
	RefreshToken string `json:"refresh_token,omitempty"`
	// IDToken is the OpenID Connect ID token (JWT)
	IDToken string `json:"id_token,omitempty"`
	// Scope is the granted scopes
	Scope string `json:"scope,omitempty"`

	// Calculated fields
	ExpiresAt time.Time `json:"-"`
}

// IsExpired returns true if the token is expired
func (t *Token) IsExpired() bool {
	// Add a small buffer to avoid using tokens right at expiry
	return time.Now().Add(30 * time.Second).After(t.ExpiresAt)
}

// IDTokenClaims represents the claims in the ID token
type IDTokenClaims struct {
	// Standard OIDC claims
	Issuer    string `json:"iss"`
	Subject   string `json:"sub"`
	Audience  string `json:"aud"`
	ExpiresAt int64  `json:"exp"`
	IssuedAt  int64  `json:"iat"`
	Nonce     string `json:"nonce,omitempty"`

	// ID Austria specific claims
	Name          string `json:"name,omitempty"`
	GivenName     string `json:"given_name,omitempty"`
	FamilyName    string `json:"family_name,omitempty"`
	Email         string `json:"email,omitempty"`
	EmailVerified bool   `json:"email_verified,omitempty"`

	// Signature-related claims
	SignerCertID string `json:"signer_cert_id,omitempty"`
	ACR          string `json:"acr,omitempty"` // Authentication Context Class Reference
}

// AuthorizationRequest represents the parameters for an authorization request
type AuthorizationRequest struct {
	// State is a random string for CSRF protection
	State string
	// Nonce is a random string for replay protection (bound to ID token)
	Nonce string
	// CodeVerifier is the PKCE code verifier
	CodeVerifier string
	// CodeChallenge is the PKCE code challenge (SHA256 of verifier)
	CodeChallenge string
	// RedirectAfter is the URL to redirect to after successful authentication
	RedirectAfter string
	// Scopes are the requested OIDC scopes
	Scopes []string
}

// Session represents an ID Austria authentication session
type Session struct {
	// ID is the unique session identifier
	ID string `json:"id"`
	// State is the OIDC state parameter
	State string `json:"state"`
	// Nonce is the OIDC nonce parameter
	Nonce string `json:"nonce"`
	// CodeVerifier is the PKCE code verifier
	CodeVerifier string `json:"code_verifier"`
	// RedirectAfter is where to redirect after authentication
	RedirectAfter string `json:"redirect_after"`
	// CreatedAt is when the session was created
	CreatedAt time.Time `json:"created_at"`
	// ExpiresAt is when the session expires
	ExpiresAt time.Time `json:"expires_at"`

	// Context
	SignerID string `json:"signer_id,omitempty"`
	BatchID  string `json:"batch_id,omitempty"`

	// Result (populated after successful auth)
	Status      SessionStatus `json:"status"`
	UserInfo    *UserInfo     `json:"user_info,omitempty"`
	Token       *Token        `json:"token,omitempty"`
	Error       string        `json:"error,omitempty"`
	AuthenticatedAt time.Time `json:"authenticated_at,omitempty"`
}

// SessionStatus represents the status of an ID Austria session
type SessionStatus string

const (
	SessionStatusPending       SessionStatus = "pending"
	SessionStatusAuthenticated SessionStatus = "authenticated"
	SessionStatusUsed          SessionStatus = "used"
	SessionStatusExpired       SessionStatus = "expired"
	SessionStatusFailed        SessionStatus = "failed"
)

// OIDCConfig represents the OIDC provider configuration
type OIDCConfig struct {
	Issuer                            string   `json:"issuer"`
	AuthorizationEndpoint             string   `json:"authorization_endpoint"`
	TokenEndpoint                     string   `json:"token_endpoint"`
	UserInfoEndpoint                  string   `json:"userinfo_endpoint"`
	JWKSEndpoint                      string   `json:"jwks_uri"`
	ScopesSupported                   []string `json:"scopes_supported"`
	ResponseTypesSupported            []string `json:"response_types_supported"`
	TokenEndpointAuthMethodsSupported []string `json:"token_endpoint_auth_methods_supported"`
	ClaimsSupported                   []string `json:"claims_supported"`
}

// Error codes
const (
	ErrCodeInvalidRequest       = "invalid_request"
	ErrCodeUnauthorizedClient   = "unauthorized_client"
	ErrCodeAccessDenied         = "access_denied"
	ErrCodeUnsupportedResponse  = "unsupported_response_type"
	ErrCodeInvalidScope         = "invalid_scope"
	ErrCodeServerError          = "server_error"
	ErrCodeTemporarilyUnavail   = "temporarily_unavailable"
	ErrCodeInvalidGrant         = "invalid_grant"
	ErrCodeInvalidToken         = "invalid_token"
	ErrCodeLoginRequired        = "login_required"
	ErrCodeConsentRequired      = "consent_required"
	ErrCodeInteractionRequired  = "interaction_required"
)

// Standard scopes
const (
	ScopeOpenID    = "openid"
	ScopeProfile   = "profile"
	ScopeEmail     = "email"
	ScopeSignature = "signature"
	ScopeBPK       = "bpk"
)
