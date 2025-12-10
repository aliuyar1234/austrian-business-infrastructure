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

	"austrian-business-infrastructure/internal/analysis"
	"austrian-business-infrastructure/internal/config"
	"austrian-business-infrastructure/internal/job"
	"austrian-business-infrastructure/internal/jobs"
	"austrian-business-infrastructure/pkg/cache"
	"austrian-business-infrastructure/pkg/database"
	"github.com/google/uuid"
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

	// Generate unique worker ID
	workerID := fmt.Sprintf("worker-%s-%d", hostname(), os.Getpid())
	logger.Info("starting worker", "worker_id", workerID)

	// Load configuration
	cfg, err := config.LoadWorkerConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

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

	// Initialize Redis connection (optional for worker, used for distributed locks)
	var redis *cache.Client
	if cfg.RedisURL != "" {
		redisConfig := cache.DefaultRedisConfig(cfg.RedisURL)
		redis, err = cache.NewClient(ctx, redisConfig)
		if err != nil {
			logger.Warn("failed to connect to redis, proceeding without distributed locks", "error", err)
		} else {
			defer redis.Close()
			logger.Info("connected to redis")
		}
	}

	// Initialize job queue
	queue := job.NewQueue(db.Pool, &job.QueueConfig{
		WorkerID: workerID,
		Logger:   logger,
	})

	// Initialize job registry with handlers
	registry := job.NewRegistry()
	registerJobHandlers(registry, db, redis, logger)

	// Initialize worker
	worker := job.NewWorker(queue, registry, &job.WorkerConfig{
		ID:              workerID,
		Concurrency:     cfg.WorkerConcurrency,
		PollInterval:    cfg.PollInterval,
		ShutdownTimeout: cfg.ShutdownTimeout,
		Logger:          logger,
	})

	// Initialize scheduler
	scheduler := job.NewScheduler(queue, db.Pool, &job.SchedulerConfig{
		Logger: logger,
	})

	// Start health check server
	healthServer := startHealthServer(cfg.HealthPort, db, redis, worker, logger)

	// Start scheduler
	schedulerDone := make(chan struct{})
	go func() {
		defer close(schedulerDone)
		if err := scheduler.Run(ctx); err != nil && ctx.Err() == nil {
			logger.Error("scheduler error", "error", err)
		}
	}()

	// Start worker
	workerDone := make(chan struct{})
	go func() {
		defer close(workerDone)
		if err := worker.Run(ctx); err != nil && ctx.Err() == nil {
			logger.Error("worker error", "error", err)
		}
	}()

	logger.Info("worker started",
		"concurrency", cfg.WorkerConcurrency,
		"poll_interval", cfg.PollInterval,
		"health_port", cfg.HealthPort)

	// Wait for shutdown signal
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)

	sig := <-shutdown
	logger.Info("shutdown signal received", "signal", sig)

	// Cancel context to stop worker and scheduler
	cancel()

	// Wait for worker to finish with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer shutdownCancel()

	// Shutdown health server
	if err := healthServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("health server shutdown failed", "error", err)
	}

	// Wait for scheduler
	select {
	case <-schedulerDone:
		logger.Info("scheduler stopped")
	case <-shutdownCtx.Done():
		logger.Warn("scheduler shutdown timeout")
	}

	// Wait for worker
	select {
	case <-workerDone:
		logger.Info("worker stopped")
	case <-shutdownCtx.Done():
		logger.Warn("worker shutdown timeout - some jobs may not have completed")
	}

	logger.Info("shutdown complete")
	return nil
}

// registerJobHandlers registers all job handlers with the registry
func registerJobHandlers(registry *job.Registry, db *database.Pool, redis *cache.Client, logger *slog.Logger) {
	// Initialize analysis service for document analysis jobs
	analysisRepo := analysis.NewRepository(db.Pool)
	analysisService := analysis.NewService(analysisRepo, analysis.ServiceConfig{}) // AI and OCR services configured via config

	// Register document analysis handler
	docAnalysisHandler := jobs.NewDocumentAnalysisHandler(
		db.Pool,
		analysisService,
		&jobs.DocumentAnalysisHandlerConfig{
			MaxRetries: 3,
			RetryDelay: 30 * time.Second,
			Logger:     logger,
		},
	)
	registry.Register(job.TypeDocumentAnalysis, docAnalysisHandler)

	// TODO: Register other job handlers as they are implemented
	// registry.Register(job.TypeDataboxSync, jobs.NewDataboxSyncHandler(db, logger))
	// registry.Register(job.TypeDeadlineReminder, jobs.NewDeadlineReminderHandler(db, logger))
	// registry.Register(job.TypeWatchlistCheck, jobs.NewWatchlistCheckHandler(db, logger))
	// registry.Register(job.TypeSessionCleanup, jobs.NewSessionCleanupHandler(db, logger))
	// registry.Register(job.TypeWebhookDelivery, jobs.NewWebhookDeliveryHandler(db, logger))
	// registry.Register(job.TypeAuditArchive, jobs.NewAuditArchiveHandler(db, logger))

	_ = redis
	logger.Info("job handlers registered", "handlers", []string{job.TypeDocumentAnalysis})
}

// startHealthServer starts the health check HTTP server
func startHealthServer(port int, db *database.Pool, redis *cache.Client, worker *job.Worker, logger *slog.Logger) *http.Server {
	mux := http.NewServeMux()

	// Liveness probe - basic check that process is running
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status":"ok"}`)
	})

	// Readiness probe - check dependencies
	mux.HandleFunc("GET /ready", func(w http.ResponseWriter, r *http.Request) {
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

		// Check Redis if available
		if redis != nil {
			if err := redis.Health(ctx); err != nil {
				checks["redis"] = "unhealthy: " + err.Error()
				healthy = false
			} else {
				checks["redis"] = "healthy"
			}
		}

		// Check worker status
		checks["worker"] = worker.Status()

		w.Header().Set("Content-Type", "application/json")
		if healthy {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
		}

		status := "ready"
		if !healthy {
			status = "not_ready"
		}
		fmt.Fprintf(w, `{"status":"%s","checks":%s}`, status, toJSON(checks))
	})

	// Metrics endpoint
	mux.HandleFunc("GET /metrics", func(w http.ResponseWriter, r *http.Request) {
		metrics := worker.Metrics()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `%s`, toJSON(metrics))
	})

	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	}

	go func() {
		logger.Info("health server listening", "port", port)
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			logger.Error("health server error", "error", err)
		}
	}()

	return server
}

// hostname returns the hostname or a default
func hostname() string {
	if h, err := os.Hostname(); err == nil {
		return h
	}
	return uuid.New().String()[:8]
}

// toJSON is a simple JSON serializer for maps
func toJSON(m interface{}) string {
	switch v := m.(type) {
	case map[string]string:
		result := "{"
		first := true
		for k, val := range v {
			if !first {
				result += ","
			}
			result += fmt.Sprintf(`"%s":"%s"`, k, val)
			first = false
		}
		result += "}"
		return result
	case *job.WorkerMetrics:
		return fmt.Sprintf(`{"jobs_processed":%d,"jobs_failed":%d,"jobs_succeeded":%d,"queue_length":%d,"active_jobs":%d}`,
			v.JobsProcessed, v.JobsFailed, v.JobsSucceeded, v.QueueLength, v.ActiveJobs)
	default:
		return "{}"
	}
}
