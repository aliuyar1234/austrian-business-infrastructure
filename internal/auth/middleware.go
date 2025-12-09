package auth

import (
	"context"
	"net/http"
	"strings"

	"github.com/austrian-business-infrastructure/fo/internal/api"
	"github.com/austrian-business-infrastructure/fo/internal/security"
	"github.com/google/uuid"
)

// AuthMiddleware provides JWT authentication middleware
type AuthMiddleware struct {
	jwtManager *JWTManager
}

// NewAuthMiddleware creates a new auth middleware
func NewAuthMiddleware(jwtManager *JWTManager) *AuthMiddleware {
	return &AuthMiddleware{jwtManager: jwtManager}
}

// RequireAuth returns middleware that requires a valid JWT token
func (m *AuthMiddleware) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract token from Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			api.JSONError(w, http.StatusUnauthorized, "Authorization header required", api.ErrCodeUnauthorized)
			return
		}

		// Check Bearer prefix
		if !strings.HasPrefix(authHeader, "Bearer ") {
			api.JSONError(w, http.StatusUnauthorized, "Invalid authorization format", api.ErrCodeUnauthorized)
			return
		}

		token := authHeader[7:] // Remove "Bearer " prefix

		// Validate token
		claims, err := m.jwtManager.ValidateAccessToken(token)
		if err != nil {
			switch err {
			case ErrExpiredToken:
				api.JSONError(w, http.StatusUnauthorized, "Token has expired", api.ErrCodeTokenExpired)
			case ErrInvalidToken, ErrInvalidClaims:
				api.JSONError(w, http.StatusUnauthorized, "Invalid token", api.ErrCodeInvalidToken)
			default:
				api.JSONError(w, http.StatusUnauthorized, "Authentication failed", api.ErrCodeUnauthorized)
			}
			return
		}

		// Inject user info into context (Email intentionally NOT included per FR-104)
		ctx := r.Context()
		ctx = context.WithValue(ctx, api.UserIDKey, claims.UserID)
		ctx = context.WithValue(ctx, api.TenantIDKey, claims.TenantID)
		ctx = context.WithValue(ctx, api.UserRoleKey, claims.Role)
		// Note: Email is NOT stored in JWT claims per FR-104 - no PII in tokens

		// Also set RLS tenant context for Row-Level Security (FR-113)
		tenantUUID, err := uuid.Parse(claims.TenantID)
		if err == nil {
			userUUID, _ := uuid.Parse(claims.UserID)
			ctx = security.WithTenantContext(ctx, tenantUUID, userUUID)
		}

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// OptionalAuth returns middleware that validates JWT if present but doesn't require it
func (m *AuthMiddleware) OptionalAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			next.ServeHTTP(w, r)
			return
		}

		if !strings.HasPrefix(authHeader, "Bearer ") {
			next.ServeHTTP(w, r)
			return
		}

		token := authHeader[7:]
		claims, err := m.jwtManager.ValidateAccessToken(token)
		if err != nil {
			// Invalid token - continue without auth
			next.ServeHTTP(w, r)
			return
		}

		// Inject user info into context (Email intentionally NOT included per FR-104)
		ctx := r.Context()
		ctx = context.WithValue(ctx, api.UserIDKey, claims.UserID)
		ctx = context.WithValue(ctx, api.TenantIDKey, claims.TenantID)
		ctx = context.WithValue(ctx, api.UserRoleKey, claims.Role)
		// Note: Email is NOT stored in JWT claims per FR-104 - no PII in tokens

		// Also set RLS tenant context for Row-Level Security (FR-113)
		tenantUUID, parseErr := uuid.Parse(claims.TenantID)
		if parseErr == nil {
			userUUID, _ := uuid.Parse(claims.UserID)
			ctx = security.WithTenantContext(ctx, tenantUUID, userUUID)
		}

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireRole returns middleware that requires a specific role or higher
func (m *AuthMiddleware) RequireRole(minRole string) api.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userRole := api.GetUserRole(r.Context())
			if userRole == "" {
				api.JSONError(w, http.StatusUnauthorized, "Authentication required", api.ErrCodeUnauthorized)
				return
			}

			if !hasMinimumRole(userRole, minRole) {
				api.JSONError(w, http.StatusForbidden, "Insufficient permissions", api.ErrCodeForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireTenant returns middleware that requires matching tenant
func (m *AuthMiddleware) RequireTenant(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get tenant from path parameter (if present)
		pathTenant := r.PathValue("tenant_id")
		if pathTenant == "" {
			next.ServeHTTP(w, r)
			return
		}

		// Check if user's tenant matches
		userTenant := api.GetTenantID(r.Context())
		if userTenant != pathTenant {
			api.JSONError(w, http.StatusForbidden, "Access denied to this tenant", api.ErrCodeForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// Role hierarchy levels (higher number = more permissions)
var roleHierarchy = map[string]int{
	"viewer": 1,
	"member": 2,
	"admin":  3,
	"owner":  4,
}

// hasMinimumRole checks if userRole meets or exceeds minRole
func hasMinimumRole(userRole, minRole string) bool {
	userLevel, ok := roleHierarchy[userRole]
	if !ok {
		return false
	}

	minLevel, ok := roleHierarchy[minRole]
	if !ok {
		return false
	}

	return userLevel >= minLevel
}

// IsOwner checks if the current user is an owner
func IsOwner(ctx context.Context) bool {
	return api.GetUserRole(ctx) == "owner"
}

// IsAdmin checks if the current user is an admin or higher
func IsAdmin(ctx context.Context) bool {
	return hasMinimumRole(api.GetUserRole(ctx), "admin")
}

// IsMember checks if the current user is a member or higher
func IsMember(ctx context.Context) bool {
	return hasMinimumRole(api.GetUserRole(ctx), "member")
}

// Require2FA returns middleware that blocks access if user has not enabled 2FA
// This enforces FR-109: 2FA must be enabled before accessing protected resources
func (m *AuthMiddleware) Require2FA(getUserByID func(ctx context.Context, userID string) (totpEnabled bool, err error)) api.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID := api.GetUserID(r.Context())
			if userID == "" {
				api.JSONError(w, http.StatusUnauthorized, "Authentication required", api.ErrCodeUnauthorized)
				return
			}

			// Check if user has 2FA enabled
			totpEnabled, err := getUserByID(r.Context(), userID)
			if err != nil {
				// On error, fail closed - require 2FA
				api.JSONError(w, http.StatusForbidden, "2FA verification required", "2FA_REQUIRED")
				return
			}

			if !totpEnabled {
				api.JSONError(w, http.StatusForbidden, "2FA must be enabled to access this resource. Please enable 2FA in your account settings.", "2FA_REQUIRED")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
