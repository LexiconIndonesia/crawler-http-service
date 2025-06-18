package isc

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/LexiconIndonesia/crawler-http-service/common"
	"github.com/LexiconIndonesia/crawler-http-service/common/constants"
	"github.com/LexiconIndonesia/crawler-http-service/common/crawler"
	"github.com/LexiconIndonesia/crawler-http-service/common/db"
	"github.com/LexiconIndonesia/crawler-http-service/common/messaging"
	"github.com/LexiconIndonesia/crawler-http-service/common/services"
	"github.com/LexiconIndonesia/crawler-http-service/common/storage"
	"github.com/LexiconIndonesia/crawler-http-service/common/work"

	"github.com/LexiconIndonesia/crawler-http-service/repository"
	"github.com/go-rod/rod"
	"github.com/rs/zerolog/log"
)

// IndonesiaSupremeCourtScraper is a scraper for the Indonesia Supreme Court
type IndonesiaSupremeCourtScraper struct {
	crawler.BaseScraper
	Config      crawler.IndonesiaSupremeCourtConfig
	browser     *rod.Browser
	workManager *work.WorkManager
}

// NewIndonesiaSupremeCourtScraper creates a new IndonesiaSupremeCourtScraper
func NewIndonesiaSupremeCourtScraper(db *db.DB, config crawler.IndonesiaSupremeCourtConfig, baseConfig crawler.BaseScraperConfig, broker *messaging.NatsBroker) (*IndonesiaSupremeCourtScraper, error) {
	// This is just a stub - no actual implementation
	return &IndonesiaSupremeCourtScraper{
		BaseScraper: crawler.BaseScraper{
			Config:          baseConfig,
			MessageBroker:   broker,
			UrlFrontierRepo: services.NewUrlFrontierRepository(db.Queries),
			ExtractionRepo:  services.NewExtractionRepository(db.Queries),
			DataSourceRepo:  services.NewDataSourceRepository(db.Queries),
			StorageService:  storage.StorageClient,
		},
		Config:      config,
		workManager: work.NewWorkManager(db),
	}, nil
}

// Setup initializes the scraper
func (s *IndonesiaSupremeCourtScraper) Setup(ctx context.Context) error {
	browser := rod.New()

	err := browser.Connect()
	if err != nil {
		return err
	}

	s.browser = browser

	return nil
}

// Teardown cleans up resources
func (s *IndonesiaSupremeCourtScraper) Teardown(ctx context.Context) error {
	err := s.browser.Close()
	if err != nil {
		return err
	}
	return nil
}

// ScrapeAll scrapes all pending URLs for Singapore Supreme Court
func (s *IndonesiaSupremeCourtScraper) ScrapeAll(ctx context.Context, jobID string) error {
	// Not implemented as per requirements
	log.Error().Msg("ScrapeAll method not implemented")
	return common.ErrNotImplemented
}

// ScrapeByURLFrontierID scrapes a specific URL frontier by ID
func (s *IndonesiaSupremeCourtScraper) ScrapeByURLFrontierID(ctx context.Context, id string, jobID string) error {
	// Not implemented as per requirements
	log.Error().Msg("ScrapeByUrlFrontierID method not implemented")
	return common.ErrNotImplemented
}

// Consume processes a message from a queue
func (s *IndonesiaSupremeCourtScraper) Consume(ctx context.Context, message []byte) error {
	log.Info().Msg("Processing Indonesia Supreme Court message from queue")

	var req messaging.ScrapeRequest
	if err := json.Unmarshal(message, &req); err != nil {
		log.Error().Err(err).Msg("Failed to unmarshal scrape request")
		return err
	}

	switch req.Type {
	case constants.ScrapeAllAction:
		return s.ScrapeAll(ctx, req.ID)
	case constants.ScrapeByIDAction:
		if req.Payload.URLFrontierID == "" {
			err := fmt.Errorf("URLFrontierID is required for action %s", req.Type)
			log.Error().Err(err).Msg("Invalid scrape request")
			return err
		}
		return s.ScrapeByURLFrontierID(ctx, req.Payload.URLFrontierID, req.ID)
	default:
		err := fmt.Errorf("unsupported action type: %s", req.Type)
		log.Error().Err(err).Msg("Unsupported scrape request")
		return err
	}
}

func (s *IndonesiaSupremeCourtScraper) ScrapePage(ctx context.Context, page *rod.Page, url repository.UrlFrontier, jobID string) (repository.Extraction, error) {
	log.Info().Str("url", url.Url).Msg("Scraping Singapore Supreme Court page")

	return repository.Extraction{}, common.ErrNotImplemented
}
