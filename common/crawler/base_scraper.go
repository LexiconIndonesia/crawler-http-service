package crawler

import (
	"context"
	"time"

	"github.com/LexiconIndonesia/crawler-http-service/common/messaging"
	"github.com/LexiconIndonesia/crawler-http-service/common/models"
	"github.com/LexiconIndonesia/crawler-http-service/repository"
	"github.com/go-rod/rod"
	"github.com/rs/zerolog/log"
)

// BaseScraperConfig represents the base configuration for a scraper
type BaseScraperConfig struct {
	DataSourceID   string
	StorageBucket  string
	RetryAttempts  int
	RetryDelay     time.Duration
	RequestTimeout time.Duration
	MaxConcurrency int
	UserAgent      string
}

// DefaultBaseScraperConfig returns the default configuration for a scraper
func DefaultBaseScraperConfig() BaseScraperConfig {
	return BaseScraperConfig{
		RetryAttempts:  3,
		RetryDelay:     time.Second * 2,
		RequestTimeout: time.Second * 30,
		MaxConcurrency: 5,
		UserAgent:      "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
	}
}

// BaseScraper provides a base implementation of the Scraper interface
type BaseScraper struct {
	Config        BaseScraperConfig
	Browser       *rod.Browser
	MessageBroker messaging.MessageBroker
}

// Setup initializes the scraper
func (s *BaseScraper) Setup(ctx context.Context) error {
	// Not implemented as per requirements
	log.Error().Msg("Setup method not implemented")
	return ErrNotImplemented
}

// Teardown cleans up resources used by the scraper
func (s *BaseScraper) Teardown(ctx context.Context) error {
	// Not implemented as per requirements
	log.Error().Msg("Teardown method not implemented")
	return ErrNotImplemented
}

// ScrapeAll scrapes all pending URLs for a data source
func (s *BaseScraper) ScrapeAll(ctx context.Context) error {
	// Not implemented as per requirements
	log.Error().Msg("ScrapeAll method not implemented")
	return ErrNotImplemented
}

// ScrapeByUrlFrontierID scrapes a specific URL frontier by ID
func (s *BaseScraper) ScrapeByUrlFrontierID(ctx context.Context, id string) error {
	// Not implemented as per requirements
	log.Error().Msg("ScrapeByUrlFrontierID method not implemented")
	return ErrNotImplemented
}

// Consume processes a message from a queue
func (s *BaseScraper) Consume(ctx context.Context, message []byte) error {
	// Not implemented as per requirements
	log.Error().Msg("Consume method not implemented")
	return ErrNotImplemented
}

// ExtractElements extracts data from a page
func (s *BaseScraper) ExtractElements(ctx context.Context, page *rod.Page) ([]repository.Extraction, error) {
	// Not implemented as per requirements
	log.Error().Msg("ExtractElements method not implemented")
	return nil, ErrNotImplemented
}

// ExtractArtifactsFromPage extracts and downloads artifacts from a page
func (s *BaseScraper) ExtractArtifactsFromPage(ctx context.Context, page *rod.Page) ([]models.ExtractionArtifact, error) {
	// Not implemented as per requirements
	log.Error().Msg("ExtractArtifactsFromPage method not implemented")
	return nil, ErrNotImplemented
}

// Navigate navigates to a URL and returns the page
func (s *BaseScraper) Navigate(ctx context.Context, url string) (*rod.Page, error) {
	// Not implemented as per requirements
	log.Error().Msg("Navigate method not implemented")
	return nil, ErrNotImplemented
}
