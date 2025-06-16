package storage

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"cloud.google.com/go/storage"
	"github.com/rs/zerolog/log"
	"google.golang.org/api/option"
)

// Errors
var (
	StorageClient StorageService
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
func NewGCSStorage(ctx context.Context, config GCSConfig) (StorageService, error) {
	storageClient, err := storage.NewClient(ctx, option.WithCredentialsFile(config.CredentialsFile))
	if err != nil {
		return nil, err
	}
	return &GCSStorage{
		config: config,
		client: storageClient,
	}, nil
}
func SetStorageClient(storageClient StorageService) error {
	if storageClient == nil {
		log.Error().Msg("Storage client is nil")
		return errors.New("storage client is nil")
	}
	StorageClient = storageClient
	return nil
}

// Upload uploads a file to GCS and returns the object name
func (g *GCSStorage) Upload(ctx context.Context, bucket, objectName string, content []byte, contentType string) (string, error) {
	return g.StreamUpload(ctx, bucket, objectName, bytes.NewReader(content), contentType)
}

// Download downloads a file from GCS
func (g *GCSStorage) Download(ctx context.Context, bucket, objectName string) ([]byte, error) {
	rc, err := g.client.Bucket(bucket).Object(objectName).NewReader(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create reader for object %s in bucket %s: %w", objectName, bucket, err)
	}
	defer rc.Close()

	data, err := io.ReadAll(rc)
	if err != nil {
		return nil, fmt.Errorf("failed to read data for object %s in bucket %s: %w", objectName, bucket, err)
	}
	return data, nil
}

// Delete deletes a file from GCS
func (g *GCSStorage) Delete(ctx context.Context, bucket, objectName string) error {
	if err := g.client.Bucket(bucket).Object(objectName).Delete(ctx); err != nil {
		return fmt.Errorf("failed to delete object %s from bucket %s: %w", objectName, bucket, err)
	}
	return nil
}

// GetSignedURL gets a signed URL for a file in GCS
func (g *GCSStorage) GetSignedURL(ctx context.Context, bucket, objectName string, expires int64) (string, error) {
	type credentials struct {
		PrivateKey  string `json:"private_key"`
		ClientEmail string `json:"client_email"`
	}

	credsFile, err := os.ReadFile(g.config.CredentialsFile)
	if err != nil {
		return "", fmt.Errorf("failed to read credentials file: %w", err)
	}

	var creds credentials
	if err := json.Unmarshal(credsFile, &creds); err != nil {
		return "", fmt.Errorf("failed to unmarshal credentials: %w", err)
	}

	opts := &storage.SignedURLOptions{
		GoogleAccessID: creds.ClientEmail,
		PrivateKey:     []byte(creds.PrivateKey),
		Method:         "GET",
		Expires:        time.Now().Add(time.Second * time.Duration(expires)),
	}

	u, err := storage.SignedURL(bucket, objectName, opts)
	if err != nil {
		return "", fmt.Errorf("failed to sign URL: %w", err)
	}
	return u, nil
}

// StreamUpload uploads a file from a reader to GCS and returns the object name.
func (g *GCSStorage) StreamUpload(ctx context.Context, bucket, objectName string, reader io.Reader, contentType string) (string, error) {
	wc := g.client.Bucket(bucket).Object(objectName).NewWriter(ctx)
	wc.ContentType = contentType
	wc.CacheControl = "public, max-age=86400"

	if _, err := io.Copy(wc, reader); err != nil {
		return "", fmt.Errorf("failed to upload file: %w", err)
	}

	if err := wc.Close(); err != nil {
		return "", fmt.Errorf("failed to close writer: %w", err)
	}

	return objectName, nil
}
