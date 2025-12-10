package antrag

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"austrian-business-infrastructure/internal/foerderung"
)

// Service provides application business logic
type Service struct {
	repo *Repository
}

// NewService creates a new application service
func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// CreateInput contains input for creating an application
type CreateInput struct {
	TenantID          uuid.UUID
	ProfileID         uuid.UUID
	FoerderungID      uuid.UUID
	InternalReference *string
	RequestedAmount   *int
	Notes             *string
	CreatedBy         *uuid.UUID
}

// UpdateInput contains input for updating an application
type UpdateInput struct {
	Status            *string
	InternalReference *string
	RequestedAmount   *int
	ApprovedAmount    *int
	DecisionNotes     *string
	Notes             *string
}

// Create creates a new application
func (s *Service) Create(ctx context.Context, input *CreateInput) (*foerderung.FoerderungsAntrag, error) {
	// Create timeline entry
	timeline := []foerderung.TimelineEntry{
		{
			Date:        time.Now(),
			Status:      foerderung.AntragStatusPlanned,
			Description: "Antrag erstellt",
			CreatedBy:   input.CreatedBy,
		},
	}

	antrag := &foerderung.FoerderungsAntrag{
		TenantID:          input.TenantID,
		ProfileID:         input.ProfileID,
		FoerderungID:      input.FoerderungID,
		Status:            foerderung.AntragStatusPlanned,
		InternalReference: input.InternalReference,
		RequestedAmount:   input.RequestedAmount,
		Notes:             input.Notes,
		Timeline:          timeline,
		CreatedBy:         input.CreatedBy,
	}

	if err := s.repo.Create(ctx, antrag); err != nil {
		return nil, err
	}

	return antrag, nil
}

// GetByID retrieves an application by ID
func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*foerderung.FoerderungsAntrag, error) {
	return s.repo.GetByID(ctx, id)
}

// GetByIDAndTenant retrieves an application ensuring tenant access
func (s *Service) GetByIDAndTenant(ctx context.Context, id, tenantID uuid.UUID) (*foerderung.FoerderungsAntrag, error) {
	return s.repo.GetByIDAndTenant(ctx, id, tenantID)
}

// List lists applications with filters
func (s *Service) List(ctx context.Context, filter ListFilter) ([]*foerderung.FoerderungsAntrag, int, error) {
	return s.repo.List(ctx, filter)
}

// Update updates an application
func (s *Service) Update(ctx context.Context, id, tenantID uuid.UUID, input *UpdateInput, userID *uuid.UUID) (*foerderung.FoerderungsAntrag, error) {
	antrag, err := s.repo.GetByIDAndTenant(ctx, id, tenantID)
	if err != nil {
		return nil, err
	}

	if input.InternalReference != nil {
		antrag.InternalReference = input.InternalReference
	}
	if input.RequestedAmount != nil {
		antrag.RequestedAmount = input.RequestedAmount
	}
	if input.ApprovedAmount != nil {
		antrag.ApprovedAmount = input.ApprovedAmount
	}
	if input.DecisionNotes != nil {
		antrag.DecisionNotes = input.DecisionNotes
	}
	if input.Notes != nil {
		antrag.Notes = input.Notes
	}

	if err := s.repo.Update(ctx, antrag); err != nil {
		return nil, err
	}

	return antrag, nil
}

// UpdateStatus updates the status of an application with timeline entry
func (s *Service) UpdateStatus(ctx context.Context, id, tenantID uuid.UUID, newStatus, description string, userID *uuid.UUID) (*foerderung.FoerderungsAntrag, error) {
	antrag, err := s.repo.GetByIDAndTenant(ctx, id, tenantID)
	if err != nil {
		return nil, err
	}

	// Validate status transition
	if err := s.validateStatusTransition(antrag.Status, newStatus); err != nil {
		return nil, err
	}

	// Update status
	antrag.Status = newStatus

	// Add timeline entry
	entry := foerderung.TimelineEntry{
		Date:        time.Now(),
		Status:      newStatus,
		Description: description,
		CreatedBy:   userID,
	}
	antrag.Timeline = append(antrag.Timeline, entry)

	// Set submitted_at for submission
	if newStatus == foerderung.AntragStatusSubmitted && antrag.SubmittedAt == nil {
		now := time.Now()
		antrag.SubmittedAt = &now
	}

	// Set decision_date for approval/rejection
	if newStatus == foerderung.AntragStatusApproved || newStatus == foerderung.AntragStatusRejected {
		now := time.Now()
		antrag.DecisionDate = &now
	}

	if err := s.repo.Update(ctx, antrag); err != nil {
		return nil, err
	}

	return antrag, nil
}

// validateStatusTransition validates if a status transition is allowed
func (s *Service) validateStatusTransition(currentStatus, newStatus string) error {
	validTransitions := map[string][]string{
		foerderung.AntragStatusPlanned: {
			foerderung.AntragStatusDrafting,
			foerderung.AntragStatusWithdrawn,
		},
		foerderung.AntragStatusDrafting: {
			foerderung.AntragStatusPlanned,
			foerderung.AntragStatusSubmitted,
			foerderung.AntragStatusWithdrawn,
		},
		foerderung.AntragStatusSubmitted: {
			foerderung.AntragStatusInReview,
			foerderung.AntragStatusWithdrawn,
		},
		foerderung.AntragStatusInReview: {
			foerderung.AntragStatusApproved,
			foerderung.AntragStatusRejected,
			foerderung.AntragStatusWithdrawn,
		},
		foerderung.AntragStatusApproved:  {}, // Terminal state
		foerderung.AntragStatusRejected:  {}, // Terminal state
		foerderung.AntragStatusWithdrawn: {}, // Terminal state
	}

	allowed, ok := validTransitions[currentStatus]
	if !ok {
		return fmt.Errorf("unbekannter Status: %s", currentStatus)
	}

	for _, s := range allowed {
		if s == newStatus {
			return nil
		}
	}

	return fmt.Errorf("ungültiger Statusübergang von %s nach %s", currentStatus, newStatus)
}

// Delete deletes an application
func (s *Service) Delete(ctx context.Context, id, tenantID uuid.UUID) error {
	// Verify ownership
	_, err := s.repo.GetByIDAndTenant(ctx, id, tenantID)
	if err != nil {
		return err
	}
	return s.repo.Delete(ctx, id)
}

// GetStats retrieves application statistics
func (s *Service) GetStats(ctx context.Context, tenantID uuid.UUID) (*AntragStats, error) {
	return s.repo.GetStats(ctx, tenantID)
}

// AddAttachment adds an attachment to an application
func (s *Service) AddAttachment(ctx context.Context, id, tenantID uuid.UUID, attachment foerderung.Attachment) (*foerderung.FoerderungsAntrag, error) {
	antrag, err := s.repo.GetByIDAndTenant(ctx, id, tenantID)
	if err != nil {
		return nil, err
	}

	attachment.UploadedAt = time.Now()
	antrag.Attachments = append(antrag.Attachments, attachment)

	if err := s.repo.Update(ctx, antrag); err != nil {
		return nil, err
	}

	return antrag, nil
}

// RemoveAttachment removes an attachment from an application
func (s *Service) RemoveAttachment(ctx context.Context, id, tenantID uuid.UUID, attachmentName string) (*foerderung.FoerderungsAntrag, error) {
	antrag, err := s.repo.GetByIDAndTenant(ctx, id, tenantID)
	if err != nil {
		return nil, err
	}

	newAttachments := make([]foerderung.Attachment, 0)
	for _, a := range antrag.Attachments {
		if a.Name != attachmentName {
			newAttachments = append(newAttachments, a)
		}
	}
	antrag.Attachments = newAttachments

	if err := s.repo.Update(ctx, antrag); err != nil {
		return nil, err
	}

	return antrag, nil
}
