package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/austrian-business-infrastructure/fo/internal/account"
	"github.com/austrian-business-infrastructure/fo/internal/api"
	"github.com/austrian-business-infrastructure/fo/internal/auth"
	"github.com/austrian-business-infrastructure/fo/internal/config"
	"github.com/austrian-business-infrastructure/fo/internal/firmenbuch"
	"github.com/austrian-business-infrastructure/fo/internal/invoice"
	"github.com/austrian-business-infrastructure/fo/internal/payment"
	"github.com/austrian-business-infrastructure/fo/internal/tenant"
	"github.com/austrian-business-infrastructure/fo/internal/uid"
	"github.com/austrian-business-infrastructure/fo/internal/user"
	"github.com/austrian-business-infrastructure/fo/internal/uva"
	"github.com/austrian-business-infrastructure/fo/internal/zm"
	"github.com/austrian-business-infrastructure/fo/pkg/cache"
	"github.com/austrian-business-infrastructure/fo/pkg/database"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	// Setup structured logging
	logLevel := slog.LevelInfo
	if os.Getenv("LOG_LEVEL") == "debug" {
		logLevel = slog.LevelDebug
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	}))
	slog.SetDefault(logger)

	logger.Info("starting server")

	// Load configuration
	cfg, err := config.LoadServerConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Load JWT signing keys - fail fast if not configured in production
	isDev := os.Getenv("APP_ENV") == "dev" || os.Getenv("APP_ENV") == "development"
	if err := auth.MustLoadKeys(isDev); err != nil {
		return fmt.Errorf("failed to load JWT keys: %w", err)
	}
	logger.Info("JWT signing keys loaded")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize database connection
	dbConfig := database.DefaultPostgresConfig(cfg.DatabaseURL)
	db, err := database.NewPool(ctx, dbConfig)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer db.Close()
	logger.Info("connected to database")

	// Initialize Redis connection
	redisConfig := cache.DefaultRedisConfig(cfg.RedisURL)
	redis, err := cache.NewClient(ctx, redisConfig)
	if err != nil {
		return fmt.Errorf("failed to connect to redis: %w", err)
	}
	defer redis.Close()
	logger.Info("connected to redis")

	// Setup router
	router := api.NewRouter(logger)

	// Add global middlewares
	router.Use(api.RequestID)
	router.Use(api.Recovery(logger))
	router.Use(api.Logger(logger))
	router.Use(api.CORS(cfg.AllowedOrigins))
	router.Use(api.SecureHeaders)
	router.Use(api.ContentSecurityPolicy(api.DefaultCSPConfig()))

	// Health check endpoints
	router.HandleFunc("GET /health", healthHandler())
	router.HandleFunc("GET /ready", readyHandler(db, redis))

	// Initialize repositories (use db.Pool to get underlying *pgxpool.Pool)
	tenantRepo := tenant.NewRepository(db.Pool)
	userRepo := user.NewRepository(db.Pool)
	accountRepo := account.NewRepository(db.Pool)
	uvaRepo := uva.NewRepository(db.Pool)
	zmRepo := zm.NewRepository(db.Pool)
	invoiceRepo := invoice.NewRepository(db.Pool)
	paymentRepo := payment.NewRepository(db.Pool)
	firmenbuchRepo := firmenbuch.NewRepository(db.Pool)
	uidRepo := uid.NewRepository(db.Pool)

	// Initialize services
	userService := user.NewService(userRepo)
	tenantService := tenant.NewService(db.Pool, tenantRepo, userRepo)

	accountService, err := account.NewService(accountRepo, []byte(cfg.EncryptionKey))
	if err != nil {
		return fmt.Errorf("failed to create account service: %w", err)
	}

	uvaService := uva.NewService(uvaRepo, accountService)
	zmService := zm.NewService(zmRepo, accountService)
	invoiceService := invoice.NewService(invoiceRepo)
	paymentService := payment.NewService(paymentRepo)
	firmenbuchService := firmenbuch.NewService(firmenbuchRepo, nil) // client nil for now
	uidService := uid.NewService(uidRepo, accountService)

	// Initialize JWT manager with revocation support
	jwtConfig := auth.DefaultJWTConfig(cfg.JWTSecret)
	jwtManager := auth.NewJWTManager(jwtConfig)
	revocationList := auth.NewTokenRevocationList(redis.Client) // redis.Client is embedded *redis.Client
	jwtManager.SetRevocationList(revocationList)

	// Initialize session manager (needs pgxpool.Pool, cache.Client, TTL)
	sessionManager := auth.NewSessionManager(db.Pool, redis, 7*24*time.Hour)

	// Initialize handlers
	authHandler := auth.NewHandler(tenantService, userService, sessionManager, jwtManager, logger)
	accountHandler := account.NewHandler(accountService)
	uvaHandler := uva.NewHandler(uvaService)
	zmHandler := zm.NewHandler(zmService)
	invoiceHandler := invoice.NewHandler(invoiceService)
	paymentHandler := payment.NewHandler(paymentService)
	firmenbuchHandler := firmenbuch.NewHandler(firmenbuchService)
	uidHandler := uid.NewHandler(uidService)

	// Auth middleware
	authMiddleware := auth.NewAuthMiddleware(jwtManager)
	requireAuth := authMiddleware.RequireAuth
	requireAdmin := authMiddleware.RequireRole("admin")

	// Register routes
	// Auth routes (no auth required for login/register)
	authHandler.RegisterRoutes(router)

	// Protected routes
	accountHandler.RegisterRoutes(router, requireAuth, requireAdmin)
	uvaHandler.RegisterRoutes(router, requireAuth, requireAdmin)
	zmHandler.RegisterRoutes(router, requireAuth, requireAdmin)
	invoiceHandler.RegisterRoutes(router, requireAuth, requireAdmin)
	paymentHandler.RegisterRoutes(router, requireAuth, requireAdmin)
	firmenbuchHandler.RegisterRoutes(router, requireAuth, requireAdmin)
	uidHandler.RegisterRoutes(router, requireAuth, requireAdmin)

	logger.Info("API routes registered")

	// Create HTTP server
	server := &http.Server{
		Addr:         cfg.Address(),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	serverErrors := make(chan error, 1)
	go func() {
		logger.Info("server listening", "address", cfg.Address())
		serverErrors <- server.ListenAndServe()
	}()

	// Wait for shutdown signal
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-serverErrors:
		return fmt.Errorf("server error: %w", err)

	case sig := <-shutdown:
		logger.Info("shutdown signal received", "signal", sig)

		// Create deadline for shutdown
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// Attempt graceful shutdown
		if err := server.Shutdown(ctx); err != nil {
			// Force close if graceful shutdown fails
			logger.Error("graceful shutdown failed, forcing close", "error", err)
			if err := server.Close(); err != nil {
				return fmt.Errorf("could not close server: %w", err)
			}
		}

		logger.Info("server stopped gracefully")
	}

	return nil
}

// healthHandler returns liveness probe handler
func healthHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		api.JSONResponse(w, http.StatusOK, map[string]string{
			"status": "ok",
		})
	}
}

// readyHandler returns readiness probe handler
func readyHandler(db *database.Pool, redis *cache.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		checks := make(map[string]string)
		healthy := true

		// Check database
		if err := db.Health(ctx); err != nil {
			checks["database"] = "unhealthy: " + err.Error()
			healthy = false
		} else {
			checks["database"] = "healthy"
		}

		// Check Redis
		if err := redis.Health(ctx); err != nil {
			checks["redis"] = "unhealthy: " + err.Error()
			healthy = false
		} else {
			checks["redis"] = "healthy"
		}

		status := http.StatusOK
		if !healthy {
			status = http.StatusServiceUnavailable
		}

		api.JSONResponse(w, status, map[string]interface{}{
			"status": map[bool]string{true: "ready", false: "not_ready"}[healthy],
			"checks": checks,
		})
	}
}
