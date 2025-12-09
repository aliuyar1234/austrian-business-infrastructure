package platform

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/austrian-business-infrastructure/fo/internal/api"
	"github.com/austrian-business-infrastructure/fo/internal/auth"
	"github.com/austrian-business-infrastructure/fo/internal/tenant"
	"github.com/austrian-business-infrastructure/fo/internal/user"
	"github.com/austrian-business-infrastructure/fo/pkg/cache"
)

var testLogger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

// T072: Integration tests for registration and login flow

func TestAuthRegistrationFlow(t *testing.T) {
	env := Setup(t)
	if env == nil {
		return
	}
	defer env.Cleanup()

	ctx := context.Background()
	if err := env.Reset(ctx); err != nil {
		t.Fatalf("Failed to reset environment: %v", err)
	}

	// Setup services
	tenantRepo := tenant.NewRepository(env.DB)
	userRepo := user.NewRepository(env.DB)
	tenantSvc := tenant.NewService(tenantRepo, userRepo)
	userSvc := user.NewService(userRepo)

	redisClient := &cache.Client{} // Would need proper setup
	jwtCfg := &auth.JWTConfig{
		Secret:             "test-secret-key-minimum-32-chars!",
		AccessTokenExpiry:  15 * time.Minute,
		RefreshTokenExpiry: 7 * 24 * time.Hour,
		Issuer:             "test",
	}
	jwtMgr := auth.NewJWTManager(jwtCfg)
	sessionSvc := auth.NewSessionService(redisClient, jwtMgr, 7*24*time.Hour)

	handler := auth.NewHandler(tenantSvc, userSvc, sessionSvc, jwtMgr, nil, nil)

	router := api.NewRouter(testLogger)
	handler.RegisterRoutes(router)

	client := NewTestClient(t, router)

	t.Run("Register new tenant and owner", func(t *testing.T) {
		resp := client.Post("/api/v1/auth/register", map[string]interface{}{
			"email":       "owner@test.com",
			"password":    "securepassword123",
			"tenant_name": "Test Company",
		})

		AssertStatus(t, resp, http.StatusCreated)
		AssertJSON(t, resp, "access_token")
		AssertJSON(t, resp, "refresh_token")
		AssertJSON(t, resp, "user")
	})

	t.Run("Register with existing email fails", func(t *testing.T) {
		resp := client.Post("/api/v1/auth/register", map[string]interface{}{
			"email":       "owner@test.com",
			"password":    "securepassword123",
			"tenant_name": "Another Company",
		})

		AssertStatus(t, resp, http.StatusConflict)
	})

	t.Run("Register with weak password fails", func(t *testing.T) {
		resp := client.Post("/api/v1/auth/register", map[string]interface{}{
			"email":       "new@test.com",
			"password":    "short",
			"tenant_name": "New Company",
		})

		AssertStatus(t, resp, http.StatusBadRequest)
	})

	t.Run("Register with invalid email fails", func(t *testing.T) {
		resp := client.Post("/api/v1/auth/register", map[string]interface{}{
			"email":       "not-an-email",
			"password":    "securepassword123",
			"tenant_name": "New Company",
		})

		AssertStatus(t, resp, http.StatusBadRequest)
	})
}

func TestAuthLoginFlow(t *testing.T) {
	env := Setup(t)
	if env == nil {
		return
	}
	defer env.Cleanup()

	ctx := context.Background()
	if err := env.Reset(ctx); err != nil {
		t.Fatalf("Failed to reset environment: %v", err)
	}

	// Setup services
	tenantRepo := tenant.NewRepository(env.DB)
	userRepo := user.NewRepository(env.DB)
	tenantSvc := tenant.NewService(tenantRepo, userRepo)
	userSvc := user.NewService(userRepo)

	redisClient := &cache.Client{}
	jwtCfg := &auth.JWTConfig{
		Secret:             "test-secret-key-minimum-32-chars!",
		AccessTokenExpiry:  15 * time.Minute,
		RefreshTokenExpiry: 7 * 24 * time.Hour,
		Issuer:             "test",
	}
	jwtMgr := auth.NewJWTManager(jwtCfg)
	sessionSvc := auth.NewSessionService(redisClient, jwtMgr, 7*24*time.Hour)

	handler := auth.NewHandler(tenantSvc, userSvc, sessionSvc, jwtMgr, nil, nil)

	router := api.NewRouter(testLogger)
	handler.RegisterRoutes(router)

	client := NewTestClient(t, router)

	// First register a user
	resp := client.Post("/api/v1/auth/register", map[string]interface{}{
		"email":       "login@test.com",
		"password":    "securepassword123",
		"tenant_name": "Login Test Company",
	})
	AssertStatus(t, resp, http.StatusCreated)

	t.Run("Login with valid credentials", func(t *testing.T) {
		resp := client.Post("/api/v1/auth/login", map[string]interface{}{
			"email":    "login@test.com",
			"password": "securepassword123",
		})

		AssertStatus(t, resp, http.StatusOK)
		AssertJSON(t, resp, "access_token")
		AssertJSON(t, resp, "refresh_token")
	})

	t.Run("Login with wrong password fails", func(t *testing.T) {
		resp := client.Post("/api/v1/auth/login", map[string]interface{}{
			"email":    "login@test.com",
			"password": "wrongpassword123",
		})

		AssertStatus(t, resp, http.StatusUnauthorized)
	})

	t.Run("Login with non-existent user fails", func(t *testing.T) {
		resp := client.Post("/api/v1/auth/login", map[string]interface{}{
			"email":    "nonexistent@test.com",
			"password": "securepassword123",
		})

		AssertStatus(t, resp, http.StatusUnauthorized)
	})
}

func TestAuthTokenRefresh(t *testing.T) {
	env := Setup(t)
	if env == nil {
		return
	}
	defer env.Cleanup()

	ctx := context.Background()
	if err := env.Reset(ctx); err != nil {
		t.Fatalf("Failed to reset environment: %v", err)
	}

	// Setup services
	tenantRepo := tenant.NewRepository(env.DB)
	userRepo := user.NewRepository(env.DB)
	tenantSvc := tenant.NewService(tenantRepo, userRepo)
	userSvc := user.NewService(userRepo)

	redisClient := &cache.Client{}
	jwtCfg := &auth.JWTConfig{
		Secret:             "test-secret-key-minimum-32-chars!",
		AccessTokenExpiry:  15 * time.Minute,
		RefreshTokenExpiry: 7 * 24 * time.Hour,
		Issuer:             "test",
	}
	jwtMgr := auth.NewJWTManager(jwtCfg)
	sessionSvc := auth.NewSessionService(redisClient, jwtMgr, 7*24*time.Hour)

	handler := auth.NewHandler(tenantSvc, userSvc, sessionSvc, jwtMgr, nil, nil)

	router := api.NewRouter(testLogger)
	handler.RegisterRoutes(router)

	client := NewTestClient(t, router)

	// Register and get tokens
	resp := client.Post("/api/v1/auth/register", map[string]interface{}{
		"email":       "refresh@test.com",
		"password":    "securepassword123",
		"tenant_name": "Refresh Test Company",
	})
	AssertStatus(t, resp, http.StatusCreated)

	var authResp struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
	}
	ParseResponse(t, resp, &authResp)

	t.Run("Refresh with valid token", func(t *testing.T) {
		resp := client.Post("/api/v1/auth/refresh", map[string]interface{}{
			"refresh_token": authResp.RefreshToken,
		})

		AssertStatus(t, resp, http.StatusOK)
		AssertJSON(t, resp, "access_token")
		AssertJSON(t, resp, "refresh_token")
	})

	t.Run("Refresh with invalid token fails", func(t *testing.T) {
		resp := client.Post("/api/v1/auth/refresh", map[string]interface{}{
			"refresh_token": "invalid-token",
		})

		AssertStatus(t, resp, http.StatusUnauthorized)
	})
}

func TestAuthLogout(t *testing.T) {
	env := Setup(t)
	if env == nil {
		return
	}
	defer env.Cleanup()

	ctx := context.Background()
	if err := env.Reset(ctx); err != nil {
		t.Fatalf("Failed to reset environment: %v", err)
	}

	// Setup services
	tenantRepo := tenant.NewRepository(env.DB)
	userRepo := user.NewRepository(env.DB)
	tenantSvc := tenant.NewService(tenantRepo, userRepo)
	userSvc := user.NewService(userRepo)

	redisClient := &cache.Client{}
	jwtCfg := &auth.JWTConfig{
		Secret:             "test-secret-key-minimum-32-chars!",
		AccessTokenExpiry:  15 * time.Minute,
		RefreshTokenExpiry: 7 * 24 * time.Hour,
		Issuer:             "test",
	}
	jwtMgr := auth.NewJWTManager(jwtCfg)
	sessionSvc := auth.NewSessionService(redisClient, jwtMgr, 7*24*time.Hour)

	authMiddleware := auth.NewAuthMiddleware(jwtMgr, userSvc)
	handler := auth.NewHandler(tenantSvc, userSvc, sessionSvc, jwtMgr, nil, nil)

	router := api.NewRouter(testLogger)
	handler.RegisterRoutes(router)
	router.Handle("POST /api/v1/auth/logout", authMiddleware.RequireAuth(http.HandlerFunc(handler.Logout)))

	client := NewTestClient(t, router)

	// Register and get tokens
	resp := client.Post("/api/v1/auth/register", map[string]interface{}{
		"email":       "logout@test.com",
		"password":    "securepassword123",
		"tenant_name": "Logout Test Company",
	})
	AssertStatus(t, resp, http.StatusCreated)

	var authResp struct {
		AccessToken string `json:"access_token"`
	}
	ParseResponse(t, resp, &authResp)

	t.Run("Logout with valid token", func(t *testing.T) {
		client.SetToken(authResp.AccessToken)
		resp := client.Post("/api/v1/auth/logout", nil)

		AssertStatus(t, resp, http.StatusNoContent)
	})

	t.Run("Logout without token fails", func(t *testing.T) {
		client.SetToken("")
		resp := client.Post("/api/v1/auth/logout", nil)

		AssertStatus(t, resp, http.StatusUnauthorized)
	})
}
