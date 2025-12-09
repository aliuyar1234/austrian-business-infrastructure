package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// Client is the interface for storage operations
type Client interface {
	// Put uploads data to storage
	Put(ctx context.Context, path string, data io.Reader, contentType string) error

	// Get downloads data from storage
	Get(ctx context.Context, path string) (io.ReadCloser, error)

	// Delete removes data from storage
	Delete(ctx context.Context, path string) error

	// Exists checks if a path exists in storage
	Exists(ctx context.Context, path string) (bool, error)

	// List returns paths matching a prefix
	List(ctx context.Context, prefix string) ([]string, error)
}

// FileMetadata contains metadata about a stored file
type FileMetadata struct {
	Path        string    `json:"path"`
	Size        int64     `json:"size"`
	ContentType string    `json:"content_type"`
	CreatedAt   time.Time `json:"created_at"`
	ModifiedAt  time.Time `json:"modified_at"`
}

// LocalClient implements storage on the local filesystem
type LocalClient struct {
	basePath string
}

// NewLocalClient creates a new local filesystem storage client
func NewLocalClient(basePath string) (*LocalClient, error) {
	// Ensure base path exists
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return nil, fmt.Errorf("create storage directory: %w", err)
	}

	return &LocalClient{
		basePath: basePath,
	}, nil
}

// Put uploads data to local storage
func (c *LocalClient) Put(ctx context.Context, path string, data io.Reader, contentType string) error {
	fullPath := filepath.Join(c.basePath, path)

	// Create directory structure
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create directories: %w", err)
	}

	// Create file
	file, err := os.Create(fullPath)
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}
	defer file.Close()

	// Copy data
	if _, err := io.Copy(file, data); err != nil {
		return fmt.Errorf("write data: %w", err)
	}

	return nil
}

// Get downloads data from local storage
func (c *LocalClient) Get(ctx context.Context, path string) (io.ReadCloser, error) {
	fullPath := filepath.Join(c.basePath, path)

	file, err := os.Open(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("open file: %w", err)
	}

	return file, nil
}

// Delete removes data from local storage
func (c *LocalClient) Delete(ctx context.Context, path string) error {
	fullPath := filepath.Join(c.basePath, path)

	if err := os.Remove(fullPath); err != nil {
		if os.IsNotExist(err) {
			return ErrNotFound
		}
		return fmt.Errorf("delete file: %w", err)
	}

	return nil
}

// Exists checks if a path exists in local storage
func (c *LocalClient) Exists(ctx context.Context, path string) (bool, error) {
	fullPath := filepath.Join(c.basePath, path)

	_, err := os.Stat(fullPath)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, fmt.Errorf("stat file: %w", err)
}

// List returns paths matching a prefix in local storage
func (c *LocalClient) List(ctx context.Context, prefix string) ([]string, error) {
	searchPath := filepath.Join(c.basePath, prefix)

	var paths []string
	err := filepath.Walk(searchPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}

		if !info.IsDir() {
			// Get relative path
			relPath, err := filepath.Rel(c.basePath, path)
			if err != nil {
				return err
			}
			paths = append(paths, relPath)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("walk directory: %w", err)
	}

	return paths, nil
}

// ErrNotFound is returned when a file is not found
var ErrNotFound = fmt.Errorf("file not found")
