package approval

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// CreateRequest contains data for creating an approval request
type CreateRequest struct {
	DocumentID  uuid.UUID `json:"document_id"`
	ClientID    uuid.UUID `json:"client_id"`
	RequestedBy uuid.UUID `json:"requested_by"`
	Message     *string   `json:"message,omitempty"`
}

// Service provides approval business logic
type Service struct {
	repo *Repository
	pool *pgxpool.Pool
}

// NewService creates a new approval service
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

// Create creates a new approval request
func (s *Service) Create(ctx context.Context, req *CreateRequest) (*ApprovalRequest, error) {
	approval := &ApprovalRequest{
		DocumentID:  req.DocumentID,
		ClientID:    req.ClientID,
		RequestedBy: req.RequestedBy,
		Message:     req.Message,
	}

	if err := s.repo.Create(ctx, approval); err != nil {
		return nil, err
	}

	return approval, nil
}

// GetByID retrieves an approval by ID
func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*ApprovalRequest, error) {
	return s.repo.GetByID(ctx, id)
}

// GetByIDWithDetails retrieves an approval with document and client details
func (s *Service) GetByIDWithDetails(ctx context.Context, id uuid.UUID) (*ApprovalRequest, error) {
	return s.repo.GetByIDWithDetails(ctx, id)
}

// ListForClient returns approvals for a client
func (s *Service) ListForClient(ctx context.Context, clientID uuid.UUID, status *Status, limit, offset int) ([]*ApprovalRequest, int, error) {
	return s.repo.ListByClient(ctx, clientID, status, limit, offset)
}

// ListPendingForTenant returns pending approvals for a tenant
func (s *Service) ListPendingForTenant(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]*ApprovalRequest, int, error) {
	return s.repo.ListPendingByTenant(ctx, tenantID, limit, offset)
}

// Approve approves an approval request
func (s *Service) Approve(ctx context.Context, approvalID uuid.UUID) error {
	return s.repo.Approve(ctx, approvalID)
}

// Reject rejects an approval request
func (s *Service) Reject(ctx context.Context, approvalID uuid.UUID, comment string) error {
	if comment == "" {
		return ErrInvalidStatus
	}
	return s.repo.Reject(ctx, approvalID, comment)
}

// RequestRevision requests revision on an approval request
func (s *Service) RequestRevision(ctx context.Context, approvalID uuid.UUID, comment string) error {
	if comment == "" {
		return ErrInvalidStatus
	}
	return s.repo.RequestRevision(ctx, approvalID, comment)
}

// CountPendingForClient counts pending approvals for a client
func (s *Service) CountPendingForClient(ctx context.Context, clientID uuid.UUID) (int, error) {
	return s.repo.CountPendingByClient(ctx, clientID)
}

// MarkReminderSent updates reminder tracking
func (s *Service) MarkReminderSent(ctx context.Context, approvalID uuid.UUID) error {
	return s.repo.UpdateReminderSent(ctx, approvalID)
}
