package job

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Scheduler errors
var (
	ErrScheduleNotFound = errors.New("schedule not found")
)

// Scheduler manages cron-style job scheduling
type Scheduler struct {
	db       *pgxpool.Pool
	queue    *Queue
	logger   *slog.Logger
	interval time.Duration
}

// SchedulerConfig holds scheduler configuration
type SchedulerConfig struct {
	Logger   *slog.Logger
	Interval time.Duration // How often to check for due schedules
}

// NewScheduler creates a new scheduler
func NewScheduler(queue *Queue, db *pgxpool.Pool, cfg *SchedulerConfig) *Scheduler {
	logger := slog.Default()
	interval := 30 * time.Second

	if cfg != nil {
		if cfg.Logger != nil {
			logger = cfg.Logger
		}
		if cfg.Interval > 0 {
			interval = cfg.Interval
		}
	}

	return &Scheduler{
		db:       db,
		queue:    queue,
		logger:   logger,
		interval: interval,
	}
}

// Run starts the scheduler loop
func (s *Scheduler) Run(ctx context.Context) error {
	s.logger.Info("scheduler starting", "interval", s.interval)

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	// Run immediately on start
	if err := s.processDueSchedules(ctx); err != nil {
		s.logger.Error("failed to process due schedules", "error", err)
	}

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("scheduler stopping")
			return ctx.Err()

		case <-ticker.C:
			if err := s.processDueSchedules(ctx); err != nil {
				s.logger.Error("failed to process due schedules", "error", err)
			}
		}
	}
}

// processDueSchedules finds and enqueues jobs for due schedules
func (s *Scheduler) processDueSchedules(ctx context.Context) error {
	now := time.Now()

	// Get due schedules
	query := `
		SELECT id, tenant_id, name, job_type, job_payload, cron_expression, interval,
		       timezone, last_run_at, next_run_at, run_count, fail_count
		FROM schedules
		WHERE enabled = TRUE AND next_run_at <= $1
		ORDER BY next_run_at ASC
		FOR UPDATE SKIP LOCKED
	`

	rows, err := s.db.Query(ctx, query, now)
	if err != nil {
		return fmt.Errorf("query due schedules: %w", err)
	}
	defer rows.Close()

	var schedules []*Schedule
	for rows.Next() {
		schedule := &Schedule{}
		var cronExpr, intervalStr *string

		err := rows.Scan(
			&schedule.ID, &schedule.TenantID, &schedule.Name, &schedule.JobType,
			&schedule.JobPayload, &cronExpr, &intervalStr, &schedule.Timezone,
			&schedule.LastRunAt, &schedule.NextRunAt, &schedule.RunCount, &schedule.FailCount,
		)
		if err != nil {
			return fmt.Errorf("scan schedule: %w", err)
		}

		if cronExpr != nil {
			schedule.CronExpression = *cronExpr
		}
		if intervalStr != nil {
			schedule.Interval = *intervalStr
		}

		schedules = append(schedules, schedule)
	}

	// Process each due schedule
	for _, schedule := range schedules {
		if err := s.enqueueForSchedule(ctx, schedule, now); err != nil {
			s.logger.Error("failed to enqueue job for schedule",
				"schedule_id", schedule.ID,
				"schedule_name", schedule.Name,
				"error", err)
			continue
		}
	}

	return nil
}

// enqueueForSchedule creates a job for a schedule and updates next run time
func (s *Scheduler) enqueueForSchedule(ctx context.Context, schedule *Schedule, now time.Time) error {
	// Create idempotency key to prevent duplicate jobs
	idempotencyKey := fmt.Sprintf("schedule-%s-%d", schedule.ID, now.Unix()/60)

	// Enqueue the job
	opts := &EnqueueOptions{
		Priority:       PriorityNormal,
		RunAt:          now,
		MaxRetries:     3,
		TimeoutSeconds: 1800,
		IdempotencyKey: idempotencyKey,
	}

	_, err := s.queue.Enqueue(ctx, schedule.TenantID, schedule.JobType, schedule.JobPayload, opts)
	if err != nil && !errors.Is(err, ErrDuplicateJob) {
		// Update fail count
		s.updateScheduleFail(ctx, schedule.ID)
		return fmt.Errorf("enqueue job: %w", err)
	}

	// Calculate next run time
	nextRun := s.calculateNextRun(schedule, now)

	// Update schedule
	query := `
		UPDATE schedules
		SET last_run_at = $1, next_run_at = $2, run_count = run_count + 1, updated_at = $1
		WHERE id = $3
	`

	_, err = s.db.Exec(ctx, query, now, nextRun, schedule.ID)
	if err != nil {
		return fmt.Errorf("update schedule: %w", err)
	}

	s.logger.Debug("job enqueued for schedule",
		"schedule_id", schedule.ID,
		"schedule_name", schedule.Name,
		"job_type", schedule.JobType,
		"next_run", nextRun)

	return nil
}

// calculateNextRun determines the next execution time for a schedule
func (s *Scheduler) calculateNextRun(schedule *Schedule, from time.Time) time.Time {
	// If interval is set, use it
	if schedule.Interval != "" {
		duration := IntervalToDuration(schedule.Interval)
		return from.Add(duration)
	}

	// If cron expression is set, parse and calculate
	if schedule.CronExpression != "" {
		// For now, use simple interval parsing
		// A full cron parser would be implemented with robfig/cron
		// This is a simplified version
		next, err := parseCronNext(schedule.CronExpression, from)
		if err == nil {
			return next
		}
		s.logger.Warn("failed to parse cron expression, using default",
			"cron", schedule.CronExpression,
			"error", err)
	}

	// Default: 4 hours
	return from.Add(4 * time.Hour)
}

// updateScheduleFail increments the fail count for a schedule
func (s *Scheduler) updateScheduleFail(ctx context.Context, scheduleID uuid.UUID) {
	query := `UPDATE schedules SET fail_count = fail_count + 1, updated_at = NOW() WHERE id = $1`
	s.db.Exec(ctx, query, scheduleID)
}

// CreateSchedule creates a new schedule
func (s *Scheduler) CreateSchedule(ctx context.Context, schedule *Schedule) error {
	now := time.Now()

	if schedule.ID == uuid.Nil {
		schedule.ID = uuid.New()
	}
	schedule.CreatedAt = now
	schedule.UpdatedAt = now

	// Calculate first next_run_at
	if schedule.NextRunAt == nil {
		nextRun := s.calculateNextRun(schedule, now)
		schedule.NextRunAt = &nextRun
	}

	query := `
		INSERT INTO schedules (
			id, tenant_id, name, job_type, job_payload, cron_expression, interval,
			enabled, timezone, next_run_at, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`

	_, err := s.db.Exec(ctx, query,
		schedule.ID, schedule.TenantID, schedule.Name, schedule.JobType, schedule.JobPayload,
		nullString(schedule.CronExpression), nullString(schedule.Interval),
		schedule.Enabled, schedule.Timezone, schedule.NextRunAt, schedule.CreatedAt, schedule.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("create schedule: %w", err)
	}

	return nil
}

// GetSchedule retrieves a schedule by ID
func (s *Scheduler) GetSchedule(ctx context.Context, id uuid.UUID) (*Schedule, error) {
	query := `
		SELECT id, tenant_id, name, job_type, job_payload, cron_expression, interval,
		       enabled, timezone, last_run_at, next_run_at, run_count, fail_count,
		       created_at, updated_at
		FROM schedules WHERE id = $1
	`

	schedule := &Schedule{}
	var cronExpr, intervalStr *string

	err := s.db.QueryRow(ctx, query, id).Scan(
		&schedule.ID, &schedule.TenantID, &schedule.Name, &schedule.JobType, &schedule.JobPayload,
		&cronExpr, &intervalStr, &schedule.Enabled, &schedule.Timezone,
		&schedule.LastRunAt, &schedule.NextRunAt, &schedule.RunCount, &schedule.FailCount,
		&schedule.CreatedAt, &schedule.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrScheduleNotFound
		}
		return nil, fmt.Errorf("get schedule: %w", err)
	}

	if cronExpr != nil {
		schedule.CronExpression = *cronExpr
	}
	if intervalStr != nil {
		schedule.Interval = *intervalStr
	}

	return schedule, nil
}

// UpdateSchedule updates an existing schedule
func (s *Scheduler) UpdateSchedule(ctx context.Context, schedule *Schedule) error {
	schedule.UpdatedAt = time.Now()

	// Recalculate next run if interval or cron changed
	nextRun := s.calculateNextRun(schedule, time.Now())
	schedule.NextRunAt = &nextRun

	query := `
		UPDATE schedules SET
			name = $1, job_type = $2, job_payload = $3, cron_expression = $4,
			interval = $5, enabled = $6, timezone = $7, next_run_at = $8, updated_at = $9
		WHERE id = $10
	`

	_, err := s.db.Exec(ctx, query,
		schedule.Name, schedule.JobType, schedule.JobPayload,
		nullString(schedule.CronExpression), nullString(schedule.Interval),
		schedule.Enabled, schedule.Timezone, schedule.NextRunAt, schedule.UpdatedAt, schedule.ID,
	)
	if err != nil {
		return fmt.Errorf("update schedule: %w", err)
	}

	return nil
}

// DeleteSchedule removes a schedule
func (s *Scheduler) DeleteSchedule(ctx context.Context, id uuid.UUID) error {
	_, err := s.db.Exec(ctx, `DELETE FROM schedules WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete schedule: %w", err)
	}
	return nil
}

// ListSchedules returns schedules for a tenant
func (s *Scheduler) ListSchedules(ctx context.Context, tenantID uuid.UUID) ([]*Schedule, error) {
	query := `
		SELECT id, tenant_id, name, job_type, job_payload, cron_expression, interval,
		       enabled, timezone, last_run_at, next_run_at, run_count, fail_count,
		       created_at, updated_at
		FROM schedules WHERE tenant_id = $1
		ORDER BY name
	`

	rows, err := s.db.Query(ctx, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("list schedules: %w", err)
	}
	defer rows.Close()

	var schedules []*Schedule
	for rows.Next() {
		schedule := &Schedule{}
		var cronExpr, intervalStr *string

		err := rows.Scan(
			&schedule.ID, &schedule.TenantID, &schedule.Name, &schedule.JobType, &schedule.JobPayload,
			&cronExpr, &intervalStr, &schedule.Enabled, &schedule.Timezone,
			&schedule.LastRunAt, &schedule.NextRunAt, &schedule.RunCount, &schedule.FailCount,
			&schedule.CreatedAt, &schedule.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan schedule: %w", err)
		}

		if cronExpr != nil {
			schedule.CronExpression = *cronExpr
		}
		if intervalStr != nil {
			schedule.Interval = *intervalStr
		}

		schedules = append(schedules, schedule)
	}

	return schedules, nil
}

// EnableSchedule enables a schedule
func (s *Scheduler) EnableSchedule(ctx context.Context, id uuid.UUID) error {
	_, err := s.db.Exec(ctx, `UPDATE schedules SET enabled = TRUE, updated_at = NOW() WHERE id = $1`, id)
	return err
}

// DisableSchedule disables a schedule
func (s *Scheduler) DisableSchedule(ctx context.Context, id uuid.UUID) error {
	_, err := s.db.Exec(ctx, `UPDATE schedules SET enabled = FALSE, updated_at = NOW() WHERE id = $1`, id)
	return err
}

// parseCronNext parses a cron expression and returns the next run time
// This is a simplified implementation - for production, use robfig/cron
func parseCronNext(expr string, from time.Time) (time.Time, error) {
	// Simple cron format support: minute hour day-of-month month day-of-week
	// For MVP, we support only predefined patterns

	switch expr {
	case "0 * * * *": // Every hour at minute 0
		next := from.Truncate(time.Hour).Add(time.Hour)
		return next, nil

	case "0 */4 * * *": // Every 4 hours at minute 0
		next := from.Truncate(time.Hour)
		for next.Before(from) || next.Hour()%4 != 0 {
			next = next.Add(time.Hour)
		}
		return next, nil

	case "0 6 * * *": // Every day at 6:00 AM
		next := time.Date(from.Year(), from.Month(), from.Day(), 6, 0, 0, 0, from.Location())
		if next.Before(from) {
			next = next.AddDate(0, 0, 1)
		}
		return next, nil

	case "0 7 * * *": // Every day at 7:00 AM
		next := time.Date(from.Year(), from.Month(), from.Day(), 7, 0, 0, 0, from.Location())
		if next.Before(from) {
			next = next.AddDate(0, 0, 1)
		}
		return next, nil

	case "0 8 * * *": // Every day at 8:00 AM
		next := time.Date(from.Year(), from.Month(), from.Day(), 8, 0, 0, 0, from.Location())
		if next.Before(from) {
			next = next.AddDate(0, 0, 1)
		}
		return next, nil

	case "0 0 * * 0": // Every Sunday at midnight
		next := from
		for next.Weekday() != time.Sunday || next.Before(from) {
			next = next.AddDate(0, 0, 1)
		}
		return time.Date(next.Year(), next.Month(), next.Day(), 0, 0, 0, 0, from.Location()), nil

	default:
		return time.Time{}, fmt.Errorf("unsupported cron expression: %s", expr)
	}
}
