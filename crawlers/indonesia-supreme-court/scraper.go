package indonesia_supreme_court

import (
	"context"

	"github.com/LexiconIndonesia/crawler-http-service/common/crawler"
	"github.com/LexiconIndonesia/crawler-http-service/common/models"

	"github.com/LexiconIndonesia/crawler-http-service/repository"
	"github.com/go-rod/rod"
	"github.com/rs/zerolog/log"
)

// MahkamahAgungScraper is a scraper for the Indonesia Supreme Court
type MahkamahAgungScraper struct {
	crawler.BaseScraper
	Config crawler.MahkamahAgungConfig
}

// NewMahkamahAgungScraper creates a new MahkamahAgungScraper
func NewMahkamahAgungScraper(config crawler.MahkamahAgungConfig, baseConfig crawler.BaseScraperConfig, broker crawler.MessageBroker) (*MahkamahAgungScraper, error) {
	// This is just a stub - no actual implementation
	return &MahkamahAgungScraper{
		BaseScraper: crawler.BaseScraper{
			Config:        baseConfig,
			MessageBroker: broker,
		},
		Config: config,
	}, nil
}

// ScrapeAll scrapes all pending URLs for Indonesia Supreme Court
func (s *MahkamahAgungScraper) ScrapeAll(ctx context.Context) error {
	// Not implemented as per requirements
	log.Error().Msg("ScrapeAll method not implemented")
	return crawler.ErrNotImplemented
}

// ScrapeByUrlFrontierID scrapes a specific URL frontier by ID
func (s *MahkamahAgungScraper) ScrapeByUrlFrontierID(ctx context.Context, id string) error {
	// Not implemented as per requirements
	log.Error().Msg("ScrapeByUrlFrontierID method not implemented")
	return crawler.ErrNotImplemented
}

// ExtractElements extracts data from a page
func (s *MahkamahAgungScraper) ExtractElements(ctx context.Context, page *rod.Page) ([]repository.Extraction, error) {
	// Not implemented as per requirements
	log.Error().Msg("ExtractElements method not implemented")
	return nil, crawler.ErrNotImplemented
}

// ExtractArtifactsFromPage extracts and downloads artifacts from a page
func (s *MahkamahAgungScraper) ExtractArtifactsFromPage(ctx context.Context, page *rod.Page) ([]models.ExtractionArtifact, error) {
	// Not implemented as per requirements
	log.Error().Msg("ExtractArtifactsFromPage method not implemented")
	return nil, crawler.ErrNotImplemented
}
