package idaustria

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Client provides access to ID Austria OIDC
type Client struct {
	issuer       string
	clientID     string
	clientSecret string
	redirectURL  string
	scopes       []string
	httpClient   *http.Client

	// Cached OIDC configuration
	config *OIDCConfig
}

// ClientOption is a functional option for configuring the client
type ClientOption func(*Client)

// WithHTTPClient sets a custom HTTP client
func WithHTTPClient(client *http.Client) ClientOption {
	return func(c *Client) {
		c.httpClient = client
	}
}

// WithScopes sets the requested scopes
func WithScopes(scopes []string) ClientOption {
	return func(c *Client) {
		c.scopes = scopes
	}
}

// NewClient creates a new ID Austria OIDC client
func NewClient(issuer, clientID, clientSecret, redirectURL string, opts ...ClientOption) *Client {
	c := &Client{
		issuer:       issuer,
		clientID:     clientID,
		clientSecret: clientSecret,
		redirectURL:  redirectURL,
		scopes:       []string{ScopeOpenID, ScopeProfile, ScopeSignature},
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

// DiscoverConfig fetches the OIDC discovery document
func (c *Client) DiscoverConfig(ctx context.Context) (*OIDCConfig, error) {
	if c.config != nil {
		return c.config, nil
	}

	wellKnownURL := strings.TrimSuffix(c.issuer, "/") + "/.well-known/openid-configuration"

	req, err := http.NewRequestWithContext(ctx, "GET", wellKnownURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch discovery document: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("discovery endpoint returned status %d", resp.StatusCode)
	}

	var config OIDCConfig
	if err := json.NewDecoder(resp.Body).Decode(&config); err != nil {
		return nil, fmt.Errorf("failed to decode discovery document: %w", err)
	}

	c.config = &config
	return &config, nil
}

// CreateAuthorizationRequest creates a new authorization request with PKCE
func (c *Client) CreateAuthorizationRequest(redirectAfter string) (*AuthorizationRequest, error) {
	state, err := generateRandomString(32)
	if err != nil {
		return nil, fmt.Errorf("failed to generate state: %w", err)
	}

	nonce, err := generateRandomString(32)
	if err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	codeVerifier, err := generateRandomString(64)
	if err != nil {
		return nil, fmt.Errorf("failed to generate code verifier: %w", err)
	}

	codeChallenge := generateCodeChallenge(codeVerifier)

	return &AuthorizationRequest{
		State:         state,
		Nonce:         nonce,
		CodeVerifier:  codeVerifier,
		CodeChallenge: codeChallenge,
		RedirectAfter: redirectAfter,
		Scopes:        c.scopes,
	}, nil
}

// AuthorizationURL generates the authorization URL for the user to visit
func (c *Client) AuthorizationURL(ctx context.Context, authReq *AuthorizationRequest) (string, error) {
	config, err := c.DiscoverConfig(ctx)
	if err != nil {
		return "", err
	}

	params := url.Values{}
	params.Set("client_id", c.clientID)
	params.Set("response_type", "code")
	params.Set("redirect_uri", c.redirectURL)
	params.Set("scope", strings.Join(authReq.Scopes, " "))
	params.Set("state", authReq.State)
	params.Set("nonce", authReq.Nonce)
	params.Set("code_challenge", authReq.CodeChallenge)
	params.Set("code_challenge_method", "S256")

	return config.AuthorizationEndpoint + "?" + params.Encode(), nil
}

// ExchangeCode exchanges an authorization code for tokens
func (c *Client) ExchangeCode(ctx context.Context, code, codeVerifier string) (*Token, error) {
	config, err := c.DiscoverConfig(ctx)
	if err != nil {
		return nil, err
	}

	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("client_id", c.clientID)
	data.Set("client_secret", c.clientSecret)
	data.Set("code", code)
	data.Set("redirect_uri", c.redirectURL)
	data.Set("code_verifier", codeVerifier)

	req, err := http.NewRequestWithContext(ctx, "POST", config.TokenEndpoint, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("token request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var errResp struct {
			Error       string `json:"error"`
			Description string `json:"error_description"`
		}
		if err := json.Unmarshal(body, &errResp); err != nil {
			return nil, fmt.Errorf("token request failed with status %d: %s", resp.StatusCode, string(body))
		}
		return nil, &OIDCError{
			Code:        errResp.Error,
			Description: errResp.Description,
		}
	}

	var token Token
	if err := json.Unmarshal(body, &token); err != nil {
		return nil, fmt.Errorf("failed to decode token response: %w", err)
	}

	token.ExpiresAt = time.Now().Add(time.Duration(token.ExpiresIn) * time.Second)

	return &token, nil
}

// GetUserInfo retrieves user information using the access token
func (c *Client) GetUserInfo(ctx context.Context, accessToken string) (*UserInfo, error) {
	config, err := c.DiscoverConfig(ctx)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "GET", config.UserInfoEndpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("userinfo request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("userinfo request failed with status %d", resp.StatusCode)
	}

	var userInfo UserInfo
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return nil, fmt.Errorf("failed to decode userinfo response: %w", err)
	}

	return &userInfo, nil
}

// RefreshToken refreshes an access token using a refresh token
func (c *Client) RefreshToken(ctx context.Context, refreshToken string) (*Token, error) {
	config, err := c.DiscoverConfig(ctx)
	if err != nil {
		return nil, err
	}

	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("client_id", c.clientID)
	data.Set("client_secret", c.clientSecret)
	data.Set("refresh_token", refreshToken)

	req, err := http.NewRequestWithContext(ctx, "POST", config.TokenEndpoint, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("refresh request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var errResp struct {
			Error       string `json:"error"`
			Description string `json:"error_description"`
		}
		if err := json.Unmarshal(body, &errResp); err != nil {
			return nil, fmt.Errorf("refresh request failed with status %d", resp.StatusCode)
		}
		return nil, &OIDCError{
			Code:        errResp.Error,
			Description: errResp.Description,
		}
	}

	var token Token
	if err := json.Unmarshal(body, &token); err != nil {
		return nil, fmt.Errorf("failed to decode token response: %w", err)
	}

	token.ExpiresAt = time.Now().Add(time.Duration(token.ExpiresIn) * time.Second)

	return &token, nil
}

// ValidateCallback validates the callback parameters (state and code)
func (c *Client) ValidateCallback(expectedState, receivedState, code, errorCode, errorDescription string) error {
	if errorCode != "" {
		return &OIDCError{
			Code:        errorCode,
			Description: errorDescription,
		}
	}

	if expectedState != receivedState {
		return &OIDCError{
			Code:        ErrCodeInvalidRequest,
			Description: "state mismatch",
		}
	}

	if code == "" {
		return &OIDCError{
			Code:        ErrCodeInvalidRequest,
			Description: "authorization code missing",
		}
	}

	return nil
}

// OIDCError represents an OIDC error
type OIDCError struct {
	Code        string
	Description string
}

func (e *OIDCError) Error() string {
	if e.Description != "" {
		return fmt.Sprintf("OIDC error %s: %s", e.Code, e.Description)
	}
	return fmt.Sprintf("OIDC error: %s", e.Code)
}

// IsRetryable returns true if the error might be temporary
func (e *OIDCError) IsRetryable() bool {
	switch e.Code {
	case ErrCodeServerError, ErrCodeTemporarilyUnavail:
		return true
	default:
		return false
	}
}

// IsUserActionRequired returns true if user action is needed
func (e *OIDCError) IsUserActionRequired() bool {
	switch e.Code {
	case ErrCodeAccessDenied, ErrCodeLoginRequired, ErrCodeConsentRequired, ErrCodeInteractionRequired:
		return true
	default:
		return false
	}
}

// Helper functions

func generateRandomString(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(bytes)[:length], nil
}

func generateCodeChallenge(verifier string) string {
	hash := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(hash[:])
}
