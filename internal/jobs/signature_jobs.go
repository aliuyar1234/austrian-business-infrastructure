package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"

	"austrian-business-infrastructure/internal/signature"
)

// SignatureJobHandler handles signature-related background jobs
type SignatureJobHandler struct {
	repo    *signature.Repository
	service *signature.Service
}

// NewSignatureJobHandler creates a new signature job handler
func NewSignatureJobHandler(repo *signature.Repository, service *signature.Service) *SignatureJobHandler {
	return &SignatureJobHandler{
		repo:    repo,
		service: service,
	}
}

// HandleSignatureExpiry handles the signature_expiry job
// This job marks expired signature requests as expired
func (h *SignatureJobHandler) HandleSignatureExpiry(ctx context.Context, payload json.RawMessage) error {
	count, err := h.service.ExpireRequests(ctx)
	if err != nil {
		return fmt.Errorf("failed to expire requests: %w", err)
	}

	if count > 0 {
		log.Printf("Expired %d signature requests", count)
	}

	return nil
}

// HandleSignatureReminder handles the signature_reminder job
// This job sends reminders to signers who haven't signed yet
func (h *SignatureJobHandler) HandleSignatureReminder(ctx context.Context, payload json.RawMessage) error {
	var input struct {
		ReminderDays int `json:"reminder_days"`
	}
	if err := json.Unmarshal(payload, &input); err != nil {
		input.ReminderDays = 7 // Default to 7 days before expiry
	}

	// Find requests expiring soon with pending signers
	requests, err := h.findRequestsNeedingReminders(ctx, input.ReminderDays)
	if err != nil {
		return fmt.Errorf("failed to find requests: %w", err)
	}

	remindersS_ent := 0
	for _, req := range requests {
		for _, signer := range req.Signers {
			if signer.Status != signature.SignerStatusNotified {
				continue
			}

			// Check if we already sent a reminder recently (e.g., within last 24 hours)
			if signer.LastReminderAt != nil {
				if time.Since(*signer.LastReminderAt) < 24*time.Hour {
					continue
				}
			}

			if err := h.service.SendReminder(ctx, signer.ID); err != nil {
				log.Printf("Failed to send reminder to %s: %v", signer.Email, err)
				continue
			}

			remindersS_ent++
		}
	}

	if remindersS_ent > 0 {
		log.Printf("Sent %d signature reminders", remindersS_ent)
	}

	return nil
}

// findRequestsNeedingReminders finds requests that need reminders
func (h *SignatureJobHandler) findRequestsNeedingReminders(ctx context.Context, daysBeforeExpiry int) ([]*signature.SignatureRequest, error) {
	// Find pending requests expiring within the specified days
	cutoff := time.Now().AddDate(0, 0, daysBeforeExpiry)

	// This would be a custom query in the repository
	// For now, use the existing methods
	allRequests, _, err := h.repo.ListRequestsByTenant(ctx, uuid.Nil, nil, 100, 0)
	if err != nil {
		return nil, err
	}

	var needsReminder []*signature.SignatureRequest
	for _, req := range allRequests {
		if req.Status != signature.RequestStatusPending && req.Status != signature.RequestStatusInProgress {
			continue
		}
		if req.ExpiresAt.After(cutoff) {
			continue
		}

		// Load signers
		fullReq, err := h.repo.GetRequestWithSigners(ctx, req.ID)
		if err != nil {
			continue
		}

		// Check if any signers need reminders
		hasNotified := false
		for _, signer := range fullReq.Signers {
			if signer.Status == signature.SignerStatusNotified {
				hasNotified = true
				break
			}
		}

		if hasNotified {
			needsReminder = append(needsReminder, fullReq)
		}
	}

	return needsReminder, nil
}

// SignatureExpiryJobPayload is the payload for the expiry job
type SignatureExpiryJobPayload struct{}

// SignatureReminderJobPayload is the payload for the reminder job
type SignatureReminderJobPayload struct {
	ReminderDays int `json:"reminder_days"`
}

// RegisterSignatureJobs registers the signature jobs with the job scheduler
func RegisterSignatureJobs(scheduler JobScheduler, handler *SignatureJobHandler) {
	// Daily expiry check at 00:30 UTC
	scheduler.RegisterJob("signature_expiry", handler.HandleSignatureExpiry)
	scheduler.Schedule("signature_expiry", "0 30 0 * * *", SignatureExpiryJobPayload{})

	// Daily reminder at 09:00 UTC
	scheduler.RegisterJob("signature_reminder", handler.HandleSignatureReminder)
	scheduler.Schedule("signature_reminder", "0 0 9 * * *", SignatureReminderJobPayload{ReminderDays: 7})
}

// JobScheduler interface for job scheduling
type JobScheduler interface {
	RegisterJob(name string, handler func(ctx context.Context, payload json.RawMessage) error)
	Schedule(name string, cron string, payload interface{})
}

// SignatureUsageReportJob generates monthly signature usage reports
type SignatureUsageReportJob struct {
	repo *signature.Repository
}

// NewSignatureUsageReportJob creates a new usage report job
func NewSignatureUsageReportJob(repo *signature.Repository) *SignatureUsageReportJob {
	return &SignatureUsageReportJob{repo: repo}
}

// HandleMonthlyReport handles the monthly usage report generation
func (j *SignatureUsageReportJob) HandleMonthlyReport(ctx context.Context, payload json.RawMessage) error {
	var input struct {
		Year  int `json:"year"`
		Month int `json:"month"`
	}
	if err := json.Unmarshal(payload, &input); err != nil {
		// Default to previous month
		now := time.Now()
		input.Year = now.Year()
		input.Month = int(now.Month()) - 1
		if input.Month < 1 {
			input.Month = 12
			input.Year--
		}
	}

	// TODO: Generate and store/email monthly report
	// This would aggregate usage per tenant and generate a report

	log.Printf("Generated signature usage report for %d-%02d", input.Year, input.Month)
	return nil
}
