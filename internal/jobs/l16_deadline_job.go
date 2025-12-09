package jobs

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/austrian-business-infrastructure/fo/internal/elda"
)

// L16DeadlineJob monitors L16 deadlines and sends warnings
type L16DeadlineJob struct {
	db       *pgxpool.Pool
	logger   *slog.Logger
	notifier DeadlineNotifier
}

// DeadlineNotifier sends deadline notifications
type DeadlineNotifier interface {
	SendL16Warning(ctx context.Context, accountID uuid.UUID, year int, daysRemaining int) error
}

// NewL16DeadlineJob creates a new L16 deadline monitoring job
func NewL16DeadlineJob(db *pgxpool.Pool, logger *slog.Logger, notifier DeadlineNotifier) *L16DeadlineJob {
	return &L16DeadlineJob{
		db:       db,
		logger:   logger,
		notifier: notifier,
	}
}

// Run executes the deadline monitoring job
func (j *L16DeadlineJob) Run(ctx context.Context) error {
	j.logger.Info("starting L16 deadline monitoring job")

	// Get all ELDA accounts
	accounts, err := j.getActiveELDAAccounts(ctx)
	if err != nil {
		return fmt.Errorf("failed to get ELDA accounts: %w", err)
	}

	currentYear := time.Now().Year()
	previousYear := currentYear - 1

	// L16 is submitted in February for the previous year
	// So in January 2025, deadline for 2024 L16 is Feb 28/29 2025
	deadline := elda.GetL16Deadline(previousYear)
	daysRemaining := elda.DaysUntilL16Deadline(previousYear)

	j.logger.Info("checking L16 deadline",
		"for_year", previousYear,
		"deadline", deadline.Format("2006-01-02"),
		"days_remaining", daysRemaining)

	// Warning thresholds: 30 days, 14 days, 7 days, 3 days
	shouldWarn := daysRemaining == 30 || daysRemaining == 14 || daysRemaining == 7 || daysRemaining <= 3

	if !shouldWarn {
		j.logger.Info("no deadline warning needed", "days_remaining", daysRemaining)
		return nil
	}

	for _, account := range accounts {
		// Check if this account has pending L16 for previous year
		pending, err := j.hasPendingL16(ctx, account.ID, previousYear)
		if err != nil {
			j.logger.Error("failed to check pending L16",
				"account_id", account.ID,
				"error", err)
			continue
		}

		if pending {
			j.logger.Info("sending L16 deadline warning",
				"account_id", account.ID,
				"year", previousYear,
				"days_remaining", daysRemaining)

			if j.notifier != nil {
				if err := j.notifier.SendL16Warning(ctx, account.ID, previousYear, daysRemaining); err != nil {
					j.logger.Error("failed to send warning notification",
						"account_id", account.ID,
						"error", err)
				}
			}
		}
	}

	j.logger.Info("L16 deadline monitoring job completed")
	return nil
}

// ELDAAccountInfo contains basic ELDA account info
type ELDAAccountInfo struct {
	ID   uuid.UUID
	Name string
}

// getActiveELDAAccounts retrieves all active ELDA accounts
func (j *L16DeadlineJob) getActiveELDAAccounts(ctx context.Context) ([]ELDAAccountInfo, error) {
	query := `
		SELECT id, name
		FROM elda_accounts
		WHERE active = true
	`

	rows, err := j.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query ELDA accounts: %w", err)
	}
	defer rows.Close()

	var accounts []ELDAAccountInfo
	for rows.Next() {
		var acc ELDAAccountInfo
		if err := rows.Scan(&acc.ID, &acc.Name); err != nil {
			return nil, fmt.Errorf("scan account: %w", err)
		}
		accounts = append(accounts, acc)
	}

	return accounts, nil
}

// hasPendingL16 checks if an account has pending L16 for a year
func (j *L16DeadlineJob) hasPendingL16(ctx context.Context, accountID uuid.UUID, year int) (bool, error) {
	// Check if there are employees without submitted L16 for this year
	query := `
		SELECT EXISTS (
			SELECT 1 FROM lohnzettel
			WHERE elda_account_id = $1
			  AND year = $2
			  AND status NOT IN ('submitted', 'accepted')
		)
	`

	var pending bool
	err := j.db.QueryRow(ctx, query, accountID, year).Scan(&pending)
	if err != nil {
		return false, fmt.Errorf("check pending L16: %w", err)
	}

	return pending, nil
}

// L16DeadlineStatus contains status for deadline monitoring
type L16DeadlineStatus struct {
	Year            int       `json:"year"`
	Deadline        time.Time `json:"deadline"`
	DaysRemaining   int       `json:"days_remaining"`
	IsOverdue       bool      `json:"is_overdue"`
	TotalLohnzettel int       `json:"total_lohnzettel"`
	PendingCount    int       `json:"pending_count"`
	SubmittedCount  int       `json:"submitted_count"`
}

// GetDeadlineStatus returns the deadline status for an account
func (j *L16DeadlineJob) GetDeadlineStatus(ctx context.Context, accountID uuid.UUID, year int) (*L16DeadlineStatus, error) {
	deadline := elda.GetL16Deadline(year)
	daysRemaining := elda.DaysUntilL16Deadline(year)

	query := `
		SELECT
			COUNT(*) as total,
			COUNT(*) FILTER (WHERE status IN ('submitted', 'accepted')) as submitted,
			COUNT(*) FILTER (WHERE status NOT IN ('submitted', 'accepted')) as pending
		FROM lohnzettel
		WHERE elda_account_id = $1 AND year = $2
	`

	var total, submitted, pending int
	err := j.db.QueryRow(ctx, query, accountID, year).Scan(&total, &submitted, &pending)
	if err != nil {
		return nil, fmt.Errorf("get deadline status: %w", err)
	}

	return &L16DeadlineStatus{
		Year:            year,
		Deadline:        deadline,
		DaysRemaining:   daysRemaining,
		IsOverdue:       time.Now().After(deadline),
		TotalLohnzettel: total,
		PendingCount:    pending,
		SubmittedCount:  submitted,
	}, nil
}

// GetAllDeadlineStatuses returns deadline statuses for all years
func (j *L16DeadlineJob) GetAllDeadlineStatuses(ctx context.Context, accountID uuid.UUID) ([]L16DeadlineStatus, error) {
	// Get all years that have lohnzettel for this account
	query := `
		SELECT DISTINCT year
		FROM lohnzettel
		WHERE elda_account_id = $1
		ORDER BY year DESC
	`

	rows, err := j.db.Query(ctx, query, accountID)
	if err != nil {
		return nil, fmt.Errorf("get years: %w", err)
	}
	defer rows.Close()

	var statuses []L16DeadlineStatus
	for rows.Next() {
		var year int
		if err := rows.Scan(&year); err != nil {
			return nil, fmt.Errorf("scan year: %w", err)
		}

		status, err := j.GetDeadlineStatus(ctx, accountID, year)
		if err != nil {
			continue // Skip errors for individual years
		}

		statuses = append(statuses, *status)
	}

	return statuses, nil
}
