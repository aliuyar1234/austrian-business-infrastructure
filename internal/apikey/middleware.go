package apikey

import (
	"context"
	"net/http"
	"strings"

	"austrian-business-infrastructure/internal/api"
)

// Middleware provides API key authentication middleware
type Middleware struct {
	service *Service
}

// NewMiddleware creates a new API key middleware
func NewMiddleware(service *Service) *Middleware {
	return &Middleware{service: service}
}

// AuthenticateAPIKey returns middleware that authenticates via X-API-Key header
func (m *Middleware) AuthenticateAPIKey(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiKey := r.Header.Get("X-API-Key")
		if apiKey == "" {
			// No API key - continue to next handler (might have JWT)
			next.ServeHTTP(w, r)
			return
		}

		// Validate API key
		key, err := m.service.Validate(r.Context(), apiKey)
		if err != nil {
			switch err {
			case ErrAPIKeyNotFound:
				api.JSONError(w, http.StatusUnauthorized, "Invalid API key", api.ErrCodeInvalidToken)
			case ErrAPIKeyExpired:
				api.JSONError(w, http.StatusUnauthorized, "API key has expired", api.ErrCodeTokenExpired)
			case ErrAPIKeyInactive:
				api.JSONError(w, http.StatusUnauthorized, "API key is inactive", api.ErrCodeInvalidToken)
			default:
				api.JSONError(w, http.StatusUnauthorized, "Authentication failed", api.ErrCodeUnauthorized)
			}
			return
		}

		// Inject API key info into context
		ctx := r.Context()
		ctx = context.WithValue(ctx, api.UserIDKey, key.UserID.String())
		ctx = context.WithValue(ctx, api.TenantIDKey, key.TenantID.String())
		ctx = context.WithValue(ctx, apiKeyContextKey, key)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireScope returns middleware that requires a specific scope
func (m *Middleware) RequireScope(scope string) api.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := GetAPIKey(r.Context())
			if key == nil {
				// Not using API key auth - allow JWT auth to handle
				next.ServeHTTP(w, r)
				return
			}

			if !HasScope(key, scope) {
				api.JSONError(w, http.StatusForbidden, "Insufficient scope", api.ErrCodeForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// Context key for API key
type contextKeyType string

const apiKeyContextKey contextKeyType = "api_key"

// GetAPIKey retrieves the API key from context
func GetAPIKey(ctx context.Context) *APIKey {
	if key, ok := ctx.Value(apiKeyContextKey).(*APIKey); ok {
		return key
	}
	return nil
}

// IsAPIKeyAuth returns true if the request was authenticated via API key
func IsAPIKeyAuth(ctx context.Context) bool {
	return GetAPIKey(ctx) != nil
}

// CombinedAuth returns middleware that accepts either JWT or API key authentication
func CombinedAuth(jwtAuth, apiKeyAuth func(http.Handler) http.Handler) api.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check if API key is present
			if r.Header.Get("X-API-Key") != "" {
				apiKeyAuth(next).ServeHTTP(w, r)
				return
			}

			// Check if Authorization header is present
			if authHeader := r.Header.Get("Authorization"); authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
				jwtAuth(next).ServeHTTP(w, r)
				return
			}

			// No authentication provided
			api.JSONError(w, http.StatusUnauthorized, "Authentication required", api.ErrCodeUnauthorized)
		})
	}
}
