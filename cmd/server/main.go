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

	"github.com/go-chi/chi/v5"

	"austrian-business-infrastructure/internal/account"
	"austrian-business-infrastructure/internal/antrag"
	"austrian-business-infrastructure/internal/api"
	"austrian-business-infrastructure/internal/apikey"
	"austrian-business-infrastructure/internal/audit"
	"austrian-business-infrastructure/internal/auth"
	"austrian-business-infrastructure/internal/config"
	"austrian-business-infrastructure/internal/document"
	"austrian-business-infrastructure/internal/firmenbuch"
	"austrian-business-infrastructure/internal/foerderung"
	"austrian-business-infrastructure/internal/invoice"
	"austrian-business-infrastructure/internal/matcher"
	"austrian-business-infrastructure/internal/monitor"
	"austrian-business-infrastructure/internal/notification"
	"austrian-business-infrastructure/internal/payment"
	"austrian-business-infrastructure/internal/profil"
	"austrian-business-infrastructure/internal/session"
	"austrian-business-infrastructure/internal/tenant"
	"austrian-business-infrastructure/internal/uid"
	"austrian-business-infrastructure/internal/user"
	"austrian-business-infrastructure/internal/uva"
	"austrian-business-infrastructure/internal/webhook"
	"austrian-business-infrastructure/internal/zm"
	"austrian-business-infrastructure/pkg/cache"
	"austrian-business-infrastructure/pkg/database"
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

	// Förderung-related repositories
	foerderungRepo := foerderung.NewRepository(db.Pool)
	antragRepo := antrag.NewRepository(db.Pool)
	profilRepo := profil.NewRepository(db.Pool)
	monitorRepo := monitor.NewRepository(db.Pool)
	monitorNotifRepo := monitor.NewNotificationRepository(db.Pool)
	matcherSearchRepo := matcher.NewSearchRepository(db.Pool)

	// Additional repositories for new handlers
	auditRepo := audit.NewRepository(db.Pool)
	notificationRepo := notification.NewRepository(db.Pool)
	apikeyRepo := apikey.NewRepository(db.Pool)

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

	// Förderung-related services
	antragService := antrag.NewService(antragRepo)
	profilService := profil.NewService(profilRepo)
	monitorService := monitor.NewService(monitorRepo, monitorNotifRepo)
	matcherService := matcher.NewService(foerderungRepo, matcherSearchRepo, nil, nil) // nil LLM client for now

	// Additional services for new handlers (apikey only, notification needs docRepo)
	apikeyService := apikey.NewService(apikeyRepo)

	// Initialize document storage and service with IDOR protection
	docStorage, err := document.NewStorage(&document.StorageConfig{
		Type:              document.StorageType(cfg.StorageType),
		LocalPath:         cfg.StorageLocalPath,
		S3Endpoint:        cfg.StorageS3Endpoint,
		S3Bucket:          cfg.StorageS3Bucket,
		S3Region:          cfg.StorageS3Region,
		S3AccessKeyID:     cfg.StorageS3AccessKeyID,
		S3SecretAccessKey: cfg.StorageS3SecretKey,
		S3UseSSL:          cfg.StorageS3UseSSL,
	})
	if err != nil {
		return fmt.Errorf("failed to create document storage: %w", err)
	}

	docRepo := document.NewRepository(db.Pool)
	// CRITICAL: Use NewServiceWithAccountVerifier to enable tenant isolation on document creation
	// This prevents IDOR attacks where attackers could create documents for accounts they don't own
	docService := document.NewServiceWithAccountVerifier(docRepo, docStorage, accountRepo)

	// Initialize notification service (needs docRepo to be initialized first)
	notificationService := notification.NewService(notificationRepo, docRepo, nil, &notification.ServiceConfig{
		Logger: logger,
		AppURL: "http://localhost:3000", // TODO: Get from config
	})

	// Initialize webhook repository and service
	webhookRepo := webhook.NewRepository(db.Pool)
	webhookService := webhook.NewService(webhookRepo, &webhook.ServiceConfig{
		Logger: logger,
	})

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
	docHandler := document.NewHandler(docService)

	// Förderung-related handlers
	foerderungHandler := foerderung.NewHandler(foerderungRepo)
	antragHandler := antrag.NewHandler(antragService)
	profilHandler := profil.NewHandler(profilService, nil) // nil deriveService for now
	monitorHandler := monitor.NewHandler(monitorService)
	matcherHandler := matcher.NewHandler(matcherService, profilRepo)

	// Additional handlers for user management, sessions, audit, notifications, API keys, webhooks
	userHandler := user.NewHandler(userService, logger)
	sessionHandler := session.NewHandler(sessionManager, logger)
	auditHandler := audit.NewHandler(auditRepo, logger)
	notificationHandler := notification.NewHandler(notificationService)
	apikeyHandler := apikey.NewHandler(apikeyService, logger)
	webhookHandler := webhook.NewHandler(webhookRepo, webhookService)

	// Auth middleware
	authMiddleware := auth.NewAuthMiddleware(jwtManager)
	requireAuth := authMiddleware.RequireAuth
	requireAdmin := authMiddleware.RequireRole("admin")

	// Register routes
	// Auth routes (no auth required for login/register)
	authHandler.RegisterRoutes(router, requireAuth)

	// Protected routes
	accountHandler.RegisterRoutes(router, requireAuth, requireAdmin)
	uvaHandler.RegisterRoutes(router, requireAuth, requireAdmin)
	zmHandler.RegisterRoutes(router, requireAuth, requireAdmin)
	invoiceHandler.RegisterRoutes(router, requireAuth, requireAdmin)
	paymentHandler.RegisterRoutes(router, requireAuth, requireAdmin)
	firmenbuchHandler.RegisterRoutes(router, requireAuth, requireAdmin)
	uidHandler.RegisterRoutes(router, requireAuth, requireAdmin)

	// User management routes (admin-only for modifications)
	userHandler.RegisterRoutes(router, requireAuth, requireAdmin)

	// Session management routes (users can manage their own sessions)
	sessionHandler.RegisterRoutes(router, requireAuth)

	// Audit log routes (admin-only)
	auditHandler.RegisterRoutes(router, requireAuth, requireAdmin)

	// 2FA setup routes (authenticated users)
	authHandler.Register2FARoutes(router, requireAuth)

	// API key management routes (authenticated users)
	apikeyHandler.RegisterRoutes(router, requireAuth)

	// Notification preferences routes (wrap with auth middleware)
	notifMux := http.NewServeMux()
	notificationHandler.RegisterRoutes(notifMux)
	router.Handle("/api/v1/notifications/preferences", requireAuth(notifMux))
	router.Handle("/api/v1/notifications/preferences/", requireAuth(notifMux))

	// Webhook routes (wrap with auth middleware, admin-only for create/update/delete)
	webhookMux := http.NewServeMux()
	webhookHandler.RegisterRoutes(webhookMux)
	router.Handle("/api/v1/webhooks", requireAuth(webhookMux))
	router.Handle("/api/v1/webhooks/", requireAuth(webhookMux))

	// Document routes (protected by auth middleware)
	// Wrap document routes with auth middleware since RegisterRoutes uses raw mux
	docMux := http.NewServeMux()
	docHandler.RegisterRoutes(docMux)
	router.Handle("/api/v1/documents", requireAuth(docMux))
	router.Handle("/api/v1/documents/", requireAuth(docMux))

	// Förderung-related routes using chi router (these handlers use chi.URLParam)
	chiRouter := chi.NewRouter()
	foerderungHandler.RegisterRoutes(chiRouter)
	antragHandler.RegisterRoutes(chiRouter)
	profilHandler.RegisterRoutes(chiRouter)
	monitorHandler.RegisterRoutes(chiRouter)
	matcherHandler.RegisterRoutes(chiRouter)

	// Mount chi router at /api/v1 (wrap with auth middleware)
	router.Handle("/api/v1/foerderungen", requireAuth(chiRouter))
	router.Handle("/api/v1/foerderungen/", requireAuth(chiRouter))
	router.Handle("/api/v1/antraege", requireAuth(chiRouter))
	router.Handle("/api/v1/antraege/", requireAuth(chiRouter))
	router.Handle("/api/v1/profile", requireAuth(chiRouter))
	router.Handle("/api/v1/profile/", requireAuth(chiRouter))
	router.Handle("/api/v1/monitor", requireAuth(chiRouter))
	router.Handle("/api/v1/monitor/", requireAuth(chiRouter))
	router.Handle("/api/v1/foerderungssuche", requireAuth(chiRouter))
	router.Handle("/api/v1/foerderungssuche/", requireAuth(chiRouter))

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

		// Check database - don't leak error details to unauthenticated callers
		if err := db.Health(ctx); err != nil {
			checks["database"] = "unhealthy"
			healthy = false
		} else {
			checks["database"] = "healthy"
		}

		// Check Redis - don't leak error details to unauthenticated callers
		if err := redis.Health(ctx); err != nil {
			checks["redis"] = "unhealthy"
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
