package ssc

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"github.com/LexiconIndonesia/crawler-http-service/common/constants"
	"github.com/LexiconIndonesia/crawler-http-service/common/crawler"
	"github.com/LexiconIndonesia/crawler-http-service/common/db"
	"github.com/LexiconIndonesia/crawler-http-service/common/messaging"
	"github.com/LexiconIndonesia/crawler-http-service/common/models"
	"github.com/LexiconIndonesia/crawler-http-service/common/services"
	"github.com/LexiconIndonesia/crawler-http-service/common/storage"
	"github.com/LexiconIndonesia/crawler-http-service/common/work"
	"github.com/LexiconIndonesia/crawler-http-service/repository"
	"github.com/go-rod/rod"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rs/zerolog/log"
)

type SingaporeSupremeCourtScraper struct {
	crawler.BaseScraper
	Config      crawler.SingaporeSupremeCourtConfig
	browser     *rod.Browser
	workManager *work.WorkManager
}

// NewSingaporeSupremeCourtScraper creates a new SingaporeSupremeCourtScraper
func NewSingaporeSupremeCourtScraper(db *db.DB, config crawler.SingaporeSupremeCourtConfig, baseConfig crawler.BaseScraperConfig, broker *messaging.NatsBroker) (*SingaporeSupremeCourtScraper, error) {
	// This is just a stub - no actual implementation
	return &SingaporeSupremeCourtScraper{
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
func (s *SingaporeSupremeCourtScraper) Setup(ctx context.Context) error {

	browser := rod.New()

	err := browser.Connect()
	if err != nil {
		return err
	}

	s.browser = browser

	return nil
}

// Teardown cleans up resources
func (s *SingaporeSupremeCourtScraper) Teardown(ctx context.Context) error {
	err := s.browser.Close()
	if err != nil {
		return err
	}
	return nil
}

// scrapeTask wraps a single URL frontier scraping job so it can be executed
// inside the generic worker pool. It implements work.Executor[repository.Extraction].
type scrapeTask struct {
	scraper  *SingaporeSupremeCourtScraper
	pagePool *rod.Pool[rod.Page]
	frontier repository.UrlFrontier
	jobID    string
}

func (t *scrapeTask) ExecutorID() string {
	return t.frontier.ID
}

// Execute performs the actual scraping work for a single frontier and returns
// the produced repository.Extraction (if any).
func (t *scrapeTask) Execute(ctx context.Context) (repository.Extraction, error) {
	// Update status to processing
	_ = t.scraper.UpdateUrlFrontierStatus(ctx, []string{t.frontier.ID}, models.UrlFrontierStatusProcessing, "")

	if t.scraper.Config.MinDelayMs > 0 && t.scraper.Config.MaxDelayMs > 0 {
		delay := rand.Intn(t.scraper.Config.MaxDelayMs-t.scraper.Config.MinDelayMs) + t.scraper.Config.MinDelayMs
		log.Debug().Int("ms", delay).Msg("Delaying request")
		time.Sleep(time.Duration(delay) * time.Millisecond)
	}

	extraction, err := t.scraper.scrapeJob(t.pagePool, ctx, t.frontier, t.jobID)
	if err != nil {
		// Mark as failed
		_ = t.scraper.UpdateUrlFrontierStatus(ctx, []string{t.frontier.ID}, models.UrlFrontierStatusFailed, err.Error())
		return repository.Extraction{}, err
	}

	// Mark as crawled on success (ignore error so we still return success)
	_ = t.scraper.UpdateUrlFrontierStatus(ctx, []string{t.frontier.ID}, models.UrlFrontierStatusCrawled, "")

	return extraction, nil
}

func (t *scrapeTask) OnError(err error) {
	log.Error().Err(err).Str("frontier_id", t.frontier.ID).Msg("Scrape task encountered error")
}

// Timeout returns 0 so the pool uses its default timeout.
func (t *scrapeTask) Timeout() time.Duration { return 0 }

// ScrapeAll scrapes all pending URLs for Singapore Supreme Court
func (s *SingaporeSupremeCourtScraper) ScrapeAll(ctx context.Context, jobID string) error {
	log.Info().Str("jobID", jobID).Msg("Starting ScrapeAll for Singapore Supreme Court")

	// Derive cancellable context so we can propagate cancellations to the pool
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Page pool used by all workers (reused across tasks)
	pagePool := rod.NewPool[rod.Page](s.BaseScraper.Config.MaxConcurrency)
	defer pagePool.Cleanup(func(p *rod.Page) {
		if err := p.Close(); err != nil {
			log.Error().Err(err).Msg("Error closing browser page")
		}
	})

	// Validate repository
	if s.UrlFrontierRepo == nil {
		return fmt.Errorf("URL frontier repository not initialized")
	}

	// Worker pool initialisation
	numWorkers := s.BaseScraper.Config.MaxConcurrency
	if numWorkers <= 0 {
		numWorkers = 10
	}

	poolConfig := work.PoolConfig{
		NumWorkers:      numWorkers,
		TaskChannelSize: numWorkers * 2,
		WorkManager:     s.workManager,
	}
	workerPool, err := work.NewWorkerPoolWithConfig[repository.Extraction](poolConfig)
	if err != nil {
		return err
	}

	workerPool.Start(ctx, jobID)

	// Track global stats safely via atomics
	var totalSuccessCount int64
	var totalFailureCount int64

	// Results collector
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for res := range workerPool.Results() {
			if res.IsSuccess() {
				atomic.AddInt64(&totalSuccessCount, 1)
			} else {
				atomic.AddInt64(&totalFailureCount, 1)
			}
		}
	}()

	batchNumber := 1

	for {
		select {
		case <-ctx.Done():
			log.Info().Msg("Context cancelled, stopping ScrapeAll")
			workerPool.Stop()
			wg.Wait()
			return ctx.Err()
		default:
		}

		// Fetch a batch of pending frontiers (same size as workers for efficiency)
		frontiers, err := s.UrlFrontierRepo.GetPendingByDataSource(ctx, s.BaseScraper.Config.DataSource.ID, numWorkers)
		if err != nil {
			workerPool.Stop()
			wg.Wait()
			return err
		}

		// No more work? break the loop
		if len(frontiers) == 0 {
			log.Info().Int("batch", batchNumber).Msg("No more pending URL frontiers found")
			break
		}

		log.Info().Int("batch", batchNumber).Int("count", len(frontiers)).Msg("Dispatching tasks to worker pool")

		// Enqueue tasks
		for _, frontier := range frontiers {
			task := &scrapeTask{
				scraper:  s,
				pagePool: &pagePool,
				frontier: frontier,
				jobID:    jobID,
			}

			if err := workerPool.AddTask(ctx, task); err != nil {
				log.Error().Err(err).Str("frontier_id", frontier.ID).Msg("Failed to enqueue scrape task")
				// Mark as failed immediately since task could not be queued
				_ = s.UpdateUrlFrontierStatus(ctx, []string{frontier.ID}, models.UrlFrontierStatusFailed, err.Error())
			}
		}

		batchNumber++
	}

	// All tasks submitted shut down the pool and wait for completion
	workerPool.Stop()
	wg.Wait()

	log.Info().
		Int("total_batches", batchNumber-1).
		Int64("total_success", totalSuccessCount).
		Int64("total_failed", totalFailureCount).
		Msg("Completed ScrapeAll for Singapore Supreme Court")

	return nil
}

// ScrapeByURLFrontierID scrapes a specific URL frontier by ID
func (s *SingaporeSupremeCourtScraper) ScrapeByURLFrontierID(ctx context.Context, id string, jobID string) error {
	log.Info().Str("id", id).Str("jobID", jobID).Msg("Starting ScrapeByURLFrontierID for Singapore Supreme Court")

	// 1. Fetch the URL frontier
	frontier, err := s.UrlFrontierRepo.GetByID(ctx, id)
	if err != nil {
		log.Error().Err(err).Str("id", id).Msg("Failed to get URL frontier by ID")
		return err
	}

	// 2. Update status to processing
	err = s.UpdateUrlFrontierStatus(ctx, []string{frontier.ID}, models.UrlFrontierStatusProcessing, "")
	if err != nil {
		log.Warn().Err(err).Str("id", id).Msg("Failed to update URL frontier status to processing")
		// Continue anyway
	}

	// 3. Create a browser page
	page, err := s.createPage()
	if err != nil {
		_ = s.UpdateUrlFrontierStatus(ctx, []string{frontier.ID}, models.UrlFrontierStatusFailed, "failed to create browser page")
		return err
	}
	defer page.Close()

	// 4. Scrape the page
	extraction, err := s.ScrapePage(ctx, page, frontier, jobID)
	if err != nil {
		log.Error().Err(err).Str("id", id).Msg("Failed to scrape page")
		_ = s.UpdateUrlFrontierStatus(ctx, []string{frontier.ID}, models.UrlFrontierStatusFailed, err.Error())
		return err
	}

	// 5. Save the extraction
	_, err = s.SaveExtraction(ctx, extraction)
	if err != nil {
		log.Error().Err(err).Str("id", id).Msg("Failed to save extraction")
		_ = s.UpdateUrlFrontierStatus(ctx, []string{frontier.ID}, models.UrlFrontierStatusFailed, "failed to save extraction")
		return err
	}

	// 6. Mark as crawled on success
	err = s.UpdateUrlFrontierStatus(ctx, []string{frontier.ID}, models.UrlFrontierStatusCrawled, "")
	if err != nil {
		log.Warn().Err(err).Str("id", id).Msg("Failed to update URL frontier status to crawled")
		// Not a fatal error, we have scraped and saved.
	}

	log.Info().Str("id", id).Msg("Successfully completed ScrapeByURLFrontierID")
	return nil
}

// Consume processes a message from a queue
func (s *SingaporeSupremeCourtScraper) Consume(ctx context.Context, message []byte) error {
	log.Info().Msg("Processing Singapore Supreme Court message from queue")

	var req messaging.ScrapeRequest
	if err := json.Unmarshal(message, &req); err != nil {
		log.Error().Err(err).Msg("Failed to unmarshal message into ScrapeRequest")
		return err
	}

	switch req.Type {
	case constants.ScrapeAllAction:
		log.Info().Str("jobID", req.ID).Msg("Received request to scrape all")
		return s.ScrapeAll(ctx, req.ID)
	case constants.ScrapeByIDAction:
		if req.Payload.URLFrontierID == "" {
			err := fmt.Errorf("UrlFrontierID cannot be empty for type '%s'", constants.ScrapeByIDAction)
			log.Error().Err(err).Msg("Invalid scrape request")
			return err
		}
		log.Info().Str("id", req.Payload.URLFrontierID).Str("jobID", req.ID).Msg("Received URL frontier to scrape")

		// You can either call ScrapeByURLFrontierID or a more direct scraping method
		// if you have the full frontier object.
		// ScrapeByURLFrontierID fetches it again, which might be redundant but safer.
		return s.ScrapeByURLFrontierID(ctx, req.Payload.URLFrontierID, req.ID)
	default:
		log.Error().Str("type", string(req.Type)).Msg("Invalid action type for scraper")
		return fmt.Errorf("invalid action type for scraper: %s", req.Type)
	}
}

func (s *SingaporeSupremeCourtScraper) ScrapePage(ctx context.Context, page *rod.Page, urlFrontier repository.UrlFrontier, jobID string) (repository.Extraction, error) {
	select {
	case <-ctx.Done():
		return repository.Extraction{}, ctx.Err()
	default:
	}
	log.Info().Str("jobID", jobID).Str("url", urlFrontier.Url).Msg("Scraping url")

	rpCtx := page.Context(ctx)
	wait := rpCtx.MustWaitNavigation()
	err := rpCtx.Navigate(urlFrontier.Url)
	if err != nil {
		log.Error().Err(err).Msg("Error navigating to url")
		return repository.Extraction{}, err
	}
	wait()

	rp := rpCtx.MustWaitStable()

	judgement, err := rp.Element("#divJudgement")
	if err != nil {
		log.Error().Err(err).Msg("Error getting judgement")
		return repository.Extraction{}, err
	}
	now := time.Now()
	var extraction repository.Extraction
	extraction.ID = urlFrontier.ID
	extraction.UrlFrontierID = urlFrontier.ID
	extraction.Language = "en"
	extraction.CreatedAt = now
	extraction.UpdatedAt = now
	siteContent, err := judgement.HTML()
	if err != nil {
		log.Error().Err(err).Msg("Error getting site content")
		return repository.Extraction{}, err
	}
	extraction.SiteContent = pgtype.Text{String: siteContent, Valid: true}
	nav, err := rp.Element("nav")
	if err != nil {
		log.Error().Err(err).Msg("Error getting nav")
		return repository.Extraction{}, err
	}
	donwloadLink, err := extractPdfUrl(s.BaseScraper.Config, nav)
	if err != nil {
		log.Error().Err(err).Msg("Error extracting pdf url")
		return repository.Extraction{}, err
	}
	var metadata Metadata
	if len(urlFrontier.Metadata) > 0 {
		if err := json.Unmarshal(urlFrontier.Metadata, &metadata); err != nil {
			log.Warn().Err(err).Msg("Failed to unmarshal urlFrontier metadata")
		}
	}
	metadata.PdfUrl = donwloadLink

	updatedMetadata, err := json.Marshal(metadata)
	if err != nil {
		log.Error().Err(err).Msg("Error marshalling metadata")
		return repository.Extraction{}, err
	}
	extraction.Metadata = updatedMetadata

	hashPage := sha256.Sum256([]byte(siteContent))
	hashPageString := hex.EncodeToString(hashPage[:])
	extraction.PageHash = pgtype.Text{String: hashPageString, Valid: true}

	content, err := judgement.Element("content")
	if err != nil {
		log.Info().Msg("Judgement is old")
		err = scrapeOldTemplate(ctx, judgement, &extraction, &urlFrontier)
		if err != nil {
			log.Error().Err(err).Msg("Error scraping old template")
			return repository.Extraction{}, err
		}
	} else {
		err = scrapeNewTemplate(ctx, content, &extraction, &urlFrontier)
		if err != nil {
			log.Error().Err(err).Msg("Error scraping new template")
			return repository.Extraction{}, err
		}
	}

	var finalMetadata Metadata
	if err := json.Unmarshal(extraction.Metadata, &finalMetadata); err != nil {
		log.Error().Err(err).Msg("failed to unmarshal extraction metadata after scrape")
		return repository.Extraction{}, err
	}

	pdfArtifact, err := s.HandlePdf(ctx, extraction.ID, finalMetadata.PdfUrl, finalMetadata.Title)
	if err != nil {
		log.Error().Err(err).Msg("Error handling pdf")
		return repository.Extraction{}, err
	}

	htmlArtifact, err := s.HandleHtml(ctx, extraction.ID, urlFrontier.Url, finalMetadata.Title)
	if err != nil {
		log.Error().Err(err).Msg("Error handling html")
		return repository.Extraction{}, err
	}

	extraction.ArtifactLink = pgtype.Text{String: pdfArtifact.URL, Valid: true}
	extraction.RawPageLink = pgtype.Text{String: htmlArtifact.URL, Valid: true}
	log.Info().Str("jobID", jobID).Str("url", urlFrontier.Url).Msg("Scraped template for url")

	return extraction, nil
}

func (s *SingaporeSupremeCourtScraper) createPage() (*rod.Page, error) {
	incognito, err := s.browser.Incognito()
	if err != nil {
		log.Error().Err(err).Msg("Error creating incognito page")
		return nil, err
	}

	return incognito.MustPage(), nil
}

func (s *SingaporeSupremeCourtScraper) scrapeJob(pagePool *rod.Pool[rod.Page], ctx context.Context, urlFrontier repository.UrlFrontier, jobID string) (repository.Extraction, error) {
	page, err := pagePool.Get(s.createPage)
	if err != nil {
		log.Error().Err(err).Msg("Error getting page")
		return repository.Extraction{}, err
	}
	defer pagePool.Put(page)
	extraction, err := s.ScrapePage(ctx, page, urlFrontier, jobID)
	if err != nil {
		log.Error().Err(err).Msg("Error scraping url frontier")
		return repository.Extraction{}, err
	}
	return extraction, nil
}
