package job

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"
)

// Worker processes jobs from the queue
type Worker struct {
	id           string
	queue        *Queue
	registry     *Registry
	concurrency  int
	pollInterval time.Duration
	shutdownTimeout time.Duration
	logger       *slog.Logger

	// Metrics
	jobsProcessed atomic.Int64
	jobsFailed    atomic.Int64
	jobsSucceeded atomic.Int64
	activeJobs    atomic.Int32

	// State
	running atomic.Bool
	mu      sync.Mutex
}

// WorkerConfig holds worker configuration
type WorkerConfig struct {
	ID              string
	Concurrency     int
	PollInterval    time.Duration
	ShutdownTimeout time.Duration
	Logger          *slog.Logger
}

// NewWorker creates a new job worker
func NewWorker(queue *Queue, registry *Registry, cfg *WorkerConfig) *Worker {
	concurrency := 5
	pollInterval := 1 * time.Second
	shutdownTimeout := 30 * time.Second
	logger := slog.Default()
	id := "worker"

	if cfg != nil {
		if cfg.Concurrency > 0 {
			concurrency = cfg.Concurrency
		}
		if cfg.PollInterval > 0 {
			pollInterval = cfg.PollInterval
		}
		if cfg.ShutdownTimeout > 0 {
			shutdownTimeout = cfg.ShutdownTimeout
		}
		if cfg.Logger != nil {
			logger = cfg.Logger
		}
		if cfg.ID != "" {
			id = cfg.ID
		}
	}

	return &Worker{
		id:              id,
		queue:           queue,
		registry:        registry,
		concurrency:     concurrency,
		pollInterval:    pollInterval,
		shutdownTimeout: shutdownTimeout,
		logger:          logger,
	}
}

// Run starts the worker and processes jobs until context is cancelled
func (w *Worker) Run(ctx context.Context) error {
	w.running.Store(true)
	defer w.running.Store(false)

	w.logger.Info("worker starting",
		"id", w.id,
		"concurrency", w.concurrency,
		"poll_interval", w.pollInterval)

	// Create semaphore for concurrency control
	sem := make(chan struct{}, w.concurrency)

	// Create WaitGroup to track active jobs
	var wg sync.WaitGroup

	// Job processing loop
	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()

	// Also run stale job cleanup periodically
	cleanupTicker := time.NewTicker(5 * time.Minute)
	defer cleanupTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			w.logger.Info("worker stopping, waiting for active jobs", "active_jobs", w.activeJobs.Load())

			// Wait for active jobs with timeout
			done := make(chan struct{})
			go func() {
				wg.Wait()
				close(done)
			}()

			select {
			case <-done:
				w.logger.Info("all jobs completed")
			case <-time.After(w.shutdownTimeout):
				w.logger.Warn("shutdown timeout exceeded, some jobs may not have completed",
					"active_jobs", w.activeJobs.Load())
			}

			return ctx.Err()

		case <-cleanupTicker.C:
			// Cleanup stale jobs
			if _, err := w.queue.CleanupStaleJobs(ctx); err != nil {
				w.logger.Error("failed to cleanup stale jobs", "error", err)
			}

		case <-ticker.C:
			// Try to get a job
			select {
			case sem <- struct{}{}:
				// Got a slot, try to dequeue
				job, err := w.queue.Dequeue(ctx)
				if err != nil {
					<-sem // Release slot
					if !errors.Is(err, ErrNoJobsAvailable) {
						w.logger.Error("failed to dequeue job", "error", err)
					}
					continue
				}

				// Process job in goroutine
				wg.Add(1)
				w.activeJobs.Add(1)
				go func(j *Job) {
					defer func() {
						<-sem // Release slot
						wg.Done()
						w.activeJobs.Add(-1)
					}()

					w.processJob(ctx, j)
				}(job)

			default:
				// All slots busy, wait for next tick
			}
		}
	}
}

// processJob handles a single job execution
func (w *Worker) processJob(ctx context.Context, job *Job) {
	startTime := time.Now()
	logger := w.logger.With(
		"job_id", job.ID,
		"job_type", job.Type,
		"tenant_id", job.TenantID,
	)

	logger.Info("processing job")

	// Get handler for job type
	handler, err := w.registry.Get(job.Type)
	if err != nil {
		logger.Error("no handler for job type", "error", err)
		if err := w.queue.Fail(ctx, job.ID, fmt.Sprintf("no handler for job type: %s", job.Type)); err != nil {
			logger.Error("failed to mark job as failed", "error", err)
		}
		w.jobsFailed.Add(1)
		w.jobsProcessed.Add(1)
		return
	}

	// Create timeout context for job
	jobCtx, cancel := context.WithTimeout(ctx, time.Duration(job.TimeoutSeconds)*time.Second)
	defer cancel()

	// Execute handler with panic recovery
	var result []byte
	var execErr error
	func() {
		defer func() {
			if r := recover(); r != nil {
				execErr = fmt.Errorf("job panicked: %v", r)
				logger.Error("job panicked", "panic", r)
			}
		}()

		result, execErr = handler.Handle(jobCtx, job)
	}()

	duration := time.Since(startTime)
	w.jobsProcessed.Add(1)

	if execErr != nil {
		logger.Error("job failed",
			"error", execErr,
			"duration", duration,
			"retry_count", job.RetryCount+1,
			"max_retries", job.MaxRetries)

		if err := w.queue.Fail(ctx, job.ID, execErr.Error()); err != nil {
			logger.Error("failed to mark job as failed", "error", err)
		}
		w.jobsFailed.Add(1)
		return
	}

	// Mark job as completed
	if err := w.queue.Complete(ctx, job.ID, result); err != nil {
		logger.Error("failed to mark job as completed", "error", err)
		w.jobsFailed.Add(1)
		return
	}

	w.jobsSucceeded.Add(1)
	logger.Info("job completed", "duration", duration)
}

// Status returns the current worker status
func (w *Worker) Status() string {
	if w.running.Load() {
		return "running"
	}
	return "stopped"
}

// Metrics returns current worker metrics
func (w *Worker) Metrics() *WorkerMetrics {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	queueLength, _ := w.queue.QueueLength(ctx)

	return &WorkerMetrics{
		JobsProcessed: w.jobsProcessed.Load(),
		JobsFailed:    w.jobsFailed.Load(),
		JobsSucceeded: w.jobsSucceeded.Load(),
		QueueLength:   queueLength,
		ActiveJobs:    int(w.activeJobs.Load()),
	}
}

// ResetMetrics resets the worker metrics
func (w *Worker) ResetMetrics() {
	w.jobsProcessed.Store(0)
	w.jobsFailed.Store(0)
	w.jobsSucceeded.Store(0)
}
