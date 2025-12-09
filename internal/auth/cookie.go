package auth

import (
	"net/http"
	"os"
	"time"
)

// Cookie configuration constants
const (
	// RefreshTokenCookieName is the name of the refresh token cookie
	RefreshTokenCookieName = "refresh_token"
	// RefreshTokenCookiePath is the path for the refresh token cookie
	RefreshTokenCookiePath = "/api/v1/auth"
)

// CookieConfig holds cookie security configuration
type CookieConfig struct {
	// Domain for the cookie (empty = current domain only)
	Domain string
	// Secure requires HTTPS (should be true in production)
	Secure bool
	// SameSite policy (Strict recommended for auth cookies)
	SameSite http.SameSite
}

// DefaultCookieConfig returns secure defaults based on environment
func DefaultCookieConfig() *CookieConfig {
	isProduction := os.Getenv("APP_ENV") == "production" || os.Getenv("APP_ENV") == "prod"

	return &CookieConfig{
		Domain:   "", // Current domain only
		Secure:   isProduction,
		SameSite: http.SameSiteStrictMode,
	}
}

// SetRefreshTokenCookie sets the refresh token as an httpOnly secure cookie
func SetRefreshTokenCookie(w http.ResponseWriter, token string, expiry time.Time, config *CookieConfig) {
	if config == nil {
		config = DefaultCookieConfig()
	}

	cookie := &http.Cookie{
		Name:     RefreshTokenCookieName,
		Value:    token,
		Path:     RefreshTokenCookiePath,
		Domain:   config.Domain,
		Expires:  expiry,
		MaxAge:   int(time.Until(expiry).Seconds()),
		HttpOnly: true,                    // Cannot be accessed by JavaScript
		Secure:   config.Secure,           // HTTPS only in production
		SameSite: config.SameSite,         // Strict = no cross-site requests
	}

	http.SetCookie(w, cookie)
}

// ClearRefreshTokenCookie removes the refresh token cookie
func ClearRefreshTokenCookie(w http.ResponseWriter, config *CookieConfig) {
	if config == nil {
		config = DefaultCookieConfig()
	}

	cookie := &http.Cookie{
		Name:     RefreshTokenCookieName,
		Value:    "",
		Path:     RefreshTokenCookiePath,
		Domain:   config.Domain,
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   config.Secure,
		SameSite: config.SameSite,
	}

	http.SetCookie(w, cookie)
}

// GetRefreshTokenFromCookie extracts the refresh token from the request cookie
func GetRefreshTokenFromCookie(r *http.Request) (string, error) {
	cookie, err := r.Cookie(RefreshTokenCookieName)
	if err != nil {
		return "", err
	}
	return cookie.Value, nil
}
