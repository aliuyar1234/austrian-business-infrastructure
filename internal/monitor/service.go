package monitor

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"austrian-business-infrastructure/internal/foerderung"
)

// Service provides monitor business logic
type Service struct {
	repo     *Repository
	notifRepo *NotificationRepository
}

// NewService creates a new monitor service
func NewService(repo *Repository, notifRepo *NotificationRepository) *Service {
	return &Service{
		repo:      repo,
		notifRepo: notifRepo,
	}
}

// CreateInput contains input for creating a monitor
type CreateInput struct {
	TenantID          uuid.UUID
	ProfileID         uuid.UUID
	MinScoreThreshold int
	NotificationEmail bool
	NotificationPortal bool
	DigestMode        string
}

// UpdateInput contains input for updating a monitor
type UpdateInput struct {
	IsActive          *bool
	MinScoreThreshold *int
	NotificationEmail *bool
	NotificationPortal *bool
	DigestMode        *string
}

// Create creates a new monitor
func (s *Service) Create(ctx context.Context, input *CreateInput) (*foerderung.ProfilMonitor, error) {
	// Default threshold
	threshold := input.MinScoreThreshold
	if threshold <= 0 {
		threshold = 70 // Default 70%
	}

	// Default digest mode
	digestMode := input.DigestMode
	if digestMode == "" {
		digestMode = "immediate"
	}

	monitor := &foerderung.ProfilMonitor{
		TenantID:          input.TenantID,
		ProfileID:         input.ProfileID,
		IsActive:          true,
		MinScoreThreshold: threshold,
		NotificationEmail: input.NotificationEmail,
		NotificationPortal: input.NotificationPortal,
		DigestMode:        digestMode,
	}

	if err := s.repo.Create(ctx, monitor); err != nil {
		return nil, err
	}

	return monitor, nil
}

// GetByID retrieves a monitor by ID
func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*foerderung.ProfilMonitor, error) {
	return s.repo.GetByID(ctx, id)
}

// GetByIDAndTenant retrieves a monitor ensuring tenant access
func (s *Service) GetByIDAndTenant(ctx context.Context, id, tenantID uuid.UUID) (*foerderung.ProfilMonitor, error) {
	return s.repo.GetByIDAndTenant(ctx, id, tenantID)
}

// ListByTenant lists all monitors for a tenant
func (s *Service) ListByTenant(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]*foerderung.ProfilMonitor, int, error) {
	return s.repo.ListByTenant(ctx, tenantID, limit, offset)
}

// Update updates a monitor
func (s *Service) Update(ctx context.Context, id, tenantID uuid.UUID, input *UpdateInput) (*foerderung.ProfilMonitor, error) {
	monitor, err := s.repo.GetByIDAndTenant(ctx, id, tenantID)
	if err != nil {
		return nil, err
	}

	if input.IsActive != nil {
		monitor.IsActive = *input.IsActive
	}
	if input.MinScoreThreshold != nil {
		monitor.MinScoreThreshold = *input.MinScoreThreshold
	}
	if input.NotificationEmail != nil {
		monitor.NotificationEmail = *input.NotificationEmail
	}
	if input.NotificationPortal != nil {
		monitor.NotificationPortal = *input.NotificationPortal
	}
	if input.DigestMode != nil {
		monitor.DigestMode = *input.DigestMode
	}

	if err := s.repo.Update(ctx, monitor); err != nil {
		return nil, err
	}

	return monitor, nil
}

// Delete deletes a monitor
func (s *Service) Delete(ctx context.Context, id, tenantID uuid.UUID) error {
	// Verify ownership
	_, err := s.repo.GetByIDAndTenant(ctx, id, tenantID)
	if err != nil {
		return err
	}
	return s.repo.Delete(ctx, id)
}

// RecordMatch records a new match notification
func (s *Service) RecordMatch(ctx context.Context, monitorID, foerderungID uuid.UUID, score int, summary string) error {
	notification := &foerderung.MonitorNotification{
		MonitorID:    monitorID,
		FoerderungID: foerderungID,
		Score:        score,
		MatchSummary: &summary,
	}

	if err := s.notifRepo.Create(ctx, notification); err != nil {
		return err
	}

	// Update monitor stats
	monitor, err := s.repo.GetByID(ctx, monitorID)
	if err != nil {
		return err
	}

	monitor.MatchesFound++
	now := time.Now()
	monitor.LastCheckAt = &now

	return s.repo.Update(ctx, monitor)
}

// GetNotifications retrieves notifications for a monitor
func (s *Service) GetNotifications(ctx context.Context, monitorID uuid.UUID, limit, offset int) ([]*foerderung.MonitorNotification, error) {
	return s.notifRepo.ListByMonitor(ctx, monitorID, limit, offset)
}

// MarkNotificationViewed marks a notification as viewed
func (s *Service) MarkNotificationViewed(ctx context.Context, notificationID uuid.UUID) error {
	return s.notifRepo.MarkAsViewed(ctx, notificationID)
}

// DismissNotification dismisses a notification
func (s *Service) DismissNotification(ctx context.Context, notificationID uuid.UUID) error {
	return s.notifRepo.Dismiss(ctx, notificationID)
}

// ValidateDigestMode validates the digest mode value
func ValidateDigestMode(mode string) error {
	switch mode {
	case "immediate", "daily", "weekly":
		return nil
	default:
		return fmt.Errorf("invalid digest mode: %s (must be immediate, daily, or weekly)", mode)
	}
}
