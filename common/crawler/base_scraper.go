package crawler

import (
	"context"
	"fmt"
	"io"
	"math"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/LexiconIndonesia/crawler-http-service/common/messaging"
	"github.com/LexiconIndonesia/crawler-http-service/common/models"
	"github.com/LexiconIndonesia/crawler-http-service/common/services"
	"github.com/LexiconIndonesia/crawler-http-service/common/storage"
	"github.com/LexiconIndonesia/crawler-http-service/repository"
	"github.com/go-rod/rod"
	"github.com/rs/zerolog/log"
)

func sanitizeTitleForFileName(title string) string {
	nonAlphanumericRegex := regexp.MustCompile(`[^a-zA-Z0-9 ]+`)
	sanitized := nonAlphanumericRegex.ReplaceAllString(title, "_")
	size := math.Min(float64(len(sanitized)), float64(100))
	sanitized = sanitized[:int(size)]
	return strings.ReplaceAll(sanitized, " ", "_")
}

// BaseScraperConfig represents the base configuration for a scraper
type BaseScraperConfig struct {
	DataSource     repository.DataSource
	StorageBucket  string
	RetryAttempts  int
	RetryDelay     time.Duration
	RequestTimeout time.Duration
	MaxConcurrency int
	UserAgent      string
}

// DefaultBaseScraperConfig returns the default configuration for a scraper
func DefaultBaseScraperConfig() BaseScraperConfig {
	return DefaultBaseScraperConfigWithBucket("")
}

// DefaultBaseScraperConfigWithBucket returns the default configuration for a scraper with a specific storage bucket
func DefaultBaseScraperConfigWithBucket(storageBucket string) BaseScraperConfig {

	return BaseScraperConfig{
		StorageBucket:  storageBucket,
		RetryAttempts:  3,
		RetryDelay:     time.Second * 2,
		RequestTimeout: time.Second * 30,
		MaxConcurrency: 5,
		UserAgent:      "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
	}
}

// BaseScraper provides a base implementation of the Scraper interface
type BaseScraper struct {
	Config          BaseScraperConfig
	Browser         *rod.Browser
	MessageBroker   *messaging.NatsBroker
	StorageService  storage.StorageService
	UrlFrontierRepo services.UrlFrontierService
	DataSourceRepo  services.DataSourceService
	ExtractionRepo  services.ExtractionService
}

func (s *BaseScraper) UpdateUrlFrontierStatus(ctx context.Context, id []string, status models.UrlFrontierStatus, errorMessage string) error {
	if s.UrlFrontierRepo == nil {
		return fmt.Errorf("url frontier repository not initialized")
	}

	err := s.UrlFrontierRepo.UpdateStatusBatch(ctx, id, status, errorMessage)
	if err != nil {
		log.Error().Err(err).Str("id", strings.Join(id, ",")).Msg("Failed to update url frontier status")
		return err
	}

	return nil
}

// SaveExtraction saves an extraction to the database
func (s *BaseScraper) SaveExtraction(ctx context.Context, extraction repository.Extraction) (repository.Extraction, error) {
	if s.ExtractionRepo == nil {
		return repository.Extraction{}, fmt.Errorf("extraction repository not initialized")
	}

	if extraction.CreatedAt.IsZero() {
		extraction.CreatedAt = time.Now()
	}
	if extraction.UpdatedAt.IsZero() {
		extraction.UpdatedAt = time.Now()
	}
	if extraction.ExtractionDate.IsZero() {
		extraction.ExtractionDate = time.Now()
	}

	savedExtraction, err := s.ExtractionRepo.Create(ctx, extraction)
	if err != nil {
		log.Error().Err(err).Str("url_frontier_id", extraction.UrlFrontierID).Msg("Failed to save extraction")
		return repository.Extraction{}, err
	}

	log.Debug().Str("id", savedExtraction.ID).Msg("Saved extraction")
	return savedExtraction, nil
}

// SaveExtractionBatch saves multiple extractions to the database
func (s *BaseScraper) SaveExtractionBatch(ctx context.Context, extractions []repository.Extraction) ([]repository.Extraction, error) {
	if s.ExtractionRepo == nil {
		return nil, fmt.Errorf("extraction repository not initialized")
	}

	for i := range extractions {
		if extractions[i].CreatedAt.IsZero() {
			extractions[i].CreatedAt = time.Now()
		}
		if extractions[i].UpdatedAt.IsZero() {
			extractions[i].UpdatedAt = time.Now()
		}
		if extractions[i].ExtractionDate.IsZero() {
			extractions[i].ExtractionDate = time.Now()
		}
	}

	savedExtractions, err := s.ExtractionRepo.CreateBatch(ctx, extractions)
	if err != nil {
		log.Error().Err(err).Int("count", len(extractions)).Msg("Failed to save extractions batch")
		return nil, err
	}

	log.Debug().Int("count", len(savedExtractions)).Msg("Saved extractions batch")
	return savedExtractions, nil
}

// UploadFileToStorage uploads a file to the storage service
func (s *BaseScraper) UploadFileToStorage(ctx context.Context, objectName string, content []byte, contentType string) (string, error) {
	if s.StorageService == nil {
		return "", fmt.Errorf("storage service not initialized")
	}

	objectName, err := s.StorageService.Upload(ctx, s.Config.StorageBucket, objectName, content, contentType)
	if err != nil {
		log.Error().Err(err).Str("bucket", s.Config.StorageBucket).Str("object", objectName).Msg("Failed to upload file to storage")
		return "", err
	}

	log.Debug().Str("object", objectName).Str("bucket", s.Config.StorageBucket).Str("object", objectName).Msg("Uploaded file to storage")
	return objectName, nil
}

// HandlePdf downloads a PDF from a URL and uploads it to the storage service
func (s *BaseScraper) HandlePdf(ctx context.Context, extractionID string, pdfURL string, title string) (models.ExtractionArtifact, error) {
	log.Info().Msgf("Handling pdf for url: %s", pdfURL)

	req, err := http.NewRequestWithContext(ctx, "GET", pdfURL, nil)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create request for PDF download")
		return models.ExtractionArtifact{}, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Error().Err(err).Msg("Failed to download PDF")
		return models.ExtractionArtifact{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		err := fmt.Errorf("bad status downloading pdf: %s", resp.Status)
		log.Error().Err(err).Str("url", pdfURL).Msg("Failed to download PDF")
		return models.ExtractionArtifact{}, err
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Error().Err(err).Msg("Failed to read PDF response body")
		return models.ExtractionArtifact{}, err
	}

	sanitizedTitle := sanitizeTitleForFileName(title)
	fileName := fmt.Sprintf("%s_%s.pdf", extractionID, sanitizedTitle)
	objectName := fmt.Sprintf("%s/%s", s.Config.DataSource.Name, fileName)
	gcsURL, err := s.UploadFileToStorage(ctx, objectName, body, "application/pdf")
	if err != nil {
		log.Error().Err(err).Msg("Failed to upload PDF to storage")
		return models.ExtractionArtifact{}, err
	}
	log.Info().Msgf("Uploaded pdf to GCS: %s", gcsURL)

	artifact := models.ExtractionArtifact{
		FileName:    fileName,
		Size:        int64(len(body)),
		ContentType: "application/pdf",
		URL:         gcsURL,
	}

	return artifact, nil
}

// HandleHtml downloads HTML from a URL and uploads it to the storage service
func (s *BaseScraper) HandleHtml(ctx context.Context, extractionID string, htmlURL string, title string) (models.ExtractionArtifact, error) {
	log.Info().Msgf("Downloading html for url: %s", htmlURL)

	req, err := http.NewRequestWithContext(ctx, "GET", htmlURL, nil)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create request for HTML download")
		return models.ExtractionArtifact{}, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Error().Err(err).Msg("Failed to download HTML")
		return models.ExtractionArtifact{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		err := fmt.Errorf("bad status downloading html: %s", resp.Status)
		log.Error().Err(err).Str("url", htmlURL).Msg("Failed to download HTML")
		return models.ExtractionArtifact{}, err
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Error().Err(err).Msg("Failed to read HTML response body")
		return models.ExtractionArtifact{}, err
	}

	sanitizedTitle := sanitizeTitleForFileName(title)
	fileName := fmt.Sprintf("%s_%s.html", extractionID, sanitizedTitle)
	objectName := fmt.Sprintf("%s/%s", s.Config.DataSource.Name, fileName)
	gcsURL, err := s.UploadFileToStorage(ctx, objectName, body, "text/html")
	if err != nil {
		log.Error().Err(err).Msg("Failed to upload HTML to storage")
		return models.ExtractionArtifact{}, err
	}
	log.Info().Msgf("Uploaded html to GCS: %s", gcsURL)

	artifact := models.ExtractionArtifact{
		FileName:    fileName,
		Size:        int64(len(body)),
		ContentType: "text/html",
		URL:         gcsURL,
	}

	return artifact, nil
}
