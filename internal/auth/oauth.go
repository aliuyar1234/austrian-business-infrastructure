package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"golang.org/x/oauth2/microsoft"
)

var (
	ErrOAuthDisabled       = errors.New("OAuth is disabled")
	ErrOAuthProviderNotConfigured = errors.New("OAuth provider not configured")
	ErrOAuthStateMismatch  = errors.New("OAuth state mismatch")
	ErrOAuthCodeMissing    = errors.New("OAuth code missing")
	ErrOAuthUserInfo       = errors.New("failed to get OAuth user info")
)

// OAuthProvider represents an OAuth provider type
type OAuthProvider string

const (
	ProviderGoogle    OAuthProvider = "google"
	ProviderMicrosoft OAuthProvider = "microsoft"
)

// OAuthUserInfo contains user info from OAuth provider
type OAuthUserInfo struct {
	ID        string
	Email     string
	Name      string
	AvatarURL string
	Provider  OAuthProvider
}

// OAuthConfig holds OAuth configuration
type OAuthConfig struct {
	Enabled           bool
	GoogleClientID    string
	GoogleSecret      string
	MicrosoftClientID string
	MicrosoftSecret   string
	RedirectBaseURL   string
}

// OAuthManager handles OAuth authentication
type OAuthManager struct {
	config         *OAuthConfig
	googleConfig   *oauth2.Config
	microsoftConfig *oauth2.Config
}

// NewOAuthManager creates a new OAuth manager
func NewOAuthManager(config *OAuthConfig) *OAuthManager {
	m := &OAuthManager{
		config: config,
	}

	if config.GoogleClientID != "" && config.GoogleSecret != "" {
		m.googleConfig = &oauth2.Config{
			ClientID:     config.GoogleClientID,
			ClientSecret: config.GoogleSecret,
			Endpoint:     google.Endpoint,
			RedirectURL:  config.RedirectBaseURL + "/api/v1/auth/oauth/google/callback",
			Scopes:       []string{"openid", "email", "profile"},
		}
	}

	if config.MicrosoftClientID != "" && config.MicrosoftSecret != "" {
		m.microsoftConfig = &oauth2.Config{
			ClientID:     config.MicrosoftClientID,
			ClientSecret: config.MicrosoftSecret,
			Endpoint:     microsoft.AzureADEndpoint("common"),
			RedirectURL:  config.RedirectBaseURL + "/api/v1/auth/oauth/microsoft/callback",
			Scopes:       []string{"openid", "email", "profile", "User.Read"},
		}
	}

	return m
}

// IsEnabled returns whether OAuth is enabled
func (m *OAuthManager) IsEnabled() bool {
	return m.config.Enabled
}

// IsProviderConfigured returns whether a specific provider is configured
func (m *OAuthManager) IsProviderConfigured(provider OAuthProvider) bool {
	switch provider {
	case ProviderGoogle:
		return m.googleConfig != nil
	case ProviderMicrosoft:
		return m.microsoftConfig != nil
	default:
		return false
	}
}

// GetAuthURL returns the OAuth authorization URL for a provider
func (m *OAuthManager) GetAuthURL(provider OAuthProvider, state string) (string, error) {
	if !m.config.Enabled {
		return "", ErrOAuthDisabled
	}

	config, err := m.getConfig(provider)
	if err != nil {
		return "", err
	}

	return config.AuthCodeURL(state, oauth2.AccessTypeOffline), nil
}

// Exchange exchanges the authorization code for tokens and user info
func (m *OAuthManager) Exchange(ctx context.Context, provider OAuthProvider, code string) (*OAuthUserInfo, error) {
	if !m.config.Enabled {
		return nil, ErrOAuthDisabled
	}

	config, err := m.getConfig(provider)
	if err != nil {
		return nil, err
	}

	// Exchange code for token
	token, err := config.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}

	// Get user info based on provider
	switch provider {
	case ProviderGoogle:
		return m.getGoogleUserInfo(ctx, token)
	case ProviderMicrosoft:
		return m.getMicrosoftUserInfo(ctx, token)
	default:
		return nil, ErrOAuthProviderNotConfigured
	}
}

func (m *OAuthManager) getConfig(provider OAuthProvider) (*oauth2.Config, error) {
	switch provider {
	case ProviderGoogle:
		if m.googleConfig == nil {
			return nil, ErrOAuthProviderNotConfigured
		}
		return m.googleConfig, nil
	case ProviderMicrosoft:
		if m.microsoftConfig == nil {
			return nil, ErrOAuthProviderNotConfigured
		}
		return m.microsoftConfig, nil
	default:
		return nil, ErrOAuthProviderNotConfigured
	}
}

// Google user info response
type googleUserInfo struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	VerifiedEmail bool   `json:"verified_email"`
	Name          string `json:"name"`
	Picture       string `json:"picture"`
}

func (m *OAuthManager) getGoogleUserInfo(ctx context.Context, token *oauth2.Token) (*OAuthUserInfo, error) {
	client := m.googleConfig.Client(ctx, token)

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", "https://www.googleapis.com/oauth2/v2/userinfo", nil)
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, ErrOAuthUserInfo
	}

	var info googleUserInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, fmt.Errorf("failed to decode user info: %w", err)
	}

	return &OAuthUserInfo{
		ID:        info.ID,
		Email:     info.Email,
		Name:      info.Name,
		AvatarURL: info.Picture,
		Provider:  ProviderGoogle,
	}, nil
}

// Microsoft user info response
type microsoftUserInfo struct {
	ID                string `json:"id"`
	Mail              string `json:"mail"`
	UserPrincipalName string `json:"userPrincipalName"`
	DisplayName       string `json:"displayName"`
}

func (m *OAuthManager) getMicrosoftUserInfo(ctx context.Context, token *oauth2.Token) (*OAuthUserInfo, error) {
	client := m.microsoftConfig.Client(ctx, token)

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", "https://graph.microsoft.com/v1.0/me", nil)
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, ErrOAuthUserInfo
	}

	var info microsoftUserInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, fmt.Errorf("failed to decode user info: %w", err)
	}

	// Microsoft may return email in either field
	email := info.Mail
	if email == "" {
		email = info.UserPrincipalName
	}

	return &OAuthUserInfo{
		ID:        info.ID,
		Email:     email,
		Name:      info.DisplayName,
		AvatarURL: "", // Microsoft Graph requires separate call for photo
		Provider:  ProviderMicrosoft,
	}, nil
}

// ValidateProvider validates that a provider string is valid
func ValidateProvider(provider string) (OAuthProvider, error) {
	switch provider {
	case "google":
		return ProviderGoogle, nil
	case "microsoft":
		return ProviderMicrosoft, nil
	default:
		return "", fmt.Errorf("invalid OAuth provider: %s", provider)
	}
}
