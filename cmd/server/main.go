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

	"github.com/austrian-business-infrastructure/fo/internal/api"
	"github.com/austrian-business-infrastructure/fo/internal/auth"
	"github.com/austrian-business-infrastructure/fo/internal/config"
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

	// API v1 routes
	apiV1 := router.Group("/api/v1")
	_ = apiV1 // Will be used when registering handlers

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
