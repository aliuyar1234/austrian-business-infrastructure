package document

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// S3Storage implements Storage interface for S3-compatible storage (MinIO, AWS S3)
type S3Storage struct {
	client *minio.Client
	bucket string
}

// NewS3Storage creates a new S3-compatible storage client
func NewS3Storage(cfg *StorageConfig) (*S3Storage, error) {
	// Create MinIO client
	client, err := minio.New(cfg.S3Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.S3AccessKeyID, cfg.S3SecretAccessKey, ""),
		Secure: cfg.S3UseSSL,
		Region: cfg.S3Region,
	})
	if err != nil {
		return nil, fmt.Errorf("create S3 client: %w", err)
	}

	// Check if bucket exists, create if not
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	exists, err := client.BucketExists(ctx, cfg.S3Bucket)
	if err != nil {
		return nil, fmt.Errorf("check bucket: %w", err)
	}

	if !exists {
		err = client.MakeBucket(ctx, cfg.S3Bucket, minio.MakeBucketOptions{
			Region: cfg.S3Region,
		})
		if err != nil {
			return nil, fmt.Errorf("create bucket: %w", err)
		}
	}

	return &S3Storage{
		client: client,
		bucket: cfg.S3Bucket,
	}, nil
}

// Store saves a document to S3
func (s *S3Storage) Store(ctx context.Context, tenantID, accountID, filename string, content io.Reader, contentType string) (*StorageInfo, error) {
	// Generate path
	path := GeneratePath(tenantID, accountID, filename)

	// Upload to S3
	info, err := s.client.PutObject(ctx, s.bucket, path, content, -1, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrStorageWriteFailed, err)
	}

	return &StorageInfo{
		Path:        path,
		Size:        info.Size,
		ContentType: contentType,
		ETag:        info.ETag,
	}, nil
}

// Get retrieves a document from S3
func (s *S3Storage) Get(ctx context.Context, path string) (io.ReadCloser, *StorageInfo, error) {
	// Get object
	obj, err := s.client.GetObject(ctx, s.bucket, path, minio.GetObjectOptions{})
	if err != nil {
		return nil, nil, fmt.Errorf("%w: %v", ErrStorageReadFailed, err)
	}

	// Get object info
	info, err := obj.Stat()
	if err != nil {
		obj.Close()
		errResp := minio.ToErrorResponse(err)
		if errResp.Code == "NoSuchKey" {
			return nil, nil, ErrStorageNotFound
		}
		return nil, nil, fmt.Errorf("%w: %v", ErrStorageReadFailed, err)
	}

	return obj, &StorageInfo{
		Path:        path,
		Size:        info.Size,
		ContentType: info.ContentType,
		ModTime:     info.LastModified,
		ETag:        info.ETag,
	}, nil
}

// Delete removes a document from S3
func (s *S3Storage) Delete(ctx context.Context, path string) error {
	err := s.client.RemoveObject(ctx, s.bucket, path, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("%w: %v", ErrStorageDeleteFailed, err)
	}
	return nil
}

// Exists checks if a document exists in S3
func (s *S3Storage) Exists(ctx context.Context, path string) (bool, error) {
	_, err := s.client.StatObject(ctx, s.bucket, path, minio.StatObjectOptions{})
	if err != nil {
		errResp := minio.ToErrorResponse(err)
		if errResp.Code == "NoSuchKey" {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// GetSignedURL returns a presigned URL for direct download
func (s *S3Storage) GetSignedURL(ctx context.Context, path string, expiry time.Duration) (string, error) {
	// Default expiry to 15 minutes
	if expiry == 0 {
		expiry = 15 * time.Minute
	}

	// Maximum expiry is 7 days
	if expiry > 7*24*time.Hour {
		expiry = 7 * 24 * time.Hour
	}

	url, err := s.client.PresignedGetObject(ctx, s.bucket, path, expiry, nil)
	if err != nil {
		return "", fmt.Errorf("generate presigned URL: %w", err)
	}

	return url.String(), nil
}

// List returns all documents under a prefix
func (s *S3Storage) List(ctx context.Context, prefix string) ([]StorageInfo, error) {
	var results []StorageInfo

	opts := minio.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: true,
	}

	for obj := range s.client.ListObjects(ctx, s.bucket, opts) {
		if obj.Err != nil {
			return nil, obj.Err
		}

		results = append(results, StorageInfo{
			Path:        obj.Key,
			Size:        obj.Size,
			ContentType: obj.ContentType,
			ModTime:     obj.LastModified,
			ETag:        obj.ETag,
		})
	}

	return results, nil
}

// GetUsage returns total storage usage for a tenant
func (s *S3Storage) GetUsage(ctx context.Context, tenantID string) (int64, error) {
	var totalSize int64

	opts := minio.ListObjectsOptions{
		Prefix:    tenantID + "/",
		Recursive: true,
	}

	for obj := range s.client.ListObjects(ctx, s.bucket, opts) {
		if obj.Err != nil {
			return 0, obj.Err
		}
		totalSize += obj.Size
	}

	return totalSize, nil
}

// BulkDelete removes multiple documents from S3
func (s *S3Storage) BulkDelete(ctx context.Context, paths []string) error {
	objectsCh := make(chan minio.ObjectInfo, len(paths))

	go func() {
		defer close(objectsCh)
		for _, path := range paths {
			objectsCh <- minio.ObjectInfo{Key: path}
		}
	}()

	opts := minio.RemoveObjectsOptions{
		GovernanceBypass: true,
	}

	errorCh := s.client.RemoveObjects(ctx, s.bucket, objectsCh, opts)

	var errors []string
	for e := range errorCh {
		errors = append(errors, e.Err.Error())
	}

	if len(errors) > 0 {
		return fmt.Errorf("bulk delete errors: %s", strings.Join(errors, "; "))
	}

	return nil
}

// CopyObject copies a document within S3
func (s *S3Storage) CopyObject(ctx context.Context, srcPath, dstPath string) error {
	src := minio.CopySrcOptions{
		Bucket: s.bucket,
		Object: srcPath,
	}

	dst := minio.CopyDestOptions{
		Bucket: s.bucket,
		Object: dstPath,
	}

	_, err := s.client.CopyObject(ctx, dst, src)
	if err != nil {
		return fmt.Errorf("copy object: %w", err)
	}

	return nil
}
