package integration

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"austrian-business-infrastructure/internal/api"
	"austrian-business-infrastructure/internal/auth"

	"github.com/google/uuid"
)

// TestAdminRouteAuthorization tests that admin-only routes reject non-admin users
// This prevents privilege escalation vulnerabilities (CWE-269)
func TestAdminRouteAuthorization(t *testing.T) {
	// Skip if no test database available
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup test JWT config with HS256 for testing (ES256 requires key files)
	jwtConfig := &auth.JWTConfig{
		Secret:             "test-secret-key-for-testing-only-32bytes!",
		AccessTokenExpiry:  15 * time.Minute,
		RefreshTokenExpiry: 7 * 24 * time.Hour,
		Issuer:             "test",
		UseES256:           false, // Use HS256 for testing
	}

	jwtManager := auth.NewJWTManager(jwtConfig)

	tenantID := uuid.New()
	adminUserID := uuid.New()
	memberUserID := uuid.New()

	// Generate tokens for different roles
	adminToken, _, err := jwtManager.GenerateAccessToken(&auth.UserInfo{
		UserID:   adminUserID.String(),
		TenantID: tenantID.String(),
		Role:     "admin",
	})
	if err != nil {
		t.Fatalf("Failed to generate admin token: %v", err)
	}

	memberToken, _, err := jwtManager.GenerateAccessToken(&auth.UserInfo{
		UserID:   memberUserID.String(),
		TenantID: tenantID.String(),
		Role:     "member",
	})
	if err != nil {
		t.Fatalf("Failed to generate member token: %v", err)
	}

	viewerToken, _, err := jwtManager.GenerateAccessToken(&auth.UserInfo{
		UserID:   uuid.New().String(),
		TenantID: tenantID.String(),
		Role:     "viewer",
	})
	if err != nil {
		t.Fatalf("Failed to generate viewer token: %v", err)
	}

	// Create test handler with RequireRole middleware
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	// Create middleware chain
	authMiddleware := auth.NewAuthMiddleware(jwtManager)
	protectedHandler := authMiddleware.RequireAuth(authMiddleware.RequireRole("admin")(testHandler))

	testCases := []struct {
		name           string
		token          string
		expectedStatus int
	}{
		{"no_token", "", http.StatusUnauthorized},
		{"member_token", memberToken, http.StatusForbidden},
		{"viewer_token", viewerToken, http.StatusForbidden},
		{"admin_token", adminToken, http.StatusOK},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/v1/admin/test", nil)
			if tc.token != "" {
				req.Header.Set("Authorization", "Bearer "+tc.token)
			}

			rr := httptest.NewRecorder()
			protectedHandler.ServeHTTP(rr, req)

			if rr.Code != tc.expectedStatus {
				t.Errorf("Expected status %d, got %d", tc.expectedStatus, rr.Code)
			}
		})
	}
}

// TestOwnerOnlyRoutes tests that owner-only routes reject non-owner users
func TestOwnerOnlyRoutes(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	jwtConfig := &auth.JWTConfig{
		Secret:             "test-secret-key-for-testing-only-32bytes!",
		AccessTokenExpiry:  15 * time.Minute,
		RefreshTokenExpiry: 7 * 24 * time.Hour,
		Issuer:             "test",
		UseES256:           false,
	}

	jwtManager := auth.NewJWTManager(jwtConfig)
	tenantID := uuid.New()

	adminToken, _, _ := jwtManager.GenerateAccessToken(&auth.UserInfo{
		UserID:   uuid.New().String(),
		TenantID: tenantID.String(),
		Role:     "admin",
	})

	ownerToken, _, _ := jwtManager.GenerateAccessToken(&auth.UserInfo{
		UserID:   uuid.New().String(),
		TenantID: tenantID.String(),
		Role:     "owner",
	})

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	authMiddleware := auth.NewAuthMiddleware(jwtManager)
	protectedHandler := authMiddleware.RequireAuth(authMiddleware.RequireRole("owner")(testHandler))

	testCases := []struct {
		name           string
		token          string
		expectedStatus int
	}{
		{"admin_cannot_access_owner_route", adminToken, http.StatusForbidden},
		{"owner_can_access_owner_route", ownerToken, http.StatusOK},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("DELETE", "/api/v1/tenant", nil)
			req.Header.Set("Authorization", "Bearer "+tc.token)

			rr := httptest.NewRecorder()
			protectedHandler.ServeHTTP(rr, req)

			if rr.Code != tc.expectedStatus {
				t.Errorf("Expected status %d, got %d", tc.expectedStatus, rr.Code)
			}
		})
	}
}

// TestCrossTenantAccessDenied tests that users cannot access other tenants' resources
func TestCrossTenantAccessDenied(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	jwtConfig := &auth.JWTConfig{
		Secret:             "test-secret-key-for-testing-only-32bytes!",
		AccessTokenExpiry:  15 * time.Minute,
		RefreshTokenExpiry: 7 * 24 * time.Hour,
		Issuer:             "test",
		UseES256:           false,
	}

	jwtManager := auth.NewJWTManager(jwtConfig)

	tenantA := uuid.New()
	tenantB := uuid.New()

	tokenA, _, _ := jwtManager.GenerateAccessToken(&auth.UserInfo{
		UserID:   uuid.New().String(),
		TenantID: tenantA.String(),
		Role:     "admin",
	})

	// Handler that checks tenant from context matches requested tenant
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctxTenantID := api.GetTenantID(r.Context())
		requestedTenant := r.URL.Query().Get("tenant_id")

		if requestedTenant != "" && requestedTenant != ctxTenantID {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	authMiddleware := auth.NewAuthMiddleware(jwtManager)
	protectedHandler := authMiddleware.RequireAuth(testHandler)

	testCases := []struct {
		name           string
		token          string
		queryTenantID  string
		expectedStatus int
	}{
		{"own_tenant_allowed", tokenA, tenantA.String(), http.StatusOK},
		{"other_tenant_denied", tokenA, tenantB.String(), http.StatusForbidden},
		{"no_tenant_filter", tokenA, "", http.StatusOK},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			url := "/api/v1/resources"
			if tc.queryTenantID != "" {
				url += "?tenant_id=" + tc.queryTenantID
			}

			req := httptest.NewRequest("GET", url, nil)
			req.Header.Set("Authorization", "Bearer "+tc.token)

			rr := httptest.NewRecorder()
			protectedHandler.ServeHTTP(rr, req)

			if rr.Code != tc.expectedStatus {
				t.Errorf("Expected status %d, got %d", tc.expectedStatus, rr.Code)
			}
		})
	}
}

// TestExpiredTokenRejected verifies expired tokens are rejected
func TestExpiredTokenRejected(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Use very short expiry
	jwtConfig := &auth.JWTConfig{
		Secret:             "test-secret-key-for-testing-only-32bytes!",
		AccessTokenExpiry:  1 * time.Millisecond,
		RefreshTokenExpiry: 7 * 24 * time.Hour,
		Issuer:             "test",
		UseES256:           false,
	}

	jwtManager := auth.NewJWTManager(jwtConfig)

	token, _, _ := jwtManager.GenerateAccessToken(&auth.UserInfo{
		UserID:   uuid.New().String(),
		TenantID: uuid.New().String(),
		Role:     "admin",
	})

	// Wait for token to expire
	time.Sleep(10 * time.Millisecond)

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	authMiddleware := auth.NewAuthMiddleware(jwtManager)
	protectedHandler := authMiddleware.RequireAuth(testHandler)

	req := httptest.NewRequest("GET", "/api/v1/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	rr := httptest.NewRecorder()
	protectedHandler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("Expected 401 for expired token, got %d", rr.Code)
	}
}

// TestInvalidTokenRejected tests various invalid token scenarios
func TestInvalidTokenRejected(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	jwtConfig := &auth.JWTConfig{
		Secret:             "test-secret-key-for-testing-only-32bytes!",
		AccessTokenExpiry:  15 * time.Minute,
		RefreshTokenExpiry: 7 * 24 * time.Hour,
		Issuer:             "test",
		UseES256:           false,
	}

	jwtManager := auth.NewJWTManager(jwtConfig)

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	authMiddleware := auth.NewAuthMiddleware(jwtManager)
	protectedHandler := authMiddleware.RequireAuth(testHandler)

	// Token signed with different secret
	wrongSecretConfig := &auth.JWTConfig{
		Secret:             "different-secret-key-32-bytes-long!!",
		AccessTokenExpiry:  15 * time.Minute,
		RefreshTokenExpiry: 7 * 24 * time.Hour,
		Issuer:             "test",
		UseES256:           false,
	}
	wrongSecretManager := auth.NewJWTManager(wrongSecretConfig)
	wrongSecretToken, _, _ := wrongSecretManager.GenerateAccessToken(&auth.UserInfo{
		UserID:   uuid.New().String(),
		TenantID: uuid.New().String(),
		Role:     "admin",
	})

	testCases := []struct {
		name  string
		token string
	}{
		{"empty_token", ""},
		{"not_a_jwt", "not-a-jwt-token"},
		{"wrong_secret", wrongSecretToken},
		{"missing_bearer", "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.e30.test"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/v1/test", nil)
			if tc.token != "" {
				if tc.name == "missing_bearer" {
					req.Header.Set("Authorization", tc.token)
				} else {
					req.Header.Set("Authorization", "Bearer "+tc.token)
				}
			}

			rr := httptest.NewRecorder()
			protectedHandler.ServeHTTP(rr, req)

			if rr.Code != http.StatusUnauthorized {
				t.Errorf("Expected 401 for %s, got %d", tc.name, rr.Code)
			}
		})
	}
}
