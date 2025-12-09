package share

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ShareRequest contains data for creating a share
type ShareRequest struct {
	DocumentID  uuid.UUID  `json:"document_id"`
	ClientID    uuid.UUID  `json:"client_id"`
	SharedBy    uuid.UUID  `json:"shared_by"`
	CanDownload bool       `json:"can_download"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
}

// Service provides share business logic
type Service struct {
	repo *Repository
	pool *pgxpool.Pool
}

// NewService creates a new share service
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

// Share creates a new document share
func (s *Service) Share(ctx context.Context, req *ShareRequest) (*DocumentShare, error) {
	share := &DocumentShare{
		DocumentID:  req.DocumentID,
		ClientID:    req.ClientID,
		SharedBy:    req.SharedBy,
		CanDownload: req.CanDownload,
		ExpiresAt:   req.ExpiresAt,
	}

	if err := s.repo.Create(ctx, share); err != nil {
		return nil, err
	}

	return share, nil
}

// Unshare removes a document share
func (s *Service) Unshare(ctx context.Context, documentID, clientID uuid.UUID) error {
	share, err := s.repo.GetByDocumentAndClient(ctx, documentID, clientID)
	if err != nil {
		return err
	}

	return s.repo.Delete(ctx, share.ID)
}

// ListForClient returns documents shared with a client
func (s *Service) ListForClient(ctx context.Context, clientID uuid.UUID, limit, offset int) ([]*DocumentShare, int, error) {
	return s.repo.ListByClient(ctx, clientID, limit, offset)
}

// ListForDocument returns all clients a document is shared with
func (s *Service) ListForDocument(ctx context.Context, documentID uuid.UUID) ([]*DocumentShare, error) {
	return s.repo.ListByDocument(ctx, documentID)
}

// GetByID retrieves a share by ID
func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*DocumentShare, error) {
	return s.repo.GetByID(ctx, id)
}

// RecordView records that a client viewed a shared document
func (s *Service) RecordView(ctx context.Context, shareID uuid.UUID) error {
	return s.repo.RecordView(ctx, shareID)
}

// HasAccess checks if a client has access to a document
func (s *Service) HasAccess(ctx context.Context, documentID, clientID uuid.UUID) (bool, *DocumentShare, error) {
	share, err := s.repo.GetByDocumentAndClient(ctx, documentID, clientID)
	if err != nil {
		if err == ErrShareNotFound {
			return false, nil, nil
		}
		return false, nil, err
	}

	// Check if expired
	if share.ExpiresAt != nil && share.ExpiresAt.Before(time.Now()) {
		return false, nil, nil
	}

	return true, share, nil
}
