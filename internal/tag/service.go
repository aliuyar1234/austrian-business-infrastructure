package tag

import (
	"context"

	"github.com/google/uuid"
)

// Service handles tag business logic
type Service struct {
	repo *Repository
}

// NewService creates a new tag service
func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// CreateTag creates a new tag
func (s *Service) CreateTag(ctx context.Context, tenantID uuid.UUID, name string, color *string) (*Tag, error) {
	tag := &Tag{
		TenantID: tenantID,
		Name:     name,
		Color:    color,
	}
	return s.repo.Create(ctx, tag)
}

// GetTag retrieves a tag by ID
func (s *Service) GetTag(ctx context.Context, id, tenantID uuid.UUID) (*Tag, error) {
	return s.repo.GetByID(ctx, id, tenantID)
}

// ListTags retrieves all tags for a tenant
func (s *Service) ListTags(ctx context.Context, tenantID uuid.UUID) ([]*Tag, error) {
	return s.repo.List(ctx, tenantID)
}

// UpdateTag updates a tag
func (s *Service) UpdateTag(ctx context.Context, id, tenantID uuid.UUID, name string, color *string) (*Tag, error) {
	tag, err := s.repo.GetByID(ctx, id, tenantID)
	if err != nil {
		return nil, err
	}

	tag.Name = name
	tag.Color = color

	if err := s.repo.Update(ctx, tag); err != nil {
		return nil, err
	}

	return tag, nil
}

// DeleteTag deletes a tag
func (s *Service) DeleteTag(ctx context.Context, id, tenantID uuid.UUID) error {
	return s.repo.Delete(ctx, id, tenantID)
}

// AddTagToAccount assigns a tag to an account
func (s *Service) AddTagToAccount(ctx context.Context, accountID, tagID, tenantID uuid.UUID) error {
	// Verify tag belongs to tenant
	_, err := s.repo.GetByID(ctx, tagID, tenantID)
	if err != nil {
		return err
	}
	return s.repo.AddToAccount(ctx, accountID, tagID)
}

// RemoveTagFromAccount removes a tag from an account
func (s *Service) RemoveTagFromAccount(ctx context.Context, accountID, tagID uuid.UUID) error {
	return s.repo.RemoveFromAccount(ctx, accountID, tagID)
}

// GetAccountTags retrieves tags for an account
func (s *Service) GetAccountTags(ctx context.Context, accountID uuid.UUID) ([]*Tag, error) {
	return s.repo.GetAccountTags(ctx, accountID)
}

// SetAccountTags replaces all tags for an account
func (s *Service) SetAccountTags(ctx context.Context, accountID, tenantID uuid.UUID, tagIDs []uuid.UUID) error {
	// Verify all tags belong to tenant
	for _, tagID := range tagIDs {
		_, err := s.repo.GetByID(ctx, tagID, tenantID)
		if err != nil {
			return err
		}
	}
	return s.repo.SetAccountTags(ctx, accountID, tagIDs)
}
