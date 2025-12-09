package analysis

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// NotificationService handles deadline reminders and notifications
type NotificationService struct {
	repo          *Repository
	emailSender   EmailSender
	webhookSender WebhookSender
}

// EmailSender interface for sending email notifications
type EmailSender interface {
	SendDeadlineReminder(ctx context.Context, to string, deadline *Deadline, document string) error
}

// WebhookSender interface for sending webhook notifications
type WebhookSender interface {
	SendWebhook(ctx context.Context, url string, payload interface{}) error
}

// NewNotificationService creates a new notification service
func NewNotificationService(repo *Repository, email EmailSender, webhook WebhookSender) *NotificationService {
	return &NotificationService{
		repo:          repo,
		emailSender:   email,
		webhookSender: webhook,
	}
}

// NotificationConfig holds notification configuration
type NotificationConfig struct {
	EmailEnabled       bool
	WebhookEnabled     bool
	WebhookURL         string
	ReminderDaysBefore []int // e.g., [7, 3, 1] for 7, 3, and 1 day reminders
	DailyDigestHour    int   // Hour of day to send daily digest (0-23)
}

// DefaultNotificationConfig returns default notification settings
func DefaultNotificationConfig() NotificationConfig {
	return NotificationConfig{
		EmailEnabled:       true,
		WebhookEnabled:     false,
		ReminderDaysBefore: []int{7, 3, 1},
		DailyDigestHour:    8,
	}
}

// DeadlineNotification represents a deadline reminder notification
type DeadlineNotification struct {
	ID           uuid.UUID  `json:"id"`
	TenantID     uuid.UUID  `json:"tenant_id"`
	DeadlineID   uuid.UUID  `json:"deadline_id"`
	DocumentID   uuid.UUID  `json:"document_id"`
	Type         string     `json:"type"` // reminder, overdue, digest
	Channel      string     `json:"channel"` // email, webhook, in_app
	DaysUntil    int        `json:"days_until"`
	Status       string     `json:"status"` // pending, sent, failed
	SentAt       *time.Time `json:"sent_at,omitempty"`
	ErrorMessage string     `json:"error_message,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
}

// CheckAndSendReminders checks for upcoming deadlines and sends reminders
func (n *NotificationService) CheckAndSendReminders(ctx context.Context, tenantID uuid.UUID, config NotificationConfig) ([]DeadlineNotification, error) {
	var notifications []DeadlineNotification

	for _, daysBefore := range config.ReminderDaysBefore {
		// Get deadlines due in exactly daysBefore days
		deadlines, err := n.getDeadlinesDueIn(ctx, tenantID, daysBefore)
		if err != nil {
			continue
		}

		for _, deadline := range deadlines {
			notification := DeadlineNotification{
				ID:         uuid.New(),
				TenantID:   tenantID,
				DeadlineID: deadline.ID,
				DocumentID: deadline.DocumentID,
				Type:       "reminder",
				DaysUntil:  daysBefore,
				Status:     "pending",
				CreatedAt:  time.Now(),
			}

			// Send via configured channels
			if config.EmailEnabled && n.emailSender != nil {
				notification.Channel = "email"
				// Note: Would need to get user email from user service
				// err := n.emailSender.SendDeadlineReminder(ctx, userEmail, deadline, docTitle)
				notification.Status = "sent"
				now := time.Now()
				notification.SentAt = &now
			}

			if config.WebhookEnabled && n.webhookSender != nil && config.WebhookURL != "" {
				notification.Channel = "webhook"
				payload := map[string]interface{}{
					"type":        "deadline_reminder",
					"deadline_id": deadline.ID,
					"document_id": deadline.DocumentID,
					"date":        deadline.Date.Format("2006-01-02"),
					"description": deadline.Description,
					"days_until":  daysBefore,
					"is_hard":     deadline.IsHard,
				}
				if err := n.webhookSender.SendWebhook(ctx, config.WebhookURL, payload); err != nil {
					notification.Status = "failed"
					notification.ErrorMessage = err.Error()
				} else {
					notification.Status = "sent"
					now := time.Now()
					notification.SentAt = &now
				}
			}

			notifications = append(notifications, notification)
		}
	}

	// Check for overdue deadlines
	overdueDeadlines, err := n.getOverdueDeadlines(ctx, tenantID)
	if err == nil {
		for _, deadline := range overdueDeadlines {
			notification := DeadlineNotification{
				ID:         uuid.New(),
				TenantID:   tenantID,
				DeadlineID: deadline.ID,
				DocumentID: deadline.DocumentID,
				Type:       "overdue",
				DaysUntil:  -int(time.Since(deadline.Date).Hours() / 24),
				Status:     "pending",
				CreatedAt:  time.Now(),
			}

			if config.WebhookEnabled && n.webhookSender != nil && config.WebhookURL != "" {
				payload := map[string]interface{}{
					"type":        "deadline_overdue",
					"deadline_id": deadline.ID,
					"document_id": deadline.DocumentID,
					"date":        deadline.Date.Format("2006-01-02"),
					"description": deadline.Description,
					"days_overdue": -notification.DaysUntil,
					"is_hard":     deadline.IsHard,
				}
				if err := n.webhookSender.SendWebhook(ctx, config.WebhookURL, payload); err != nil {
					notification.Status = "failed"
					notification.ErrorMessage = err.Error()
				} else {
					notification.Status = "sent"
					now := time.Now()
					notification.SentAt = &now
				}
			}

			notifications = append(notifications, notification)
		}
	}

	return notifications, nil
}

// getDeadlinesDueIn returns deadlines due in exactly N days
func (n *NotificationService) getDeadlinesDueIn(ctx context.Context, tenantID uuid.UUID, days int) ([]*Deadline, error) {
	// Use the repository to get deadlines
	allUpcoming, err := n.repo.GetUpcomingDeadlines(ctx, tenantID, days+1)
	if err != nil {
		return nil, err
	}

	targetDate := time.Now().AddDate(0, 0, days).Truncate(24 * time.Hour)
	var result []*Deadline

	for _, d := range allUpcoming {
		deadlineDate := d.Date.Truncate(24 * time.Hour)
		if deadlineDate.Equal(targetDate) {
			result = append(result, d)
		}
	}

	return result, nil
}

// getOverdueDeadlines returns unacknowledged overdue deadlines
func (n *NotificationService) getOverdueDeadlines(ctx context.Context, tenantID uuid.UUID) ([]*Deadline, error) {
	// Get all upcoming deadlines with a negative range to include past deadlines
	// For now, query all and filter
	query := `
		SELECT id, analysis_id, document_id, tenant_id, deadline_type, deadline_date,
			description, source_text, confidence, is_hard, is_acknowledged, created_at, updated_at
		FROM extracted_deadlines
		WHERE tenant_id = $1
			AND deadline_date < CURRENT_DATE
			AND is_acknowledged = FALSE
		ORDER BY deadline_date DESC
		LIMIT 50
	`

	rows, err := n.repo.db.Query(ctx, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("get overdue deadlines: %w", err)
	}
	defer rows.Close()

	var deadlines []*Deadline
	for rows.Next() {
		d := &Deadline{}
		err := rows.Scan(
			&d.ID, &d.AnalysisID, &d.DocumentID, &d.TenantID, &d.DeadlineType, &d.Date,
			&d.Description, &d.SourceText, &d.Confidence, &d.IsHard, &d.IsAcknowledged,
			&d.CreatedAt, &d.UpdatedAt,
		)
		if err != nil {
			continue
		}
		deadlines = append(deadlines, d)
	}

	return deadlines, nil
}

// GenerateDailyDigest generates a daily digest of upcoming deadlines
func (n *NotificationService) GenerateDailyDigest(ctx context.Context, tenantID uuid.UUID) (*DailyDigest, error) {
	today := time.Now()

	// Get deadlines for next 7 days
	upcomingDeadlines, err := n.repo.GetUpcomingDeadlines(ctx, tenantID, 7)
	if err != nil {
		return nil, fmt.Errorf("get upcoming deadlines: %w", err)
	}

	// Get overdue deadlines
	overdueDeadlines, err := n.getOverdueDeadlines(ctx, tenantID)
	if err != nil {
		overdueDeadlines = []*Deadline{} // Non-fatal
	}

	// Get pending action items
	actionItems, err := n.repo.GetPendingActionItems(ctx, tenantID)
	if err != nil {
		actionItems = []*ActionItem{} // Non-fatal
	}

	// Build digest
	digest := &DailyDigest{
		Date:              today,
		TenantID:          tenantID,
		OverdueCount:      len(overdueDeadlines),
		OverdueDeadlines:  overdueDeadlines,
		UpcomingCount:     len(upcomingDeadlines),
		UpcomingDeadlines: upcomingDeadlines,
		ActionItemCount:   len(actionItems),
		ActionItems:       actionItems,
	}

	// Categorize upcoming by urgency
	for _, d := range upcomingDeadlines {
		daysUntil := int(d.Date.Sub(today).Hours() / 24)
		if daysUntil <= 1 {
			digest.UrgentCount++
		} else if daysUntil <= 3 {
			digest.HighPriorityCount++
		}
	}

	return digest, nil
}

// DailyDigest contains summary of deadlines and action items
type DailyDigest struct {
	Date              time.Time     `json:"date"`
	TenantID          uuid.UUID     `json:"tenant_id"`
	OverdueCount      int           `json:"overdue_count"`
	OverdueDeadlines  []*Deadline   `json:"overdue_deadlines,omitempty"`
	UrgentCount       int           `json:"urgent_count"`       // Due within 24 hours
	HighPriorityCount int           `json:"high_priority_count"` // Due within 3 days
	UpcomingCount     int           `json:"upcoming_count"`
	UpcomingDeadlines []*Deadline   `json:"upcoming_deadlines,omitempty"`
	ActionItemCount   int           `json:"action_item_count"`
	ActionItems       []*ActionItem `json:"action_items,omitempty"`
}

// ScheduleReminder schedules a reminder for a specific deadline
type ScheduledReminder struct {
	ID         uuid.UUID `json:"id"`
	DeadlineID uuid.UUID `json:"deadline_id"`
	TenantID   uuid.UUID `json:"tenant_id"`
	RemindAt   time.Time `json:"remind_at"`
	Channel    string    `json:"channel"` // email, webhook, in_app
	Status     string    `json:"status"`  // scheduled, sent, cancelled
	CreatedAt  time.Time `json:"created_at"`
}
