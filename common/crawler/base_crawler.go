package crawler

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/url"
	"time"

	"github.com/LexiconIndonesia/crawler-http-service/common/messaging"
	"github.com/LexiconIndonesia/crawler-http-service/common/models"
	"github.com/LexiconIndonesia/crawler-http-service/common/services"
	"github.com/LexiconIndonesia/crawler-http-service/common/storage"
	"github.com/LexiconIndonesia/crawler-http-service/repository"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/rs/zerolog/log"
)

// BaseCrawlerConfig represents the base configuration for a crawler
type BaseCrawlerConfig struct {
	DataSource     repository.DataSource
	RetryAttempts  int
	RetryDelay     time.Duration
	RequestTimeout time.Duration
	MaxConcurrency int
	UserAgent      string
}

// DefaultBaseCrawlerConfig returns the default configuration for a crawler
func DefaultBaseCrawlerConfig() BaseCrawlerConfig {
	return BaseCrawlerConfig{
		RetryAttempts:  3,
		RetryDelay:     time.Second * 2,
		RequestTimeout: time.Second * 30,
		MaxConcurrency: 1,
		UserAgent:      "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
	}
}

// BaseCrawler provides a foundation for all crawlers
type BaseCrawler struct {
	Config          BaseCrawlerConfig
	MessageBroker   *messaging.NatsBroker
	UrlFrontierRepo services.UrlFrontierService
	ExtractionRepo  services.ExtractionService
	DataSourceRepo  services.DataSourceService
	StorageService  storage.StorageService
}

// SaveUrlFrontier saves a URL frontier to the database
func (c *BaseCrawler) SaveUrlFrontier(ctx context.Context, frontier repository.UrlFrontier) (repository.UrlFrontier, error) {
	if c.UrlFrontierRepo == nil {
		return repository.UrlFrontier{}, fmt.Errorf("URL frontier repository not initialized")
	}

	if frontier.CreatedAt.IsZero() {
		frontier.CreatedAt = time.Now()
	}
	if frontier.DataSourceID == "" {
		frontier.DataSourceID = c.Config.DataSource.ID
	}

	// Parse domain from URL if not set
	if frontier.Domain == "" && frontier.Url != "" {
		parsedURL, err := url.Parse(frontier.Url)
		if err == nil {
			frontier.Domain = parsedURL.Host
		}
	}

	// Save to database
	savedFrontier, err := c.UrlFrontierRepo.Create(ctx, frontier)
	if err != nil {
		log.Error().Err(err).Str("url", frontier.Url).Msg("Failed to save URL frontier")
		return repository.UrlFrontier{}, err
	}

	log.Debug().Str("id", savedFrontier.ID).Str("url", savedFrontier.Url).Msg("Saved URL frontier")
	return savedFrontier, nil
}

// SaveUrlFrontierBatch saves multiple URL frontiers to the database in a batch
func (c *BaseCrawler) SaveUrlFrontierBatch(ctx context.Context, frontiers []repository.UrlFrontier) ([]repository.UrlFrontier, error) {
	if c.UrlFrontierRepo == nil {
		return nil, fmt.Errorf("URL frontier repository not initialized")
	}

	// Fill in standard fields for each frontier
	for i := range frontiers {

		if frontiers[i].CreatedAt.IsZero() {
			frontiers[i].CreatedAt = time.Now()
		}
		if frontiers[i].DataSourceID == "" {
			frontiers[i].DataSourceID = c.Config.DataSource.ID
		}
		if frontiers[i].Domain == "" && frontiers[i].Url != "" {
			parsedURL, err := url.Parse(frontiers[i].Url)
			if err == nil {
				frontiers[i].Domain = parsedURL.Host
			}
		}
	}

	// Save to database in batch
	savedFrontiers, err := c.UrlFrontierRepo.CreateBatch(ctx, frontiers)
	if err != nil {
		log.Error().Err(err).Int("count", len(frontiers)).Msg("Failed to save URL frontiers batch")
		return nil, err
	}

	log.Debug().Int("count", len(savedFrontiers)).Msg("Saved URL frontiers batch")
	return savedFrontiers, nil
}

// UploadFileToStorage uploads a file to the storage service
func (c *BaseCrawler) UploadFileToStorage(ctx context.Context, bucket, objectName string, content []byte, contentType string) (string, error) {
	if c.StorageService == nil {
		return "", fmt.Errorf("storage service not initialized")
	}

	// Upload to storage
	url, err := c.StorageService.Upload(ctx, bucket, objectName, content, contentType)
	if err != nil {
		log.Error().Err(err).Str("bucket", bucket).Str("object", objectName).Msg("Failed to upload file to storage")
		return "", err
	}

	log.Debug().Str("url", url).Str("bucket", bucket).Str("object", objectName).Msg("Uploaded file to storage")
	return url, nil
}

// UpdateUrlFrontierStatus updates the status of a URL frontier
func (c *BaseCrawler) UpdateUrlFrontierStatus(ctx context.Context, id string, status models.UrlFrontierStatus, errorMessage string) error {
	if c.UrlFrontierRepo == nil {
		return fmt.Errorf("URL frontier repository not initialized")
	}

	// Update status
	if err := c.UrlFrontierRepo.UpdateStatus(ctx, id, status, errorMessage); err != nil {
		log.Error().Err(err).Str("id", id).Int("status", int(status)).Msg("Failed to update URL frontier status")
		return err
	}

	log.Debug().Str("id", id).Int("status", int(status)).Msg("Updated URL frontier status")
	return nil
}

// WithRetry executes a function with retry logic
func (c *BaseCrawler) WithRetry(ctx context.Context, operation func() error) error {
	var lastErr error

	for attempt := 0; attempt < c.Config.RetryAttempts; attempt++ {
		if attempt > 0 {
			log.Debug().Int("attempt", attempt+1).Msg("Retrying operation")
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(c.Config.RetryDelay):
				// Wait before retry
			}
		}

		if err := operation(); err != nil {
			lastErr = err
			log.Debug().Err(err).Int("attempt", attempt+1).Msg("Operation failed, will retry")
			continue
		}

		return nil // Success
	}

	return fmt.Errorf("operation failed after %d attempts: %w", c.Config.RetryAttempts, lastErr)
}

// GenerateID generates a unique ID
func (c *BaseCrawler) GenerateID(url string) string {
	// generate id by hashing the url with sha256 in hex format
	hash := sha256.Sum256([]byte(url))
	return hex.EncodeToString(hash[:])
}

// WithHeartbeat executes a long-running operation with periodic heartbeating to JetStream
// This prevents the message from timing out during extended processing
func (c *BaseCrawler) WithHeartbeat(ctx context.Context, msg jetstream.Msg, operation func(ctx context.Context) error, heartbeatInterval time.Duration) error {
	if heartbeatInterval <= 0 {
		heartbeatInterval = 30 * time.Second // Default to 30 seconds
	}

	// Create a context for the heartbeating goroutine
	heartbeatCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Channel to signal when the operation is complete
	done := make(chan struct{})
	var operationErr error

	// Start heartbeating in a separate goroutine
	go func() {
		ticker := time.NewTicker(heartbeatInterval)
		defer ticker.Stop()

		for {
			select {
			case <-heartbeatCtx.Done():
				return
			case <-done:
				return
			case <-ticker.C:
				if err := msg.InProgress(); err != nil {
					log.Error().Err(err).Msg("Failed to send heartbeat")
					// Continue anyway, as the heartbeat failure shouldn't stop processing
				} else {
					log.Debug().Msg("Sent heartbeat to JetStream")
				}
			}
		}
	}()

	// Execute the operation
	go func() {
		defer close(done)
		operationErr = operation(ctx)
	}()

	// Wait for either completion or context cancellation
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-done:
		return operationErr
	}
}
