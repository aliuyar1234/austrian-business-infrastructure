package job

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Status constants for job states
const (
	StatusPending   = "pending"
	StatusRunning   = "running"
	StatusCompleted = "completed"
	StatusFailed    = "failed"
	StatusDead      = "dead"
)

// Priority levels
const (
	PriorityHigh   = 10
	PriorityNormal = 5
	PriorityLow    = 1
)

// Job types
const (
	TypeDataboxSync       = "databox_sync"
	TypeDocumentAnalysis  = "document_analysis"
	TypeDeadlineReminder  = "deadline_reminder"
	TypeWatchlistCheck    = "watchlist_check"
	TypeSessionCleanup    = "session_cleanup"
	TypeWebhookDelivery   = "webhook_delivery"
	TypeAuditArchive      = "audit_archive"
	TypeSoftDeleteCleanup = "soft_delete_cleanup"
)

// Sync intervals
const (
	IntervalHourly  = "hourly"
	Interval4Hourly = "4hourly"
	IntervalDaily   = "daily"
	IntervalWeekly  = "weekly"
	IntervalDisabled = "disabled"
)

// Job represents a background job in the queue
type Job struct {
	ID             uuid.UUID       `json:"id"`
	TenantID       uuid.UUID       `json:"tenant_id"`
	Type           string          `json:"type"`
	Payload        json.RawMessage `json:"payload"`
	Priority       int             `json:"priority"`
	Status         string          `json:"status"`
	MaxRetries     int             `json:"max_retries"`
	RetryCount     int             `json:"retry_count"`
	LastError      string          `json:"last_error,omitempty"`
	RunAt          time.Time       `json:"run_at"`
	StartedAt      *time.Time      `json:"started_at,omitempty"`
	CompletedAt    *time.Time      `json:"completed_at,omitempty"`
	TimeoutSeconds int             `json:"timeout_seconds"`
	WorkerID       string          `json:"worker_id,omitempty"`
	IdempotencyKey string          `json:"idempotency_key,omitempty"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

// Schedule represents a recurring job schedule
type Schedule struct {
	ID             uuid.UUID       `json:"id"`
	TenantID       uuid.UUID       `json:"tenant_id"`
	Name           string          `json:"name"`
	JobType        string          `json:"job_type"`
	JobPayload     json.RawMessage `json:"job_payload"`
	CronExpression string          `json:"cron_expression,omitempty"`
	Interval       string          `json:"interval,omitempty"` // hourly, 4hourly, daily, weekly
	Enabled        bool            `json:"enabled"`
	Timezone       string          `json:"timezone"`
	LastRunAt      *time.Time      `json:"last_run_at,omitempty"`
	NextRunAt      *time.Time      `json:"next_run_at,omitempty"`
	RunCount       int             `json:"run_count"`
	FailCount      int             `json:"fail_count"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

// JobHistory represents a completed job execution
type JobHistory struct {
	ID          uuid.UUID       `json:"id"`
	TenantID    uuid.UUID       `json:"tenant_id"`
	JobID       *uuid.UUID      `json:"job_id,omitempty"`
	ScheduleID  *uuid.UUID      `json:"schedule_id,omitempty"`
	Type        string          `json:"type"`
	Payload     json.RawMessage `json:"payload"`
	Status      string          `json:"status"` // completed, failed
	Result      json.RawMessage `json:"result,omitempty"`
	ErrorMessage string         `json:"error_message,omitempty"`
	StartedAt   time.Time       `json:"started_at"`
	CompletedAt time.Time       `json:"completed_at"`
	DurationMs  int             `json:"duration_ms"`
	WorkerID    string          `json:"worker_id"`
	CreatedAt   time.Time       `json:"created_at"`
}

// DeadLetter represents a permanently failed job
type DeadLetter struct {
	ID              uuid.UUID       `json:"id"`
	TenantID        uuid.UUID       `json:"tenant_id"`
	OriginalJobID   *uuid.UUID      `json:"original_job_id,omitempty"`
	Type            string          `json:"type"`
	Payload         json.RawMessage `json:"payload"`
	Errors          []string        `json:"errors"`
	MaxRetries      int             `json:"max_retries"`
	TotalAttempts   int             `json:"total_attempts"`
	FirstAttemptedAt time.Time      `json:"first_attempted_at"`
	LastAttemptedAt  time.Time      `json:"last_attempted_at"`
	Acknowledged    bool            `json:"acknowledged"`
	AcknowledgedBy  *uuid.UUID      `json:"acknowledged_by,omitempty"`
	AcknowledgedAt  *time.Time      `json:"acknowledged_at,omitempty"`
	CreatedAt       time.Time       `json:"created_at"`
}

// Handler is the interface that job handlers must implement
type Handler interface {
	// Handle processes a job and returns a result or error
	Handle(ctx context.Context, job *Job) (json.RawMessage, error)
}

// HandlerFunc is an adapter to allow use of ordinary functions as job handlers
type HandlerFunc func(ctx context.Context, job *Job) (json.RawMessage, error)

// Handle calls f(ctx, job)
func (f HandlerFunc) Handle(ctx context.Context, job *Job) (json.RawMessage, error) {
	return f(ctx, job)
}

// EnqueueOptions provides options when enqueuing a job
type EnqueueOptions struct {
	Priority       int
	RunAt          time.Time
	MaxRetries     int
	TimeoutSeconds int
	IdempotencyKey string
}

// DefaultEnqueueOptions returns default options for enqueueing
func DefaultEnqueueOptions() *EnqueueOptions {
	return &EnqueueOptions{
		Priority:       PriorityNormal,
		RunAt:          time.Now(),
		MaxRetries:     3,
		TimeoutSeconds: 1800, // 30 minutes
	}
}

// JobResult represents the result of a job execution
type JobResult struct {
	Success bool            `json:"success"`
	Data    json.RawMessage `json:"data,omitempty"`
	Error   string          `json:"error,omitempty"`
}

// WorkerMetrics contains worker statistics
type WorkerMetrics struct {
	JobsProcessed int64 `json:"jobs_processed"`
	JobsFailed    int64 `json:"jobs_failed"`
	JobsSucceeded int64 `json:"jobs_succeeded"`
	QueueLength   int64 `json:"queue_length"`
	ActiveJobs    int   `json:"active_jobs"`
}

// IntervalToDuration converts an interval string to duration
func IntervalToDuration(interval string) time.Duration {
	switch interval {
	case IntervalHourly:
		return time.Hour
	case Interval4Hourly:
		return 4 * time.Hour
	case IntervalDaily:
		return 24 * time.Hour
	case IntervalWeekly:
		return 7 * 24 * time.Hour
	default:
		return 4 * time.Hour // default
	}
}

// PayloadTo unmarshals the job payload into the given struct
func (j *Job) PayloadTo(v interface{}) error {
	return json.Unmarshal(j.Payload, v)
}

// SetPayload marshals the given value into the job payload
func (j *Job) SetPayload(v interface{}) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	j.Payload = data
	return nil
}
