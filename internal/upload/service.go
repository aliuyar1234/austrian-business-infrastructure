package upload

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/austrian-business-infrastructure/fo/internal/storage"
)

var (
	ErrFileTooLarge     = errors.New("file size exceeds maximum allowed")
	ErrInvalidFileType  = errors.New("file type not allowed")
	ErrStorageError     = errors.New("storage operation failed")
	ErrNoAccountAccess  = errors.New("no access to this account")
)

// UploadRequest contains data for creating an upload
type UploadRequest struct {
	ClientID  uuid.UUID
	AccountID uuid.UUID
	Filename  string
	FileSize  int64
	MimeType  string
	Category  *Category
	Note      *string
	Reader    io.Reader
}

// Service provides upload business logic
type Service struct {
	repo             *Repository
	storage          storage.Client
	maxFileSize      int64
	allowedMimeTypes map[string]bool
	uploadPath       string
}

// NewService creates a new upload service
func NewService(pool *pgxpool.Pool, storageClient storage.Client, maxFileSize int64, uploadPath string) *Service {
	return &Service{
		repo:        NewRepository(pool),
		storage:     storageClient,
		maxFileSize: maxFileSize,
		uploadPath:  uploadPath,
		allowedMimeTypes: map[string]bool{
			"application/pdf":                                                       true,
			"image/jpeg":                                                             true,
			"image/png":                                                              true,
			"application/msword":                                                     true,
			"application/vnd.openxmlformats-officedocument.wordprocessingml.document": true,
			"application/vnd.ms-excel":                                               true,
			"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet":      true,
		},
	}
}

// Repository returns the underlying repository
func (s *Service) Repository() *Repository {
	return s.repo
}

// Upload processes a file upload
func (s *Service) Upload(ctx context.Context, req *UploadRequest) (*Upload, error) {
	// Validate file size
	if req.FileSize > s.maxFileSize {
		return nil, ErrFileTooLarge
	}

	// Validate file type
	if !s.allowedMimeTypes[req.MimeType] {
		return nil, ErrInvalidFileType
	}

	// Generate storage path
	uploadID := uuid.New()
	storagePath := s.generateStoragePath(req.ClientID, req.AccountID, uploadID, req.Filename)

	// Calculate content hash while uploading
	hasher := sha256.New()
	teeReader := io.TeeReader(req.Reader, hasher)

	// Store file
	err := s.storage.Put(ctx, storagePath, teeReader, req.MimeType)
	if err != nil {
		return nil, errors.Join(ErrStorageError, err)
	}

	contentHash := hex.EncodeToString(hasher.Sum(nil))

	// Create database record
	now := time.Now()
	upload := &Upload{
		ID:          uploadID,
		ClientID:    req.ClientID,
		AccountID:   req.AccountID,
		Filename:    req.Filename,
		StoragePath: storagePath,
		FileSize:    req.FileSize,
		MimeType:    &req.MimeType,
		ContentHash: &contentHash,
		Category:    req.Category,
		Note:        req.Note,
		UploadDate:  now,
		Status:      StatusNew,
	}

	if err := s.repo.Create(ctx, upload); err != nil {
		// Cleanup storage on database error
		_ = s.storage.Delete(ctx, storagePath)
		return nil, err
	}

	return upload, nil
}

// generateStoragePath creates a unique storage path for an upload
func (s *Service) generateStoragePath(clientID, accountID, uploadID uuid.UUID, filename string) string {
	ext := filepath.Ext(filename)
	return filepath.Join(
		s.uploadPath,
		clientID.String(),
		accountID.String(),
		uploadID.String()+ext,
	)
}

// GetByID retrieves an upload by ID
func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*Upload, error) {
	return s.repo.GetByID(ctx, id)
}

// ListByClient returns uploads for a client
func (s *Service) ListByClient(ctx context.Context, clientID uuid.UUID, status *Status, limit, offset int) ([]*Upload, int, error) {
	return s.repo.ListByClient(ctx, clientID, status, limit, offset)
}

// ListByTenant returns uploads for a tenant
func (s *Service) ListByTenant(ctx context.Context, tenantID uuid.UUID, status *Status, limit, offset int) ([]*Upload, int, error) {
	return s.repo.ListByTenant(ctx, tenantID, status, limit, offset)
}

// MarkProcessed marks an upload as processed
func (s *Service) MarkProcessed(ctx context.Context, uploadID, processedBy uuid.UUID) error {
	return s.repo.MarkProcessed(ctx, uploadID, processedBy)
}

// Delete deletes an upload and its file
func (s *Service) Delete(ctx context.Context, uploadID uuid.UUID) error {
	upload, err := s.repo.GetByID(ctx, uploadID)
	if err != nil {
		return err
	}

	// Delete from storage first
	if err := s.storage.Delete(ctx, upload.StoragePath); err != nil && !errors.Is(err, storage.ErrNotFound) {
		return errors.Join(ErrStorageError, err)
	}

	// Delete database record
	return s.repo.Delete(ctx, uploadID)
}

// GetFile retrieves the file content for an upload
func (s *Service) GetFile(ctx context.Context, uploadID uuid.UUID) (io.ReadCloser, *Upload, error) {
	upload, err := s.repo.GetByID(ctx, uploadID)
	if err != nil {
		return nil, nil, err
	}

	reader, err := s.storage.Get(ctx, upload.StoragePath)
	if err != nil {
		return nil, nil, errors.Join(ErrStorageError, err)
	}

	return reader, upload, nil
}

// GetStats returns upload statistics
func (s *Service) GetStats(ctx context.Context, clientID uuid.UUID) (*UploadStats, error) {
	newCount, err := s.repo.CountNewByClient(ctx, clientID)
	if err != nil {
		return nil, err
	}

	return &UploadStats{
		NewUploads: newCount,
	}, nil
}

// UploadStats contains upload statistics
type UploadStats struct {
	NewUploads int `json:"new_uploads"`
}
