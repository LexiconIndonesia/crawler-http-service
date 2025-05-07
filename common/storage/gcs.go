package storage

import (
	"context"
	"errors"
	"io"

	"cloud.google.com/go/storage"
	"github.com/rs/zerolog/log"
)

// Errors
var (
	ErrNotImplemented = errors.New("method not implemented")
)

// GCSConfig represents the configuration for GCS
type GCSConfig struct {
	ProjectID       string
	CredentialsFile string
}

// GCSStorage implements the StorageService interface for Google Cloud Storage
type GCSStorage struct {
	client *storage.Client
	config GCSConfig
}

// NewGCSStorage creates a new GCS storage service
func NewGCSStorage(ctx context.Context, config GCSConfig) (*GCSStorage, error) {
	// This is just a stub - no actual implementation
	return &GCSStorage{
		config: config,
	}, nil
}

// Upload uploads a file to GCS and returns the public URL
func (g *GCSStorage) Upload(ctx context.Context, bucket, objectName string, content []byte, contentType string) (string, error) {
	// Not implemented as per requirements
	log.Error().Msg("Upload method not implemented")
	return "", ErrNotImplemented
}

// Download downloads a file from GCS
func (g *GCSStorage) Download(ctx context.Context, bucket, objectName string) ([]byte, error) {
	// Not implemented as per requirements
	log.Error().Msg("Download method not implemented")
	return nil, ErrNotImplemented
}

// Delete deletes a file from GCS
func (g *GCSStorage) Delete(ctx context.Context, bucket, objectName string) error {
	// Not implemented as per requirements
	log.Error().Msg("Delete method not implemented")
	return ErrNotImplemented
}

// GetSignedURL gets a signed URL for a file in GCS
func (g *GCSStorage) GetSignedURL(ctx context.Context, bucket, objectName string, expires int64) (string, error) {
	// Not implemented as per requirements
	log.Error().Msg("GetSignedURL method not implemented")
	return "", ErrNotImplemented
}

// StreamUpload uploads a file from a reader to GCS
func (g *GCSStorage) StreamUpload(ctx context.Context, bucket, objectName string, reader io.Reader, contentType string) (string, error) {
	// Not implemented as per requirements
	log.Error().Msg("StreamUpload method not implemented")
	return "", ErrNotImplemented
}
