package integration

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"austrian-business-infrastructure/internal/job"
	"github.com/google/uuid"
)

// MockDB implements a minimal in-memory job store for testing
type MockJobStore struct {
	jobs map[uuid.UUID]*job.Job
}

func NewMockJobStore() *MockJobStore {
	return &MockJobStore{
		jobs: make(map[uuid.UUID]*job.Job),
	}
}

func TestJobTypes(t *testing.T) {
	t.Run("job status constants are defined", func(t *testing.T) {
		statuses := []string{
			job.StatusPending,
			job.StatusRunning,
			job.StatusCompleted,
			job.StatusFailed,
			job.StatusDead,
		}

		for _, s := range statuses {
			if s == "" {
				t.Error("status constant is empty")
			}
		}
	})

	t.Run("job priority levels are defined", func(t *testing.T) {
		if job.PriorityHigh <= job.PriorityNormal {
			t.Error("PriorityHigh should be greater than PriorityNormal")
		}
		if job.PriorityNormal <= job.PriorityLow {
			t.Error("PriorityNormal should be greater than PriorityLow")
		}
	})

	t.Run("job type constants are defined", func(t *testing.T) {
		types := []string{
			job.TypeDataboxSync,
			job.TypeDeadlineReminder,
			job.TypeWatchlistCheck,
			job.TypeWebhookDelivery,
			job.TypeAuditArchive,
			job.TypeSessionCleanup,
		}

		for _, typ := range types {
			if typ == "" {
				t.Error("job type constant is empty")
			}
		}
	})
}

func TestJobStruct(t *testing.T) {
	t.Run("create job with all fields", func(t *testing.T) {
		tenantID := uuid.New()
		payload := json.RawMessage(`{"key": "value"}`)

		j := &job.Job{
			ID:         uuid.New(),
			TenantID:   tenantID,
			Type:       job.TypeDataboxSync,
			Payload:    payload,
			Priority:   job.PriorityNormal,
			Status:     job.StatusPending,
			MaxRetries: 3,
			RetryCount: 0,
			RunAt:      time.Now(),
			CreatedAt:  time.Now(),
		}

		if j.ID == uuid.Nil {
			t.Error("job ID should not be nil")
		}
		if j.TenantID != tenantID {
			t.Error("tenant ID mismatch")
		}
		if j.Type != job.TypeDataboxSync {
			t.Error("job type mismatch")
		}
		if j.Priority != job.PriorityNormal {
			t.Error("priority mismatch")
		}
	})

	t.Run("job payload serialization", func(t *testing.T) {
		type TestPayload struct {
			AccountID string `json:"account_id"`
			Force     bool   `json:"force"`
		}

		payload := TestPayload{
			AccountID: uuid.New().String(),
			Force:     true,
		}

		data, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("failed to marshal payload: %v", err)
		}

		j := &job.Job{
			Payload: data,
		}

		var parsed TestPayload
		if err := json.Unmarshal(j.Payload, &parsed); err != nil {
			t.Fatalf("failed to unmarshal payload: %v", err)
		}

		if parsed.AccountID != payload.AccountID {
			t.Error("account ID mismatch after serialization")
		}
		if parsed.Force != payload.Force {
			t.Error("force flag mismatch after serialization")
		}
	})
}

func TestScheduleStructInQueue(t *testing.T) {
	t.Run("create schedule with cron expression", func(t *testing.T) {
		s := &job.Schedule{
			ID:             uuid.New(),
			TenantID:       uuid.New(),
			Name:           "daily-sync",
			JobType:        job.TypeDataboxSync,
			CronExpression: "0 6 * * *", // 6 AM daily
			Enabled:        true,
			Timezone:       "UTC",
		}

		if s.Name != "daily-sync" {
			t.Error("schedule name mismatch")
		}
		if s.CronExpression != "0 6 * * *" {
			t.Error("cron expression mismatch")
		}
	})

	t.Run("create schedule with interval", func(t *testing.T) {
		s := &job.Schedule{
			ID:       uuid.New(),
			TenantID: uuid.New(),
			Name:     "hourly-check",
			JobType:  job.TypeWatchlistCheck,
			Interval: "hourly",
			Enabled:  true,
		}

		if s.Interval != "hourly" {
			t.Error("interval mismatch")
		}
	})
}

func TestJobHistoryStruct(t *testing.T) {
	t.Run("create job history entry", func(t *testing.T) {
		now := time.Now()
		completed := now.Add(5 * time.Second)
		jobID := uuid.New()

		h := &job.JobHistory{
			ID:          uuid.New(),
			TenantID:    uuid.New(),
			JobID:       &jobID,
			Type:        job.TypeDataboxSync,
			Status:      job.StatusCompleted,
			StartedAt:   now,
			CompletedAt: completed,
			DurationMs:  5000,
			WorkerID:    "worker-1",
		}

		if h.Status != job.StatusCompleted {
			t.Error("status mismatch")
		}
		if h.DurationMs != 5000 {
			t.Error("duration mismatch")
		}
	})

	t.Run("job history with error", func(t *testing.T) {
		h := &job.JobHistory{
			ID:           uuid.New(),
			Type:         job.TypeWebhookDelivery,
			Status:       job.StatusFailed,
			ErrorMessage: "connection timeout",
		}

		if h.Status != job.StatusFailed {
			t.Error("status should be failed")
		}
		if h.ErrorMessage == "" {
			t.Error("error message should be set")
		}
	})
}

func TestDeadLetterStruct(t *testing.T) {
	t.Run("create dead letter entry", func(t *testing.T) {
		originalJobID := uuid.New()
		dl := &job.DeadLetter{
			ID:               uuid.New(),
			TenantID:         uuid.New(),
			OriginalJobID:    &originalJobID,
			Type:             job.TypeWebhookDelivery,
			Payload:          json.RawMessage(`{"url": "https://example.com"}`),
			Errors:           []string{"attempt 1: timeout", "attempt 2: timeout", "attempt 3: timeout"},
			MaxRetries:       3,
			TotalAttempts:    3,
			FirstAttemptedAt: time.Now().Add(-1 * time.Hour),
			LastAttemptedAt:  time.Now(),
			Acknowledged:     false,
		}

		if len(dl.Errors) != 3 {
			t.Errorf("expected 3 errors, got %d", len(dl.Errors))
		}
		if dl.Acknowledged {
			t.Error("should not be acknowledged initially")
		}
	})
}

func TestHandlerInterface(t *testing.T) {
	t.Run("handler interface compliance", func(t *testing.T) {
		// Create a test handler
		handler := &testHandler{}

		// Verify it implements the Handler interface
		var _ job.Handler = handler
	})
}

// testHandler is a mock handler for testing
type testHandler struct {
	called bool
}

func (h *testHandler) Handle(ctx context.Context, j *job.Job) (json.RawMessage, error) {
	h.called = true
	return json.RawMessage(`{"status": "ok"}`), nil
}

func TestRegistryOperations(t *testing.T) {
	t.Run("register and get handler", func(t *testing.T) {
		registry := job.NewRegistry()
		handler := &testHandler{}

		err := registry.Register("test_job", handler)
		if err != nil {
			t.Fatalf("failed to register handler: %v", err)
		}

		retrieved, err := registry.Get("test_job")
		if err != nil {
			t.Fatalf("failed to get handler: %v", err)
		}

		if retrieved == nil {
			t.Error("retrieved handler should not be nil")
		}
	})

	t.Run("get unregistered handler returns error", func(t *testing.T) {
		registry := job.NewRegistry()

		_, err := registry.Get("nonexistent")
		if err == nil {
			t.Error("expected error for unregistered handler")
		}
	})

	t.Run("duplicate registration returns error", func(t *testing.T) {
		registry := job.NewRegistry()
		handler := &testHandler{}

		err := registry.Register("test_job", handler)
		if err != nil {
			t.Fatalf("first registration failed: %v", err)
		}

		err = registry.Register("test_job", handler)
		if err == nil {
			t.Error("expected error for duplicate registration")
		}
	})

	t.Run("must register panics on duplicate", func(t *testing.T) {
		registry := job.NewRegistry()
		handler := &testHandler{}

		registry.MustRegister("test_job", handler)

		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic for duplicate MustRegister")
			}
		}()

		registry.MustRegister("test_job", handler)
	})
}

func TestEnqueueOptions(t *testing.T) {
	t.Run("default enqueue options", func(t *testing.T) {
		opts := job.DefaultEnqueueOptions()

		if opts.Priority != job.PriorityNormal {
			t.Errorf("expected priority %d, got %d", job.PriorityNormal, opts.Priority)
		}
		if opts.MaxRetries != 3 {
			t.Errorf("expected max retries 3, got %d", opts.MaxRetries)
		}
		if opts.TimeoutSeconds != 1800 {
			t.Errorf("expected timeout 1800, got %d", opts.TimeoutSeconds)
		}
	})

	t.Run("custom enqueue options", func(t *testing.T) {
		runAt := time.Now().Add(1 * time.Hour)
		opts := &job.EnqueueOptions{
			Priority:       job.PriorityHigh,
			RunAt:          runAt,
			MaxRetries:     5,
			TimeoutSeconds: 3600,
			IdempotencyKey: "unique-key-123",
		}

		if opts.Priority != job.PriorityHigh {
			t.Error("priority mismatch")
		}
		if !opts.RunAt.Equal(runAt) {
			t.Error("run at mismatch")
		}
		if opts.MaxRetries != 5 {
			t.Error("max retries mismatch")
		}
		if opts.TimeoutSeconds != 3600 {
			t.Error("timeout seconds mismatch")
		}
		if opts.IdempotencyKey != "unique-key-123" {
			t.Error("idempotency key mismatch")
		}
	})
}

func TestQueueConfig(t *testing.T) {
	t.Run("queue config with worker id", func(t *testing.T) {
		cfg := &job.QueueConfig{
			WorkerID: "test-worker",
		}

		if cfg.WorkerID != "test-worker" {
			t.Error("worker id mismatch")
		}
	})
}

func TestWorkerConfig(t *testing.T) {
	t.Run("worker config defaults", func(t *testing.T) {
		cfg := &job.WorkerConfig{}

		// Verify config can be created
		if cfg.Concurrency < 0 {
			t.Error("concurrency should not be negative")
		}
	})

	t.Run("worker config custom values", func(t *testing.T) {
		cfg := &job.WorkerConfig{
			ID:              "test-worker",
			Concurrency:     10,
			PollInterval:    5 * time.Second,
			ShutdownTimeout: 30 * time.Second,
		}

		if cfg.Concurrency != 10 {
			t.Error("concurrency mismatch")
		}
		if cfg.PollInterval != 5*time.Second {
			t.Error("poll interval mismatch")
		}
		if cfg.ShutdownTimeout != 30*time.Second {
			t.Error("shutdown timeout mismatch")
		}
	})
}

func TestIntervalToDuration(t *testing.T) {
	tests := []struct {
		interval string
		expected time.Duration
	}{
		{job.IntervalHourly, time.Hour},
		{job.Interval4Hourly, 4 * time.Hour},
		{job.IntervalDaily, 24 * time.Hour},
		{job.IntervalWeekly, 7 * 24 * time.Hour},
		{"unknown", 4 * time.Hour}, // default
	}

	for _, tt := range tests {
		t.Run(tt.interval, func(t *testing.T) {
			duration := job.IntervalToDuration(tt.interval)
			if duration != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, duration)
			}
		})
	}
}
