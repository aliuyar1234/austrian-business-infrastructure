package document

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"time"

	"github.com/google/uuid"
)

// Default limits for document uploads
const (
	DefaultMaxDocumentSize = 50 * 1024 * 1024 // 50MB default max
)

// Service handles document business logic
type Service struct {
	repo           *Repository
	storage        Storage
	maxDocumentSize int64
}

// NewService creates a new document service
func NewService(repo *Repository, storage Storage) *Service {
	return &Service{
		repo:           repo,
		storage:        storage,
		maxDocumentSize: DefaultMaxDocumentSize,
	}
}

// NewServiceWithLimit creates a new document service with custom size limit
func NewServiceWithLimit(repo *Repository, storage Storage, maxSize int64) *Service {
	return &Service{
		repo:           repo,
		storage:        storage,
		maxDocumentSize: maxSize,
	}
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

// GetByID retrieves a document by ID
func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*Document, error) {
	return s.repo.GetByID(ctx, id)
}

// GetContent retrieves document content
func (s *Service) GetContent(ctx context.Context, id uuid.UUID) (io.ReadCloser, *StorageInfo, error) {
	doc, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, nil, err
	}

	return s.storage.Get(ctx, doc.StoragePath)
}

// GetSignedURL returns a presigned URL for direct download
func (s *Service) GetSignedURL(ctx context.Context, id uuid.UUID, expiry time.Duration) (string, error) {
	doc, err := s.repo.GetByID(ctx, id)
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

// MarkAsRead marks a document as read
func (s *Service) MarkAsRead(ctx context.Context, id uuid.UUID) error {
	return s.repo.UpdateStatus(ctx, id, StatusRead)
}

// UpdateStatus updates the status of a document
func (s *Service) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	// Validate status
	if status != StatusNew && status != StatusRead && status != StatusArchived {
		return fmt.Errorf("invalid status: %s", status)
	}
	return s.repo.UpdateStatus(ctx, id, status)
}

// Archive archives a document
func (s *Service) Archive(ctx context.Context, id uuid.UUID) error {
	return s.repo.Archive(ctx, id)
}

// BulkArchive archives multiple documents
func (s *Service) BulkArchive(ctx context.Context, ids []uuid.UUID) (int, error) {
	return s.repo.BulkArchive(ctx, ids)
}

// Delete permanently removes a document
func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	doc, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// Delete from storage
	if err := s.storage.Delete(ctx, doc.StoragePath); err != nil {
		return fmt.Errorf("delete from storage: %w", err)
	}

	// Delete record
	return s.repo.Delete(ctx, id)
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
func (s *Service) GetExpired(ctx context.Context, tenantID uuid.UUID) ([]*Document, error) {
	return s.repo.GetExpired(ctx, tenantID)
}

// DeleteExpired deletes all expired documents
func (s *Service) DeleteExpired(ctx context.Context, tenantID uuid.UUID) (int, error) {
	expired, err := s.repo.GetExpired(ctx, tenantID)
	if err != nil {
		return 0, err
	}

	deleted := 0
	for _, doc := range expired {
		if err := s.Delete(ctx, doc.ID); err != nil {
			continue // Log error but continue with others
		}
		deleted++
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
