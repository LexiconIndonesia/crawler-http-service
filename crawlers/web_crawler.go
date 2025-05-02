package crawler

import (
	"context"
	"errors"
	"fmt"
	"time"

	"encoding/json"
	"strings"

	"github.com/adryanev/go-http-service-template/common/messaging"
	"github.com/adryanev/go-http-service-template/repository"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"github.com/samber/lo"
)

// Item represents a generic data item extracted by a crawler
type Item struct {
	URL      string
	Title    string
	Content  string
	Metadata map[string]interface{}
}

// BrowserOptions contains options for browser initialization
type BrowserOptions struct {
	Headless      bool
	Timeout       time.Duration
	WaitAfterLoad time.Duration
}

// ConsumeMessage defines the structure for messages consumed by crawlers
type ConsumeMessage struct {
	MessageID string          `json:"message_id"`
	Type      string          `json:"type"`
	Payload   json.RawMessage `json:"payload"`
}

// BaseCrawler provides common functionality for all web crawlers
type BaseCrawler struct {
	Config      *CrawlerConfig
	Browser     *rod.Browser
	Service     CrawlerService
	BrowserOpts BrowserOptions
}

// NewBaseCrawler creates a new base crawler with the given configuration
func NewBaseCrawler(config *CrawlerConfig, service CrawlerService) *BaseCrawler {
	return &BaseCrawler{
		Config:  config,
		Service: service,
		BrowserOpts: BrowserOptions{
			Headless:      true,
			Timeout:       30 * time.Second,
			WaitAfterLoad: 2 * time.Second,
		},
	}
}

// Setup initializes the browser
func (c *BaseCrawler) Setup(ctx context.Context) error {
	log.Info().Str("crawler", c.Config.Name).Msg("Setting up browser")

	// Initialize browser
	url := launcher.New().
		Headless(c.BrowserOpts.Headless).
		MustLaunch()

	browser := rod.New().ControlURL(url).MustConnect()
	c.Browser = browser

	log.Info().Str("crawler", c.Config.Name).Msg("Browser setup complete")
	return nil
}

// Teardown cleans up resources
func (c *BaseCrawler) Teardown(ctx context.Context) error {
	log.Info().Str("crawler", c.Config.Name).Msg("Tearing down browser")

	if c.Browser != nil {
		c.Browser.MustClose()
	}

	log.Info().Str("crawler", c.Config.Name).Msg("Browser teardown complete")
	return nil
}

// Navigate opens a page and navigates to the specified URL
func (c *BaseCrawler) Navigate(ctx context.Context, url string) (*rod.Page, error) {
	log.Info().Str("crawler", c.Config.Name).Str("url", url).Msg("Navigating to URL")

	if c.Browser == nil {
		if err := c.Setup(ctx); err != nil {
			return nil, fmt.Errorf("failed to set up browser: %w", err)
		}
	}

	page := c.Browser.MustPage(url)

	// Set timeout for page operations
	page = page.Context(ctx)

	// Wait for page to load
	err := page.WaitLoad()
	if err != nil {
		return nil, fmt.Errorf("failed to wait for page load: %w", err)
	}

	// Additional wait to ensure dynamic content is loaded
	if c.BrowserOpts.WaitAfterLoad > 0 {
		time.Sleep(c.BrowserOpts.WaitAfterLoad)
	}

	return page, nil
}

// CrawlAll crawls all available URLs
func (bc *BaseCrawler) CrawlAll(ctx context.Context) error {
	// Implementation depends on how URLs are stored and retrieved
	// This is a basic implementation that can be extended

	limit := int32(100) // Process in batches
	frontiers, err := bc.Service.GetUnscrappedUrlFrontiers(ctx, limit)
	if err != nil {
		return fmt.Errorf("failed to get unscrapped frontiers: %w", err)
	}

	for _, frontier := range frontiers {
		err := bc.CrawlByURL(ctx, frontier.Url)
		if err != nil {
			log.Error().
				Str("crawler", bc.Config.Name).
				Str("url", frontier.Url).
				Err(err).
				Msg("Failed to crawl URL")

			// Update status to error
			bc.Service.UpdateFrontierStatuses(ctx, []lo.Tuple2[string, int16]{
				{frontier.ID, int16(URL_FRONTIER_STATUS_ERROR)},
			})
		}
	}

	return nil
}

// CrawlByKeyword crawls URLs related to the given keyword
func (bc *BaseCrawler) CrawlByKeyword(ctx context.Context, keyword string) error {
	// Implementation would depend on how keywords are stored and linked to URLs
	// This is a placeholder that would need to be customized
	log.Info().
		Str("crawler", bc.Config.Name).
		Str("keyword", keyword).
		Msg("Crawling by keyword")

	// Actual implementation would be specific to the application
	return errors.New("CrawlByKeyword not implemented")
}

// CrawlByURL crawls a specific URL
func (c *BaseCrawler) CrawlByURL(ctx context.Context, url string) error {
	log.Info().Str("crawler", c.Config.Name).Str("url", url).Msg("Crawling URL")

	page, err := c.Navigate(ctx, url)
	if err != nil {
		return fmt.Errorf("failed to navigate to URL: %w", err)
	}
	defer page.Close()

	items, err := c.ExtractElements(ctx, page)
	if err != nil {
		return fmt.Errorf("failed to extract elements: %w", err)
	}

	log.Info().Str("crawler", c.Config.Name).Int("items_count", len(items)).Msg("Extracted items")

	// Get or create URL frontier
	frontier, err := c.Service.GetUrlFrontierByUrl(ctx, url)
	if err != nil {
		// Create a new frontier if it doesn't exist
		id := uuid.New().String()
		frontier = repository.UrlFrontier{
			ID:           id,
			DataSourceID: c.Config.DataSource.ID,
			Domain:       getDomain(url),
			Url:          url,
			Status:       1, // Crawled
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}

		if err := c.Service.UpsertUrlFrontier(ctx, frontier); err != nil {
			return fmt.Errorf("failed to create URL frontier: %w", err)
		}
	} else {
		// Update existing frontier
		frontier.Status = 1 // Crawled
		frontier.UpdatedAt = time.Now()

		if err := c.Service.UpsertUrlFrontier(ctx, frontier); err != nil {
			return fmt.Errorf("failed to update URL frontier: %w", err)
		}
	}

	// Add raw page content to GCS
	htmlContent, err := page.HTML()
	if err != nil {
		log.Error().Err(err).Str("url", url).Msg("Failed to get HTML content")
	} else {
		// Save HTML to GCS and update metadata
		rawPageURL, err := c.saveRawPage(ctx, url, htmlContent)
		if err != nil {
			log.Error().Err(err).Str("url", url).Msg("Failed to save raw page")
		} else {
			// Update metadata with raw page link
			metadata := map[string]interface{}{
				"raw_page_link": rawPageURL,
			}
			metadataJSON, _ := json.Marshal(metadata)
			frontier.Metadata = metadataJSON

			if err := c.Service.UpsertUrlFrontier(ctx, frontier); err != nil {
				log.Error().Err(err).Str("url", url).Msg("Failed to update URL frontier with metadata")
			}
		}
	}

	// Emit extract.run message
	extractMsg := messaging.ExtractRunMessage{
		URLFrontierID: frontier.ID,
		DataSourceID:  c.Config.DataSource.ID,
		JobID:         uuid.New().String(),
	}

	// Marshal the message
	msgData, err := json.Marshal(extractMsg)
	if err != nil {
		log.Error().Err(err).Str("url", url).Msg("Failed to marshal extract.run message")
		return fmt.Errorf("failed to marshal extract.run message: %w", err)
	}

	// Get NATS client via service
	natsClient := c.Service.GetNatsClient()
	if natsClient == nil {
		log.Error().Str("url", url).Msg("NATS client is nil, cannot publish extract.run message")
		return fmt.Errorf("NATS client is nil")
	}

	// Publish to extract.run subject
	ack, err := natsClient.PublishAsync(messaging.SubjectExtractRun, msgData)
	if err != nil {
		log.Error().Err(err).Str("url", url).Msg("Failed to publish extract.run message")
		return fmt.Errorf("failed to publish extract.run message: %w", err)
	}

	// Wait for acknowledgement
	select {
	case <-ack.Ok():
		log.Info().Str("url", url).Str("job_id", extractMsg.JobID).Msg("Published extract.run message")
	case err := <-ack.Err():
		log.Error().Err(err).Str("url", url).Msg("Failed to get acknowledgement for extract.run message")
		return fmt.Errorf("failed to get acknowledgement for extract.run message: %w", err)
	case <-ctx.Done():
		log.Warn().Str("url", url).Msg("Context cancelled while waiting for acknowledgement")
		return ctx.Err()
	}

	return nil
}

// saveRawPage saves the raw HTML content to storage
func (c *BaseCrawler) saveRawPage(ctx context.Context, url string, htmlContent string) (string, error) {
	// Implementation would depend on your storage solution
	// This is a placeholder that would save to GCS in a real implementation

	// Generate a filename based on URL
	filename := fmt.Sprintf("%s.html", generateFilename(url))
	path := fmt.Sprintf("%s/%s", c.Config.GetGCSHTMLFolder(), filename)

	// In a real implementation, save to GCS
	// For now, we'll just log it
	log.Info().
		Str("crawler", c.Config.Name).
		Str("url", url).
		Str("path", path).
		Msg("Would save raw HTML to GCS")

	// Return the GCS URL
	return fmt.Sprintf("gs://%s/%s", c.Config.GCSBucket, path), nil
}

// generateFilename creates a filename from a URL
func generateFilename(url string) string {
	// Remove protocol and replace special characters
	url = strings.TrimPrefix(url, "http://")
	url = strings.TrimPrefix(url, "https://")
	url = strings.ReplaceAll(url, "/", "_")
	url = strings.ReplaceAll(url, ":", "_")
	url = strings.ReplaceAll(url, "?", "_")
	url = strings.ReplaceAll(url, "&", "_")
	url = strings.ReplaceAll(url, "=", "_")

	// Add a timestamp to ensure uniqueness
	return fmt.Sprintf("%s_%d", url, time.Now().Unix())
}

// getDomain extracts the domain from a URL
func getDomain(url string) string {
	// Simple implementation to extract domain
	url = strings.TrimPrefix(url, "http://")
	url = strings.TrimPrefix(url, "https://")
	parts := strings.Split(url, "/")
	if len(parts) > 0 {
		return parts[0]
	}
	return url
}

// ExtractMsg extracts data from raw content and sends it to the extract service
func (c *BaseCrawler) Consume(ctx context.Context, message []byte) error {
	// Implement message consumption logic here
	// This is just a placeholder implementation
	// Parse the message

	// Example:
	var consumeMsg ConsumeMessage
	if err := json.Unmarshal(message, &consumeMsg); err != nil {
		return fmt.Errorf("failed to unmarshal consume message: %w", err)
	}

	// Process the message
	log.Info().
		Str("crawler", c.Config.Name).
		Str("message_id", consumeMsg.MessageID).
		Msg("Processing consume message")

	// Extract the JSON payload
	if consumeMsg.Type == "extract" {
		var extractMsg messaging.ExtractRunMessage
		if err := json.Unmarshal(consumeMsg.Payload, &extractMsg); err != nil {
			return fmt.Errorf("failed to unmarshal extract message: %w", err)
		}

		// Fill in data source if not provided
		if extractMsg.DataSourceID == "" {
			extractMsg.DataSourceID = c.Config.DataSource.ID
		}

		// Process the extraction
		// ...
	}

	return nil
}

// ExtractElements is a stub that must be implemented by concrete crawlers
// This method should not be called directly on BaseCrawler
func (bc *BaseCrawler) ExtractElements(ctx context.Context, page *rod.Page) ([]Item, error) {
	return nil, errors.New("ExtractElements must be implemented by concrete crawler")
}
