package document

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/google/uuid"
)

// Default limits for document uploads
const (
	DefaultMaxDocumentSize = 50 * 1024 * 1024 // 50MB default max
)

// ErrAccountNotOwned is returned when the account doesn't belong to the tenant
var ErrAccountNotOwned = errors.New("account does not belong to tenant")

// AccountVerifier verifies account ownership for tenant isolation
type AccountVerifier interface {
	// VerifyAccountOwnership checks if an account belongs to the specified tenant
	// Returns nil if the account exists and belongs to the tenant, error otherwise
	VerifyAccountOwnership(ctx context.Context, accountID, tenantID uuid.UUID) error
}

// Service handles document business logic
type Service struct {
	repo            *Repository
	storage         Storage
	accountVerifier AccountVerifier
	maxDocumentSize int64
}

// NewService creates a new document service
func NewService(repo *Repository, storage Storage) *Service {
	return &Service{
		repo:            repo,
		storage:         storage,
		maxDocumentSize: DefaultMaxDocumentSize,
	}
}

// NewServiceWithAccountVerifier creates a document service with account ownership verification
func NewServiceWithAccountVerifier(repo *Repository, storage Storage, verifier AccountVerifier) *Service {
	return &Service{
		repo:            repo,
		storage:         storage,
		accountVerifier: verifier,
		maxDocumentSize: DefaultMaxDocumentSize,
	}
}

// NewServiceWithLimit creates a new document service with custom size limit
func NewServiceWithLimit(repo *Repository, storage Storage, maxSize int64) *Service {
	return &Service{
		repo:            repo,
		storage:         storage,
		maxDocumentSize: maxSize,
	}
}

// SetAccountVerifier sets the account verifier (for dependency injection after construction)
func (s *Service) SetAccountVerifier(verifier AccountVerifier) {
	s.accountVerifier = verifier
}

// CreateDocumentInput holds input for creating a document
type CreateDocumentInput struct {
	AccountID   uuid.UUID
	ExternalID  string
	Type        string
	Title       string
	Sender      string
	ReceivedAt  time.Time
	Content     io.Reader
	ContentType string
	Metadata    map[string]interface{}
}

// Create stores a new document
func (s *Service) Create(ctx context.Context, tenantID string, input *CreateDocumentInput) (*Document, error) {
	// CRITICAL: Verify account belongs to the caller's tenant before creating document
	// This prevents cross-tenant data writes (IDOR vulnerability)
	if s.accountVerifier != nil {
		tenantUUID, err := uuid.Parse(tenantID)
		if err != nil {
			return nil, fmt.Errorf("invalid tenant ID: %w", err)
		}
		if err := s.accountVerifier.VerifyAccountOwnership(ctx, input.AccountID, tenantUUID); err != nil {
			return nil, ErrAccountNotOwned
		}
	}

	// Check if document already exists by external ID
	existing, err := s.repo.GetByExternalID(ctx, input.AccountID, input.ExternalID)
	if err == nil && existing != nil {
		return existing, ErrDuplicateDocument
	}

	// Read content with size limit to prevent memory exhaustion
	// Use LimitReader to cap memory usage; if content exceeds limit, Read will stop
	limitedReader := io.LimitReader(input.Content, s.maxDocumentSize+1)
	content, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, fmt.Errorf("read content: %w", err)
	}

	// Check if content exceeded size limit
	if int64(len(content)) > s.maxDocumentSize {
		return nil, ErrDocumentTooLarge
	}

	// Calculate content hash for deduplication
	contentHash := calculateHash(content)

	// Check if document with same hash exists (deduplication)
	existingByHash, err := s.repo.GetByContentHash(ctx, input.AccountID, contentHash)
	if err == nil && existingByHash != nil {
		// Document with same content already exists
		return existingByHash, nil
	}

	// Generate filename from external ID or UUID
	filename := input.ExternalID
	if filename == "" {
		filename = uuid.New().String()
	}
	filename = sanitizeFilename(filename) + getExtension(input.ContentType)

	// Store content
	storageInfo, err := s.storage.Store(ctx, tenantID, input.AccountID.String(), filename, newBytesReader(content), input.ContentType)
	if err != nil {
		return nil, fmt.Errorf("store content: %w", err)
	}

	// Create document record
	doc := &Document{
		AccountID:   input.AccountID,
		ExternalID:  input.ExternalID,
		Type:        input.Type,
		Title:       input.Title,
		Sender:      input.Sender,
		ReceivedAt:  input.ReceivedAt,
		ContentHash: contentHash,
		StoragePath: storageInfo.Path,
		FileSize:    int(storageInfo.Size),
		MimeType:    input.ContentType,
		Status:      StatusNew,
		Metadata:    input.Metadata,
	}

	if err := s.repo.Create(ctx, doc); err != nil {
		// Cleanup storage on failure
		s.storage.Delete(ctx, storageInfo.Path)
		return nil, fmt.Errorf("create document record: %w", err)
	}

	return doc, nil
}

// GetByID retrieves a document by ID with tenant isolation
func (s *Service) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*Document, error) {
	return s.repo.GetByID(ctx, tenantID, id)
}

// GetContent retrieves document content with tenant isolation
func (s *Service) GetContent(ctx context.Context, tenantID, id uuid.UUID) (io.ReadCloser, *StorageInfo, error) {
	doc, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, nil, err
	}

	return s.storage.Get(ctx, doc.StoragePath)
}

// GetSignedURL returns a presigned URL for direct download with tenant isolation
func (s *Service) GetSignedURL(ctx context.Context, tenantID, id uuid.UUID, expiry time.Duration) (string, error) {
	doc, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return "", err
	}

	url, err := s.storage.GetSignedURL(ctx, doc.StoragePath, expiry)
	if err != nil {
		return "", err
	}

	if url == "" {
		return "", ErrSignedURLNotSupported
	}

	return url, nil
}

// MarkAsRead marks a document as read with tenant isolation
func (s *Service) MarkAsRead(ctx context.Context, tenantID, id uuid.UUID) error {
	return s.repo.UpdateStatus(ctx, tenantID, id, StatusRead)
}

// UpdateStatus updates the status of a document with tenant isolation
func (s *Service) UpdateStatus(ctx context.Context, tenantID, id uuid.UUID, status string) error {
	// Validate status
	if status != StatusNew && status != StatusRead && status != StatusArchived {
		return fmt.Errorf("invalid status: %s", status)
	}
	return s.repo.UpdateStatus(ctx, tenantID, id, status)
}

// Archive archives a document with tenant isolation
func (s *Service) Archive(ctx context.Context, tenantID, id uuid.UUID) error {
	return s.repo.Archive(ctx, tenantID, id)
}

// BulkArchive archives multiple documents with tenant isolation
func (s *Service) BulkArchive(ctx context.Context, tenantID uuid.UUID, ids []uuid.UUID) (int, error) {
	return s.repo.BulkArchive(ctx, tenantID, ids)
}

// Delete permanently removes a document with tenant isolation
func (s *Service) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	doc, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return err
	}

	// Delete from storage
	if err := s.storage.Delete(ctx, doc.StoragePath); err != nil {
		return fmt.Errorf("delete from storage: %w", err)
	}

	// Delete record
	return s.repo.Delete(ctx, tenantID, id)
}

// List returns documents matching the filter
func (s *Service) List(ctx context.Context, filter *DocumentFilter) ([]*Document, int, error) {
	return s.repo.List(ctx, filter)
}

// GetStats returns document statistics
func (s *Service) GetStats(ctx context.Context, tenantID uuid.UUID) (*DocumentStats, error) {
	return s.repo.GetStats(ctx, tenantID)
}

// GetUnreadCounts returns unread document counts per account
func (s *Service) GetUnreadCounts(ctx context.Context, tenantID uuid.UUID) (map[uuid.UUID]int, error) {
	return s.repo.GetUnreadCount(ctx, tenantID)
}

// GetExpired returns documents past their retention date
func (s *Service) GetExpired(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]*Document, int, error) {
	return s.repo.GetExpired(ctx, tenantID, limit, offset)
}

// DeleteExpired deletes all expired documents in batches
func (s *Service) DeleteExpired(ctx context.Context, tenantID uuid.UUID) (int, error) {
	deleted := 0
	batchSize := 100

	// Process in batches to avoid memory issues with large numbers of expired docs
	for {
		expired, _, err := s.repo.GetExpired(ctx, tenantID, batchSize, 0)
		if err != nil {
			return deleted, err
		}

		if len(expired) == 0 {
			break
		}

		for _, doc := range expired {
			if err := s.Delete(ctx, tenantID, doc.ID); err != nil {
				continue // Log error but continue with others
			}
			deleted++
		}

		// If we got fewer than batch size, we're done
		if len(expired) < batchSize {
			break
		}
	}

	return deleted, nil
}

// GetStorageUsage returns storage usage for a tenant
func (s *Service) GetStorageUsage(ctx context.Context, tenantID string) (int64, error) {
	return s.storage.GetUsage(ctx, tenantID)
}

// Helper functions

// calculateHash computes SHA-256 hash of content
func calculateHash(content []byte) string {
	hash := sha256.Sum256(content)
	return hex.EncodeToString(hash[:])
}

// CalculateHashFromReader computes SHA-256 hash from a reader
func CalculateHashFromReader(r io.Reader) (string, error) {
	hash := sha256.New()
	if _, err := io.Copy(hash, r); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

// sanitizeFilename removes unsafe characters from filename
func sanitizeFilename(name string) string {
	// Replace unsafe characters
	result := make([]byte, 0, len(name))
	for i := 0; i < len(name); i++ {
		c := name[i]
		if isAlphanumeric(c) || c == '-' || c == '_' || c == '.' {
			result = append(result, c)
		} else if c == ' ' {
			result = append(result, '_')
		}
	}
	if len(result) == 0 {
		return "document"
	}
	return string(result)
}

func isAlphanumeric(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')
}

// getExtension returns file extension for content type
func getExtension(contentType string) string {
	switch contentType {
	case "application/pdf":
		return ".pdf"
	case "application/xml", "text/xml":
		return ".xml"
	case "text/html":
		return ".html"
	case "text/plain":
		return ".txt"
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "application/json":
		return ".json"
	default:
		return ""
	}
}

// bytesReader wraps []byte as io.Reader
type bytesReader struct {
	data []byte
	pos  int
}

func newBytesReader(data []byte) *bytesReader {
	return &bytesReader{data: data}
}

func (r *bytesReader) Read(p []byte) (n int, err error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	n = copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}
