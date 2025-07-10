package storage

import (
	"context"
	"io"
)

// StorageService defines the interface for storage operations
type StorageService interface {
	// Upload uploads a file to storage and returns the public URL
	Upload(ctx context.Context, bucket, objectName string, content []byte, contentType string) (string, error)

	// Download downloads a file from storage
	Download(ctx context.Context, bucket, objectName string) ([]byte, error)

	// Delete deletes a file from storage
	Delete(ctx context.Context, bucket, objectName string) error

	// GetSignedURL gets a signed URL for a file
	GetSignedURL(ctx context.Context, bucket, objectName string, expires int64) (string, error)

	// StreamUpload uploads a file from a reader
	StreamUpload(ctx context.Context, bucket, objectName string, reader io.Reader, contentType string) (string, error)
}
