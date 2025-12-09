package document

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// LocalStorage implements Storage interface for local filesystem
type LocalStorage struct {
	basePath string
}

// NewLocalStorage creates a new local filesystem storage
func NewLocalStorage(basePath string) (*LocalStorage, error) {
	// Ensure base path exists
	if err := os.MkdirAll(basePath, 0750); err != nil {
		return nil, fmt.Errorf("create storage directory: %w", err)
	}

	// Resolve to absolute and clean path
	absPath, err := filepath.Abs(basePath)
	if err != nil {
		return nil, fmt.Errorf("resolve storage path: %w", err)
	}

	return &LocalStorage{basePath: filepath.Clean(absPath)}, nil
}

// isPathSafe checks if the given path is safely within the base directory.
// This prevents directory traversal attacks by cleaning and canonicalizing paths.
func (s *LocalStorage) isPathSafe(fullPath string) bool {
	// Clean both paths to resolve any . or .. components
	cleanPath := filepath.Clean(fullPath)
	cleanBase := filepath.Clean(s.basePath)

	// Ensure the cleaned path starts with the base path followed by a separator
	// This prevents attacks like /base/path/../outside being accepted
	if cleanPath == cleanBase {
		return true
	}
	return strings.HasPrefix(cleanPath, cleanBase+string(os.PathSeparator))
}

// Store saves a document to local filesystem
func (s *LocalStorage) Store(ctx context.Context, tenantID, accountID, filename string, content io.Reader, contentType string) (*StorageInfo, error) {
	// Generate path
	relPath := GeneratePath(tenantID, accountID, filename)
	fullPath := filepath.Join(s.basePath, relPath)

	// Validate path is within base directory (prevent directory traversal)
	if !s.isPathSafe(fullPath) {
		return nil, ErrInvalidPath
	}

	// Create directory structure
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrStorageWriteFailed, err)
	}

	// Create file
	file, err := os.Create(fullPath)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrStorageWriteFailed, err)
	}
	defer file.Close()

	// Copy content
	written, err := io.Copy(file, content)
	if err != nil {
		os.Remove(fullPath) // Cleanup on failure
		return nil, fmt.Errorf("%w: %v", ErrStorageWriteFailed, err)
	}

	// Get file info
	info, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("stat file: %w", err)
	}

	return &StorageInfo{
		Path:        relPath,
		Size:        written,
		ContentType: contentType,
		ModTime:     info.ModTime(),
	}, nil
}

// Get retrieves a document from local filesystem
func (s *LocalStorage) Get(ctx context.Context, path string) (io.ReadCloser, *StorageInfo, error) {
	fullPath := filepath.Join(s.basePath, path)

	// Validate path is within base directory
	if !s.isPathSafe(fullPath) {
		return nil, nil, ErrInvalidPath
	}

	file, err := os.Open(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil, ErrStorageNotFound
		}
		return nil, nil, fmt.Errorf("%w: %v", ErrStorageReadFailed, err)
	}

	info, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, nil, fmt.Errorf("stat file: %w", err)
	}

	// Detect content type from extension
	contentType := detectContentType(path)

	return file, &StorageInfo{
		Path:        path,
		Size:        info.Size(),
		ContentType: contentType,
		ModTime:     info.ModTime(),
	}, nil
}

// Delete removes a document from local filesystem
func (s *LocalStorage) Delete(ctx context.Context, path string) error {
	fullPath := filepath.Join(s.basePath, path)

	// Validate path is within base directory
	if !s.isPathSafe(fullPath) {
		return ErrInvalidPath
	}

	err := os.Remove(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // Already deleted
		}
		return fmt.Errorf("%w: %v", ErrStorageDeleteFailed, err)
	}

	// Try to remove empty parent directories
	s.cleanupEmptyDirs(filepath.Dir(fullPath))

	return nil
}

// Exists checks if a document exists
func (s *LocalStorage) Exists(ctx context.Context, path string) (bool, error) {
	fullPath := filepath.Join(s.basePath, path)

	// Validate path is within base directory
	if !s.isPathSafe(fullPath) {
		return false, ErrInvalidPath
	}

	_, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// GetSignedURL is not supported for local storage
func (s *LocalStorage) GetSignedURL(ctx context.Context, path string, expiry time.Duration) (string, error) {
	// Local storage doesn't support signed URLs
	return "", nil
}

// List returns all documents under a prefix
func (s *LocalStorage) List(ctx context.Context, prefix string) ([]StorageInfo, error) {
	listPath := filepath.Join(s.basePath, prefix)

	// Validate path is within base directory
	if !s.isPathSafe(listPath) {
		return nil, ErrInvalidPath
	}

	var results []StorageInfo

	err := filepath.Walk(listPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}
		if info.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(s.basePath, path)
		if err != nil {
			return nil
		}

		results = append(results, StorageInfo{
			Path:        relPath,
			Size:        info.Size(),
			ContentType: detectContentType(path),
			ModTime:     info.ModTime(),
		})

		return nil
	})

	if err != nil {
		return nil, err
	}

	return results, nil
}

// GetUsage returns total storage usage for a tenant
func (s *LocalStorage) GetUsage(ctx context.Context, tenantID string) (int64, error) {
	tenantPath := filepath.Join(s.basePath, tenantID)

	// Validate path
	if !s.isPathSafe(tenantPath) {
		return 0, ErrInvalidPath
	}

	var totalSize int64

	err := filepath.Walk(tenantPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}
		if !info.IsDir() {
			totalSize += info.Size()
		}
		return nil
	})

	if err != nil && !os.IsNotExist(err) {
		return 0, err
	}

	return totalSize, nil
}

// cleanupEmptyDirs removes empty parent directories up to basePath
func (s *LocalStorage) cleanupEmptyDirs(dir string) {
	for dir != s.basePath && s.isPathSafe(dir) {
		entries, err := os.ReadDir(dir)
		if err != nil || len(entries) > 0 {
			break
		}
		os.Remove(dir)
		dir = filepath.Dir(dir)
	}
}

// detectContentType returns MIME type based on file extension
func detectContentType(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".pdf":
		return "application/pdf"
	case ".xml":
		return "application/xml"
	case ".html", ".htm":
		return "text/html"
	case ".txt":
		return "text/plain"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".zip":
		return "application/zip"
	case ".json":
		return "application/json"
	default:
		return "application/octet-stream"
	}
}
