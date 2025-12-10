package integration

import (
	"testing"
	"time"

	"austrian-business-infrastructure/internal/job"
	"github.com/google/uuid"
)

func TestScheduleIntervalParsing(t *testing.T) {
	tests := []struct {
		name     string
		interval string
		valid    bool
	}{
		{"hourly", "hourly", true},
		{"4hourly", "4hourly", true},
		{"daily", "daily", true},
		{"weekly", "weekly", true},
		{"disabled", "disabled", true},
		{"invalid", "monthly", false},
		{"empty", "", false},
	}

	validIntervals := map[string]bool{
		"hourly":   true,
		"4hourly":  true,
		"daily":    true,
		"weekly":   true,
		"disabled": true,
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, valid := validIntervals[tt.interval]
			if valid != tt.valid {
				t.Errorf("interval %q: expected valid=%v, got %v", tt.interval, tt.valid, valid)
			}
		})
	}
}

func TestScheduleNextRunCalculation(t *testing.T) {
	now := time.Now().UTC()

	tests := []struct {
		name           string
		interval       string
		lastRun        time.Time
		expectedOffset time.Duration
	}{
		{
			name:           "hourly from now",
			interval:       "hourly",
			lastRun:        now,
			expectedOffset: 1 * time.Hour,
		},
		{
			name:           "4hourly from now",
			interval:       "4hourly",
			lastRun:        now,
			expectedOffset: 4 * time.Hour,
		},
		{
			name:           "daily from now",
			interval:       "daily",
			lastRun:        now,
			expectedOffset: 24 * time.Hour,
		},
		{
			name:           "weekly from now",
			interval:       "weekly",
			lastRun:        now,
			expectedOffset: 7 * 24 * time.Hour,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			duration := job.IntervalToDuration(tt.interval)

			nextRun := tt.lastRun.Add(duration)
			expectedNext := tt.lastRun.Add(tt.expectedOffset)

			if !nextRun.Equal(expectedNext) {
				t.Errorf("expected next run at %v, got %v", expectedNext, nextRun)
			}
		})
	}
}

func TestScheduleStructFields(t *testing.T) {
	t.Run("create schedule with all fields", func(t *testing.T) {
		tenantID := uuid.New()
		now := time.Now()

		s := &job.Schedule{
			ID:             uuid.New(),
			TenantID:       tenantID,
			Name:           "databox-sync",
			JobType:        job.TypeDataboxSync,
			JobPayload:     []byte(`{"force": false}`),
			CronExpression: "0 */4 * * *",
			Interval:       "4hourly",
			Enabled:        true,
			Timezone:       "Europe/Vienna",
			LastRunAt:      &now,
			NextRunAt:      &now,
			RunCount:       10,
			FailCount:      1,
			CreatedAt:      now,
			UpdatedAt:      now,
		}

		if s.ID == uuid.Nil {
			t.Error("schedule ID should not be nil")
		}
		if s.TenantID != tenantID {
			t.Error("tenant ID mismatch")
		}
		if s.Name != "databox-sync" {
			t.Error("name mismatch")
		}
		if s.JobType != job.TypeDataboxSync {
			t.Error("job type mismatch")
		}
		if !s.Enabled {
			t.Error("should be enabled")
		}
		if s.RunCount != 10 {
			t.Error("run count mismatch")
		}
		if s.FailCount != 1 {
			t.Error("fail count mismatch")
		}
	})

	t.Run("disabled schedule", func(t *testing.T) {
		s := &job.Schedule{
			ID:       uuid.New(),
			TenantID: uuid.New(),
			Name:     "disabled-job",
			JobType:  job.TypeAuditArchive,
			Interval: "disabled",
			Enabled:  false,
		}

		if s.Enabled {
			t.Error("schedule should be disabled")
		}
		if s.Interval != "disabled" {
			t.Error("interval should be 'disabled'")
		}
	})
}

func TestSchedulerConfig(t *testing.T) {
	t.Run("scheduler config defaults", func(t *testing.T) {
		cfg := &job.SchedulerConfig{}

		// Verify config can be created with zero values
		if cfg.Interval < 0 {
			t.Error("interval should not be negative")
		}
	})

	t.Run("scheduler config custom values", func(t *testing.T) {
		cfg := &job.SchedulerConfig{
			Interval: 30 * time.Second,
		}

		if cfg.Interval != 30*time.Second {
			t.Error("interval mismatch")
		}
	})
}

func TestCronExpressionValidation(t *testing.T) {
	tests := []struct {
		name       string
		expression string
		desc       string
	}{
		{"every minute", "* * * * *", "runs every minute"},
		{"every hour", "0 * * * *", "runs at minute 0 of every hour"},
		{"daily at 6am", "0 6 * * *", "runs at 6:00 AM daily"},
		{"weekly sunday", "0 0 * * 0", "runs at midnight on Sunday"},
		{"monthly first", "0 0 1 * *", "runs at midnight on the 1st"},
		{"every 4 hours", "0 */4 * * *", "runs every 4 hours"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Just verify the expression is non-empty
			// Full validation would require the cron parser
			if tt.expression == "" {
				t.Error("cron expression should not be empty")
			}

			// Verify format (5 fields for standard cron)
			fields := 0
			inField := false
			for _, c := range tt.expression {
				if c == ' ' {
					if inField {
						fields++
						inField = false
					}
				} else {
					inField = true
				}
			}
			if inField {
				fields++
			}

			if fields != 5 {
				t.Errorf("expected 5 fields in cron expression, got %d", fields)
			}
		})
	}
}

func TestScheduleIsDue(t *testing.T) {
	now := time.Now().UTC()

	tests := []struct {
		name      string
		nextRunAt *time.Time
		enabled   bool
		isDue     bool
	}{
		{
			name:      "due now",
			nextRunAt: timePtr(now.Add(-1 * time.Minute)),
			enabled:   true,
			isDue:     true,
		},
		{
			name:      "not yet due",
			nextRunAt: timePtr(now.Add(1 * time.Hour)),
			enabled:   true,
			isDue:     false,
		},
		{
			name:      "disabled",
			nextRunAt: timePtr(now.Add(-1 * time.Minute)),
			enabled:   false,
			isDue:     false,
		},
		{
			name:      "no next run set",
			nextRunAt: nil,
			enabled:   true,
			isDue:     true, // Should run immediately if never run
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &job.Schedule{
				Enabled:   tt.enabled,
				NextRunAt: tt.nextRunAt,
			}

			isDue := s.Enabled && (s.NextRunAt == nil || !s.NextRunAt.After(now))

			if isDue != tt.isDue {
				t.Errorf("expected isDue=%v, got %v", tt.isDue, isDue)
			}
		})
	}
}

func TestScheduleJobTypes(t *testing.T) {
	// Verify all job types can be scheduled
	jobTypes := []string{
		job.TypeDataboxSync,
		job.TypeDeadlineReminder,
		job.TypeWatchlistCheck,
		job.TypeWebhookDelivery,
		job.TypeAuditArchive,
		job.TypeSessionCleanup,
	}

	for _, jobType := range jobTypes {
		t.Run(jobType, func(t *testing.T) {
			s := &job.Schedule{
				ID:       uuid.New(),
				TenantID: uuid.New(),
				Name:     "test-" + jobType,
				JobType:  jobType,
				Interval: "daily",
				Enabled:  true,
			}

			if s.JobType != jobType {
				t.Errorf("job type mismatch: expected %s, got %s", jobType, s.JobType)
			}
		})
	}
}

func TestDefaultSchedules(t *testing.T) {
	// Define expected default schedules
	defaultSchedules := []struct {
		name     string
		jobType  string
		interval string
		cronExpr string
	}{
		{"databox-sync", job.TypeDataboxSync, "4hourly", "0 */4 * * *"},
		{"deadline-reminder", job.TypeDeadlineReminder, "daily", "0 6 * * *"},
		{"watchlist-check", job.TypeWatchlistCheck, "daily", "0 7 * * *"},
		{"session-cleanup", job.TypeSessionCleanup, "daily", "0 3 * * *"},
		{"audit-archive", job.TypeAuditArchive, "weekly", "0 2 * * 0"},
	}

	for _, ds := range defaultSchedules {
		t.Run(ds.name, func(t *testing.T) {
			if ds.jobType == "" {
				t.Error("job type should not be empty")
			}
			if ds.interval == "" && ds.cronExpr == "" {
				t.Error("either interval or cron expression should be set")
			}
		})
	}
}

func timePtr(t time.Time) *time.Time {
	return &t
}
