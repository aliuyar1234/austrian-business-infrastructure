package task

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// CreateRequest contains data for creating a task
type CreateRequest struct {
	TenantID    uuid.UUID  `json:"tenant_id"`
	ClientID    uuid.UUID  `json:"client_id"`
	CreatedBy   uuid.UUID  `json:"created_by"`
	Title       string     `json:"title"`
	Description *string    `json:"description,omitempty"`
	Priority    Priority   `json:"priority"`
	DueDate     *time.Time `json:"due_date,omitempty"`
	DocumentID  *uuid.UUID `json:"document_id,omitempty"`
	UploadID    *uuid.UUID `json:"upload_id,omitempty"`
	ApprovalID  *uuid.UUID `json:"approval_id,omitempty"`
}

// UpdateRequest contains data for updating a task
type UpdateRequest struct {
	Title       *string    `json:"title,omitempty"`
	Description *string    `json:"description,omitempty"`
	Priority    *Priority  `json:"priority,omitempty"`
	DueDate     *time.Time `json:"due_date,omitempty"`
	DocumentID  *uuid.UUID `json:"document_id,omitempty"`
	UploadID    *uuid.UUID `json:"upload_id,omitempty"`
	ApprovalID  *uuid.UUID `json:"approval_id,omitempty"`
}

// Service provides task business logic
type Service struct {
	repo *Repository
	pool *pgxpool.Pool
}

// NewService creates a new task service
func NewService(pool *pgxpool.Pool) *Service {
	return &Service{
		repo: NewRepository(pool),
		pool: pool,
	}
}

// Repository returns the underlying repository
func (s *Service) Repository() *Repository {
	return s.repo
}

// Create creates a new task
func (s *Service) Create(ctx context.Context, req *CreateRequest) (*ClientTask, error) {
	priority := req.Priority
	if priority == "" {
		priority = PriorityMedium
	}

	task := &ClientTask{
		TenantID:    req.TenantID,
		ClientID:    req.ClientID,
		CreatedBy:   req.CreatedBy,
		Title:       req.Title,
		Description: req.Description,
		Status:      StatusOpen,
		Priority:    priority,
		DueDate:     req.DueDate,
		DocumentID:  req.DocumentID,
		UploadID:    req.UploadID,
		ApprovalID:  req.ApprovalID,
	}

	if err := s.repo.Create(ctx, task); err != nil {
		return nil, err
	}

	return task, nil
}

// GetByID retrieves a task by ID
func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*ClientTask, error) {
	return s.repo.GetByID(ctx, id)
}

// ListForTenant returns tasks for a tenant
func (s *Service) ListForTenant(ctx context.Context, tenantID uuid.UUID, status *Status, limit, offset int) ([]*ClientTask, int, error) {
	return s.repo.ListByTenant(ctx, tenantID, status, limit, offset)
}

// ListForClient returns tasks for a client
func (s *Service) ListForClient(ctx context.Context, clientID uuid.UUID, status *Status, limit, offset int) ([]*ClientTask, int, error) {
	return s.repo.ListByClient(ctx, clientID, status, limit, offset)
}

// Update updates a task
func (s *Service) Update(ctx context.Context, id uuid.UUID, req *UpdateRequest) (*ClientTask, error) {
	task, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.Title != nil {
		task.Title = *req.Title
	}
	if req.Description != nil {
		task.Description = req.Description
	}
	if req.Priority != nil {
		task.Priority = *req.Priority
	}
	if req.DueDate != nil {
		task.DueDate = req.DueDate
	}
	if req.DocumentID != nil {
		task.DocumentID = req.DocumentID
	}
	if req.UploadID != nil {
		task.UploadID = req.UploadID
	}
	if req.ApprovalID != nil {
		task.ApprovalID = req.ApprovalID
	}

	if err := s.repo.Update(ctx, task); err != nil {
		return nil, err
	}

	return task, nil
}

// Complete marks a task as completed
func (s *Service) Complete(ctx context.Context, id uuid.UUID) error {
	return s.repo.Complete(ctx, id)
}

// Cancel cancels a task
func (s *Service) Cancel(ctx context.Context, id uuid.UUID) error {
	return s.repo.Cancel(ctx, id)
}

// Delete deletes a task
func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}

// CountOpenForClient counts open tasks for a client
func (s *Service) CountOpenForClient(ctx context.Context, clientID uuid.UUID) (int, error) {
	return s.repo.CountOpenByClient(ctx, clientID)
}

// ListOverdue returns overdue tasks for a tenant
func (s *Service) ListOverdue(ctx context.Context, tenantID uuid.UUID) ([]*ClientTask, error) {
	return s.repo.ListOverdue(ctx, tenantID)
}
