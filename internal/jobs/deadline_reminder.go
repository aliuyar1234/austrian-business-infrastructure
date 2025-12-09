package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/austrian-business-infrastructure/fo/internal/analysis"
	"github.com/austrian-business-infrastructure/fo/internal/document"
	"github.com/austrian-business-infrastructure/fo/internal/email"
	"github.com/austrian-business-infrastructure/fo/internal/job"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DeadlineReminderPayload contains the job payload for deadline reminders
type DeadlineReminderPayload struct {
	TenantID     uuid.UUID `json:"tenant_id"`
	ReminderDays []int     `json:"reminder_days,omitempty"` // e.g., [7, 3, 1]
}

// DeadlineReminderResult contains the result of a deadline reminder job
type DeadlineReminderResult struct {
	DocumentsChecked     int      `json:"documents_checked"`
	RemindersSent        int      `json:"reminders_sent"`
	AnalysisRemindersSent int     `json:"analysis_reminders_sent"` // From AI-extracted deadlines
	OverdueAlerts        int      `json:"overdue_alerts"`
	Errors               []string `json:"errors,omitempty"`
	Duration             string   `json:"duration"`
}

// DeadlineReminderHandler handles deadline reminder jobs
type DeadlineReminderHandler struct {
	db                  *pgxpool.Pool
	docRepo             *document.Repository
	emailSvc            email.Service
	notificationService *analysis.NotificationService
	logger              *slog.Logger
	appURL              string

	// Default reminder days if not specified in payload
	defaultReminderDays []int
}

// DeadlineReminderHandlerConfig holds handler configuration
type DeadlineReminderHandlerConfig struct {
	Logger              *slog.Logger
	AppURL              string
	ReminderDays        []int
	NotificationService *analysis.NotificationService
}

// NewDeadlineReminderHandler creates a new deadline reminder handler
func NewDeadlineReminderHandler(
	db *pgxpool.Pool,
	docRepo *document.Repository,
	emailSvc email.Service,
	cfg *DeadlineReminderHandlerConfig,
) *DeadlineReminderHandler {
	logger := slog.Default()
	appURL := "http://localhost:3000"
	reminderDays := []int{7, 3, 1}
	var notifSvc *analysis.NotificationService

	if cfg != nil {
		if cfg.Logger != nil {
			logger = cfg.Logger
		}
		if cfg.AppURL != "" {
			appURL = cfg.AppURL
		}
		if len(cfg.ReminderDays) > 0 {
			reminderDays = cfg.ReminderDays
		}
		if cfg.NotificationService != nil {
			notifSvc = cfg.NotificationService
		}
	}

	return &DeadlineReminderHandler{
		db:                  db,
		docRepo:             docRepo,
		emailSvc:            emailSvc,
		notificationService: notifSvc,
		logger:              logger,
		appURL:              appURL,
		defaultReminderDays: reminderDays,
	}
}

// Handle processes a deadline reminder job
func (h *DeadlineReminderHandler) Handle(ctx context.Context, j *job.Job) (json.RawMessage, error) {
	startTime := time.Now()

	// Parse payload
	var payload DeadlineReminderPayload
	if err := j.PayloadTo(&payload); err != nil {
		return nil, fmt.Errorf("parse payload: %w", err)
	}

	reminderDays := payload.ReminderDays
	if len(reminderDays) == 0 {
		reminderDays = h.defaultReminderDays
	}

	logger := h.logger.With(
		"job_id", j.ID,
		"tenant_id", payload.TenantID,
		"reminder_days", reminderDays,
	)

	logger.Info("processing deadline reminders")

	result := &DeadlineReminderResult{}

	// Get documents with upcoming deadlines
	now := time.Now()
	for _, days := range reminderDays {
		deadline := now.AddDate(0, 0, days)
		deadlineStart := time.Date(deadline.Year(), deadline.Month(), deadline.Day(), 0, 0, 0, 0, time.UTC)
		deadlineEnd := deadlineStart.Add(24 * time.Hour)

		docs, err := h.getDocumentsWithDeadline(ctx, payload.TenantID, deadlineStart, deadlineEnd, days)
		if err != nil {
			logger.Error("failed to get documents for deadline", "days", days, "error", err)
			result.Errors = append(result.Errors, fmt.Sprintf("day %d: %v", days, err))
			continue
		}

		result.DocumentsChecked += len(docs)

		for _, doc := range docs {
			if err := h.sendReminder(ctx, doc, days); err != nil {
				logger.Error("failed to send reminder",
					"document_id", doc.ID,
					"days", days,
					"error", err)
				result.Errors = append(result.Errors, fmt.Sprintf("doc %s: %v", doc.ID, err))
				continue
			}

			// Mark reminder as sent
			if err := h.markReminderSent(ctx, doc.ID, days); err != nil {
				logger.Error("failed to mark reminder sent", "document_id", doc.ID, "error", err)
			}

			result.RemindersSent++
		}
	}

	// Also process AI-extracted deadlines if notification service is available
	if h.notificationService != nil {
		notifConfig := analysis.NotificationConfig{
			EmailEnabled:       h.emailSvc != nil,
			WebhookEnabled:     false,
			ReminderDaysBefore: reminderDays,
		}

		notifications, err := h.notificationService.CheckAndSendReminders(ctx, payload.TenantID, notifConfig)
		if err != nil {
			logger.Error("failed to check analysis-based deadlines", "error", err)
			result.Errors = append(result.Errors, fmt.Sprintf("analysis deadlines: %v", err))
		} else {
			for _, n := range notifications {
				switch n.Type {
				case "reminder":
					if n.Status == "sent" {
						result.AnalysisRemindersSent++
					}
				case "overdue":
					if n.Status == "sent" {
						result.OverdueAlerts++
					}
				}
			}
		}
	}

	result.Duration = time.Since(startTime).String()

	logger.Info("deadline reminders completed",
		"documents_checked", result.DocumentsChecked,
		"reminders_sent", result.RemindersSent,
		"analysis_reminders_sent", result.AnalysisRemindersSent,
		"overdue_alerts", result.OverdueAlerts,
		"duration", result.Duration)

	resultJSON, _ := json.Marshal(result)
	return resultJSON, nil
}

// getDocumentsWithDeadline finds documents with deadlines in the specified range
// that haven't had a reminder sent for this interval
func (h *DeadlineReminderHandler) getDocumentsWithDeadline(
	ctx context.Context,
	tenantID uuid.UUID,
	deadlineStart, deadlineEnd time.Time,
	reminderDays int,
) ([]*document.Document, error) {
	// Determine which reminder field to check based on days
	var reminderField string
	switch reminderDays {
	case 7:
		reminderField = "reminder_7d_sent_at"
	case 3:
		reminderField = "reminder_3d_sent_at"
	case 1:
		reminderField = "reminder_1d_sent_at"
	default:
		return nil, fmt.Errorf("unsupported reminder days: %d", reminderDays)
	}

	query := fmt.Sprintf(`
		SELECT d.id, d.account_id, d.external_id, d.type, d.title, d.sender,
		       d.received_at, d.content_hash, d.storage_path, d.file_size, d.mime_type,
		       d.status, d.archived_at, d.retention_until, d.deadline, d.metadata,
		       d.created_at, d.updated_at
		FROM documents d
		JOIN accounts a ON d.account_id = a.id
		WHERE a.tenant_id = $1
		  AND d.deadline >= $2 AND d.deadline < $3
		  AND d.status != 'archived'
		  AND d.%s IS NULL
		ORDER BY d.deadline ASC
	`, reminderField)

	rows, err := h.db.Query(ctx, query, tenantID, deadlineStart, deadlineEnd)
	if err != nil {
		return nil, fmt.Errorf("query documents: %w", err)
	}
	defer rows.Close()

	var docs []*document.Document
	for rows.Next() {
		doc := &document.Document{}
		err := rows.Scan(
			&doc.ID, &doc.AccountID, &doc.ExternalID, &doc.Type, &doc.Title, &doc.Sender,
			&doc.ReceivedAt, &doc.ContentHash, &doc.StoragePath, &doc.FileSize, &doc.MimeType,
			&doc.Status, &doc.ArchivedAt, &doc.RetentionUntil, &doc.Deadline, &doc.Metadata,
			&doc.CreatedAt, &doc.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan document: %w", err)
		}
		docs = append(docs, doc)
	}

	return docs, rows.Err()
}

// sendReminder sends a deadline reminder email
func (h *DeadlineReminderHandler) sendReminder(ctx context.Context, doc *document.Document, daysRemaining int) error {
	// TODO: Get user emails for the tenant/account
	// For now, this is a placeholder

	subject := fmt.Sprintf("Frist-Erinnerung: %s (%d Tage)", doc.Title, daysRemaining)
	body := h.buildReminderBody(doc, daysRemaining)

	h.logger.Info("sending deadline reminder",
		"document_id", doc.ID,
		"title", doc.Title,
		"days_remaining", daysRemaining,
		"deadline", doc.Deadline)

	// Send via email service
	if h.emailSvc != nil {
		// TODO: Get actual recipient email
		// return h.emailSvc.Send(ctx, recipientEmail, subject, body)
		_ = subject
		_ = body
	}

	return nil
}

// buildReminderBody creates the reminder email body
func (h *DeadlineReminderHandler) buildReminderBody(doc *document.Document, daysRemaining int) string {
	urgency := "Erinnerung"
	if daysRemaining == 1 {
		urgency = "DRINGEND"
	} else if daysRemaining == 3 {
		urgency = "Wichtig"
	}

	deadlineStr := ""
	if doc.Deadline != nil {
		deadlineStr = doc.Deadline.Format("02.01.2006")
	}

	return fmt.Sprintf(`%s: Frist in %d Tag(en)

Dokument: %s
Typ: %s
Absender: %s
Frist: %s

Bitte prüfen Sie das Dokument und erledigen Sie die erforderlichen Aktionen rechtzeitig.

Zum Dokument: %s/documents/%s

--
Austrian Business Platform
`, urgency, daysRemaining, doc.Title, doc.Type, doc.Sender, deadlineStr, h.appURL, doc.ID)
}

// markReminderSent marks a reminder as sent for a document
func (h *DeadlineReminderHandler) markReminderSent(ctx context.Context, docID uuid.UUID, days int) error {
	var field string
	switch days {
	case 7:
		field = "reminder_7d_sent_at"
	case 3:
		field = "reminder_3d_sent_at"
	case 1:
		field = "reminder_1d_sent_at"
	default:
		return fmt.Errorf("unsupported reminder days: %d", days)
	}

	query := fmt.Sprintf(`UPDATE documents SET %s = NOW(), updated_at = NOW() WHERE id = $1`, field)
	_, err := h.db.Exec(ctx, query, docID)
	return err
}

// CreateDefaultSchedule creates the default deadline reminder schedule
func CreateDeadlineReminderSchedule(ctx context.Context, scheduler *job.Scheduler, tenantID uuid.UUID) error {
	schedule := &job.Schedule{
		TenantID:       tenantID,
		Name:           "deadline-reminder",
		JobType:        job.TypeDeadlineReminder,
		CronExpression: "0 6 * * *", // Every day at 6:00 AM
		Enabled:        true,
		Timezone:       "Europe/Vienna",
	}
	schedule.JobPayload, _ = json.Marshal(DeadlineReminderPayload{
		TenantID:     tenantID,
		ReminderDays: []int{7, 3, 1},
	})

	return scheduler.CreateSchedule(ctx, schedule)
}
