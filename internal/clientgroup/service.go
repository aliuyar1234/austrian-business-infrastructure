package clientgroup

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// CreateRequest contains data for creating a group
type CreateRequest struct {
	TenantID    uuid.UUID `json:"tenant_id"`
	Name        string    `json:"name"`
	Description *string   `json:"description,omitempty"`
	Color       *string   `json:"color,omitempty"`
}

// UpdateRequest contains data for updating a group
type UpdateRequest struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
	Color       *string `json:"color,omitempty"`
}

// Service provides client group business logic
type Service struct {
	repo *Repository
	pool *pgxpool.Pool
}

// NewService creates a new client group service
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

// Create creates a new client group
func (s *Service) Create(ctx context.Context, req *CreateRequest) (*ClientGroup, error) {
	group := &ClientGroup{
		TenantID:    req.TenantID,
		Name:        req.Name,
		Description: req.Description,
		Color:       req.Color,
	}

	if err := s.repo.Create(ctx, group); err != nil {
		return nil, err
	}

	return group, nil
}

// GetByID retrieves a group by ID
func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*ClientGroup, error) {
	return s.repo.GetByID(ctx, id)
}

// ListForTenant returns all groups for a tenant
func (s *Service) ListForTenant(ctx context.Context, tenantID uuid.UUID) ([]*ClientGroup, error) {
	return s.repo.ListByTenant(ctx, tenantID)
}

// Update updates a group
func (s *Service) Update(ctx context.Context, id uuid.UUID, req *UpdateRequest) (*ClientGroup, error) {
	group, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.Name != nil {
		group.Name = *req.Name
	}
	if req.Description != nil {
		group.Description = req.Description
	}
	if req.Color != nil {
		group.Color = req.Color
	}

	if err := s.repo.Update(ctx, group); err != nil {
		return nil, err
	}

	return group, nil
}

// Delete deletes a group
func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}

// AddMember adds a client to a group
func (s *Service) AddMember(ctx context.Context, groupID, clientID uuid.UUID) error {
	return s.repo.AddMember(ctx, groupID, clientID)
}

// RemoveMember removes a client from a group
func (s *Service) RemoveMember(ctx context.Context, groupID, clientID uuid.UUID) error {
	return s.repo.RemoveMember(ctx, groupID, clientID)
}

// SetMembers replaces all members of a group
func (s *Service) SetMembers(ctx context.Context, groupID uuid.UUID, clientIDs []uuid.UUID) error {
	return s.repo.SetMembers(ctx, groupID, clientIDs)
}

// ListMembers returns all members of a group
func (s *Service) ListMembers(ctx context.Context, groupID uuid.UUID) ([]*GroupMember, error) {
	return s.repo.ListMembers(ctx, groupID)
}

// ListGroupsForClient returns all groups a client belongs to
func (s *Service) ListGroupsForClient(ctx context.Context, clientID uuid.UUID) ([]*ClientGroup, error) {
	return s.repo.ListGroupsForClient(ctx, clientID)
}

// GetMemberClientIDs returns all client IDs in a group
func (s *Service) GetMemberClientIDs(ctx context.Context, groupID uuid.UUID) ([]uuid.UUID, error) {
	return s.repo.GetMemberClientIDs(ctx, groupID)
}
