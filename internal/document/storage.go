package document

import (
	"context"
	"errors"
	"io"
	"time"
)

// Storage errors
var (
	ErrStorageNotFound     = errors.New("document not found in storage")
	ErrStorageWriteFailed  = errors.New("failed to write document to storage")
	ErrStorageReadFailed   = errors.New("failed to read document from storage")
	ErrStorageDeleteFailed = errors.New("failed to delete document from storage")
	ErrInvalidPath         = errors.New("invalid storage path")
)

// StorageInfo contains metadata about a stored document
type StorageInfo struct {
	Path        string
	Size        int64
	ContentType string
	ModTime     time.Time
	ETag        string
}

// Storage defines the interface for document storage backends
type Storage interface {
	// Store saves a document and returns the storage path
	Store(ctx context.Context, tenantID, accountID, filename string, content io.Reader, contentType string) (*StorageInfo, error)

	// Get retrieves a document by path
	Get(ctx context.Context, path string) (io.ReadCloser, *StorageInfo, error)

	// Delete removes a document from storage
	Delete(ctx context.Context, path string) error

	// Exists checks if a document exists at the given path
	Exists(ctx context.Context, path string) (bool, error)

	// GetSignedURL returns a time-limited URL for direct download (for S3-compatible storage)
	// Returns empty string and nil error if not supported (local storage)
	GetSignedURL(ctx context.Context, path string, expiry time.Duration) (string, error)

	// List returns all documents under a prefix
	List(ctx context.Context, prefix string) ([]StorageInfo, error)

	// GetUsage returns total storage usage in bytes for a tenant
	GetUsage(ctx context.Context, tenantID string) (int64, error)
}

// StorageType identifies the storage backend type
type StorageType string

const (
	StorageTypeLocal StorageType = "local"
	StorageTypeS3    StorageType = "s3"
)

// StorageConfig holds configuration for storage backends
type StorageConfig struct {
	Type StorageType

	// Local storage config
	LocalPath string

	// S3 storage config
	S3Endpoint        string
	S3Bucket          string
	S3Region          string
	S3AccessKeyID     string
	S3SecretAccessKey string
	S3UseSSL          bool
}

// NewStorage creates a new storage instance based on configuration
func NewStorage(cfg *StorageConfig) (Storage, error) {
	switch cfg.Type {
	case StorageTypeLocal:
		return NewLocalStorage(cfg.LocalPath)
	case StorageTypeS3:
		return NewS3Storage(cfg)
	default:
		return NewLocalStorage(cfg.LocalPath)
	}
}

// GeneratePath creates a storage path for a document
// Format: tenants/{tenant_id}/accounts/{account_id}/{year}/{month}/{filename}
func GeneratePath(tenantID, accountID, filename string) string {
	now := time.Now()
	return tenantID + "/accounts/" + accountID + "/" +
		now.Format("2006") + "/" + now.Format("01") + "/" + filename
}
