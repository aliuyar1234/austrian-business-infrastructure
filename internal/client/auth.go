package client

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

var (
	ErrUnauthorized     = errors.New("unauthorized")
	ErrTokenExpired     = errors.New("token expired")
	ErrInvalidToken     = errors.New("invalid token")
	ErrNotClientToken   = errors.New("not a client token")
)

// ClientClaims represents JWT claims for a client
type ClientClaims struct {
	jwt.RegisteredClaims
	UserID    uuid.UUID `json:"user_id"`
	ClientID  uuid.UUID `json:"client_id"`
	TenantID  uuid.UUID `json:"tenant_id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	IsClient  bool      `json:"is_client"`
}

// ClientAuth handles client authentication
type ClientAuth struct {
	jwtSecret        []byte
	accessExpiry     time.Duration
	refreshExpiry    time.Duration
}

// NewClientAuth creates a new client authentication handler
func NewClientAuth(jwtSecret string, accessExpiry, refreshExpiry time.Duration) *ClientAuth {
	return &ClientAuth{
		jwtSecret:     []byte(jwtSecret),
		accessExpiry:  accessExpiry,
		refreshExpiry: refreshExpiry,
	}
}

// GenerateTokens generates access and refresh tokens for a client
func (a *ClientAuth) GenerateTokens(client *Client, userID uuid.UUID) (accessToken, refreshToken string, expiresAt time.Time, err error) {
	now := time.Now()
	accessExpiresAt := now.Add(a.accessExpiry)
	refreshExpiresAt := now.Add(a.refreshExpiry)

	// Access token claims
	accessClaims := &ClientClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(accessExpiresAt),
			Issuer:    "austrian-business-portal",
		},
		UserID:   userID,
		ClientID: client.ID,
		TenantID: client.TenantID,
		Email:    client.Email,
		Name:     client.Name,
		IsClient: true,
	}

	accessToken, err = a.signToken(accessClaims)
	if err != nil {
		return "", "", time.Time{}, err
	}

	// Refresh token claims (minimal)
	refreshClaims := &ClientClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(refreshExpiresAt),
			Issuer:    "austrian-business-portal-refresh",
		},
		UserID:   userID,
		ClientID: client.ID,
		TenantID: client.TenantID,
		IsClient: true,
	}

	refreshToken, err = a.signToken(refreshClaims)
	if err != nil {
		return "", "", time.Time{}, err
	}

	return accessToken, refreshToken, accessExpiresAt, nil
}

// ValidateToken validates a JWT token and returns the claims
func (a *ClientAuth) ValidateToken(tokenString string) (*ClientClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &ClientClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return a.jwtSecret, nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*ClientClaims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	if !claims.IsClient {
		return nil, ErrNotClientToken
	}

	return claims, nil
}

func (a *ClientAuth) signToken(claims *ClientClaims) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(a.jwtSecret)
}

// ============== Context Helpers ==============

type contextKey string

const clientContextKey contextKey = "client"

// WithClient adds client claims to context
func WithClient(ctx context.Context, claims *ClientClaims) context.Context {
	return context.WithValue(ctx, clientContextKey, claims)
}

// ClientFromContext retrieves client claims from context
func ClientFromContext(ctx context.Context) (*ClientClaims, bool) {
	claims, ok := ctx.Value(clientContextKey).(*ClientClaims)
	return claims, ok
}

// MustClientFromContext retrieves client claims or panics
func MustClientFromContext(ctx context.Context) *ClientClaims {
	claims, ok := ClientFromContext(ctx)
	if !ok {
		panic("client claims not in context")
	}
	return claims
}

// ============== Middleware ==============

// PortalAuthMiddleware creates middleware that validates client JWT tokens
func PortalAuthMiddleware(auth *ClientAuth) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Try to get token from Authorization header
			tokenString := extractBearerToken(r)
			if tokenString == "" {
				// Try cookie as fallback
				cookie, err := r.Cookie("portal_access_token")
				if err == nil {
					tokenString = cookie.Value
				}
			}

			if tokenString == "" {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			claims, err := auth.ValidateToken(tokenString)
			if err != nil {
				if errors.Is(err, ErrTokenExpired) {
					http.Error(w, "token expired", http.StatusUnauthorized)
					return
				}
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			// Add claims to context
			ctx := WithClient(r.Context(), claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// extractBearerToken extracts the token from Authorization header
func extractBearerToken(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if len(auth) < 7 {
		return ""
	}
	if auth[:7] != "Bearer " {
		return ""
	}
	return auth[7:]
}

// ============== Response Helpers ==============

// AuthResponse is the response for successful authentication
type AuthResponse struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
	Client       *Client   `json:"client"`
}

// SetAuthCookies sets HTTP-only cookies for portal authentication
func SetAuthCookies(w http.ResponseWriter, accessToken, refreshToken string, accessExpiry, refreshExpiry time.Duration, secure bool) {
	// Access token cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "portal_access_token",
		Value:    accessToken,
		Path:     "/",
		MaxAge:   int(accessExpiry.Seconds()),
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteStrictMode,
	})

	// Refresh token cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "portal_refresh_token",
		Value:    refreshToken,
		Path:     "/api/v1/portal/refresh",
		MaxAge:   int(refreshExpiry.Seconds()),
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteStrictMode,
	})
}

// ClearAuthCookies clears the authentication cookies
func ClearAuthCookies(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     "portal_access_token",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})

	http.SetCookie(w, &http.Cookie{
		Name:     "portal_refresh_token",
		Value:    "",
		Path:     "/api/v1/portal/refresh",
		MaxAge:   -1,
		HttpOnly: true,
	})
}
