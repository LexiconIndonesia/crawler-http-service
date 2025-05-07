package crawler

import (
	"context"
	"time"

	"github.com/LexiconIndonesia/crawler-http-service/repository"
	"github.com/go-rod/rod"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

// BaseCrawlerConfig represents the base configuration for a crawler
type BaseCrawlerConfig struct {
	DataSourceID   string
	BrowserFlags   []string
	ProxyURL       string
	RetryAttempts  int
	RetryDelay     time.Duration
	RequestTimeout time.Duration
	MaxConcurrency int
	UserAgent      string
}

// DefaultBaseCrawlerConfig returns the default configuration for a crawler
func DefaultBaseCrawlerConfig() BaseCrawlerConfig {
	return BaseCrawlerConfig{
		BrowserFlags:   []string{"--no-sandbox", "--disable-setuid-sandbox", "--disable-gpu"},
		RetryAttempts:  3,
		RetryDelay:     time.Second * 2,
		RequestTimeout: time.Second * 30,
		MaxConcurrency: 5,
		UserAgent:      "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
	}
}

// BaseCrawler provides a base implementation of the Crawler interface
type BaseCrawler struct {
	Config        BaseCrawlerConfig
	Browser       *rod.Browser
	MessageBroker MessageBroker
}

// Setup initializes the crawler
func (c *BaseCrawler) Setup(ctx context.Context) error {
	// Not implemented as per requirements
	log.Error().Msg("Setup method not implemented")
	return ErrNotImplemented
}

// Teardown cleans up resources used by the crawler
func (c *BaseCrawler) Teardown(ctx context.Context) error {
	// Not implemented as per requirements
	log.Error().Msg("Teardown method not implemented")
	return ErrNotImplemented
}

// CrawlAll crawls all pages from the configured data source
func (c *BaseCrawler) CrawlAll(ctx context.Context) error {
	// Not implemented as per requirements
	log.Error().Msg("CrawlAll method not implemented")
	return ErrNotImplemented
}

// CrawlByKeyword crawls pages based on a search term
func (c *BaseCrawler) CrawlByKeyword(ctx context.Context, keyword string) error {
	// Not implemented as per requirements
	log.Error().Msg("CrawlByKeyword method not implemented")
	return ErrNotImplemented
}

// CrawlByURL crawls a specific URL
func (c *BaseCrawler) CrawlByURL(ctx context.Context, url string) error {
	// Not implemented as per requirements
	log.Error().Msg("CrawlByURL method not implemented")
	return ErrNotImplemented
}

// Consume processes a message from a queue
func (c *BaseCrawler) Consume(ctx context.Context, message []byte) error {
	// Not implemented as per requirements
	log.Error().Msg("Consume method not implemented")
	return ErrNotImplemented
}

// ExtractElements extracts URL frontiers from a page
func (c *BaseCrawler) ExtractElements(ctx context.Context, page *rod.Page) ([]repository.UrlFrontier, error) {
	// Not implemented as per requirements
	log.Error().Msg("ExtractElements method not implemented")
	return nil, ErrNotImplemented
}

// Navigate navigates to a URL and returns the page
func (c *BaseCrawler) Navigate(ctx context.Context, url string) (*rod.Page, error) {
	// Not implemented as per requirements
	log.Error().Msg("Navigate method not implemented")
	return nil, ErrNotImplemented
}

// GenerateID generates a unique ID
func (c *BaseCrawler) GenerateID() string {
	return uuid.New().String()
}
