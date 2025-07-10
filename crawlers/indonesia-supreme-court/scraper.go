package isc

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/LexiconIndonesia/crawler-http-service/common"
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
	"github.com/nats-io/nats.go/jetstream"
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

// scrapeTask wraps a single URL frontier scraping job so it can be executed
// inside the generic worker pool. It implements work.Executor[repository.Extraction].
type scrapeTask struct {
	scraper  *IndonesiaSupremeCourtScraper
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
func (s *IndonesiaSupremeCourtScraper) ScrapeAll(ctx context.Context, jobID string) error {
	log.Info().Str("dataSourceID", s.BaseScraper.Config.DataSource.ID).Str("jobID", jobID).Msg("Starting ScrapeAll for Indonesia Supreme Court")

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

	// Start the job in work manager for overall scrape session tracking
	if err := s.workManager.Start(ctx, jobID); err != nil {
		if strings.Contains(err.Error(), "already running") {
			log.Warn().Str("jobID", jobID).Msg("Scrape job already running – ignoring duplicate message")
			return nil
		}
		log.Error().Err(err).Str("dataSourceID", s.BaseScraper.Config.DataSource.ID).Str("jobID", jobID).Msg("Failed to start work in manager")
		return fmt.Errorf("failed to start work manager for job %s: %w", jobID, err)
	}

	// Ensure we mark completion when exiting the function
	defer func() {
		if err := s.workManager.Complete(ctx, jobID); err != nil {
			log.Error().Err(err).Str("dataSourceID", s.BaseScraper.Config.DataSource.ID).Str("jobID", jobID).Msg("Failed to complete work in manager")
		}
	}()

	poolConfig := work.PoolConfig{
		NumWorkers:      numWorkers,
		TaskChannelSize: numWorkers * 2,
		TaskTimeout:     120 * time.Second, // 2-minute timeout per task
		WorkManager:     nil,               // Don't track individual tasks in WorkManager
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
		Msg("Completed ScrapeAll for Indonesia Supreme Court")

	return nil
}

// ScrapeByURLFrontierID scrapes a specific URL frontier by ID
func (s *IndonesiaSupremeCourtScraper) ScrapeByURLFrontierID(ctx context.Context, id string, jobID string) error {
	log.Info().Str("dataSourceID", s.BaseScraper.Config.DataSource.ID).Str("id", id).Str("jobID", jobID).Msg("Starting ScrapeByURLFrontierID for Indonesia Supreme Court")

	// 1. Fetch the URL frontier
	frontier, err := s.UrlFrontierRepo.GetByID(ctx, id)
	if err != nil {
		log.Error().Err(err).Str("id", id).Msg("Failed to get URL frontier by ID")
		return err
	}

	// 2. Update status to processing
	err = s.UpdateUrlFrontierStatus(ctx, []string{frontier.ID}, models.UrlFrontierStatusProcessing, "")
	if err != nil {
		// Log the error but continue, as we can still attempt to scrape
		log.Error().Err(err).Str("id", id).Msg("Failed to update URL frontier status to processing")
	}

	// 3. Perform the scrape
	pagePool := rod.NewPool[rod.Page](s.BaseScraper.Config.MaxConcurrency)
	defer pagePool.Cleanup(func(p *rod.Page) {
		if err := p.Close(); err != nil {
			log.Error().Err(err).Msg("Error closing browser page")
		}
	})
	extraction, err := s.scrapeJob(&pagePool, ctx, frontier, jobID)
	if err != nil {
		log.Error().Err(err).Str("id", id).Msg("Failed to scrape URL frontier")
		_ = s.UpdateUrlFrontierStatus(ctx, []string{frontier.ID}, models.UrlFrontierStatusFailed, err.Error())
		return err
	}

	// 4. Update status to crawled
	err = s.UpdateUrlFrontierStatus(ctx, []string{frontier.ID}, models.UrlFrontierStatusCrawled, "")
	if err != nil {
		log.Error().Err(err).Str("id", id).Msg("Failed to update URL frontier status to crawled")
		// Don't return error, as the scraping was successful
	}

	// 5. Optionally, save or publish the extraction
	log.Info().Interface("extraction", extraction).Msg("Scraping successful")

	return nil
}

// Consume processes a message from a queue
func (s *IndonesiaSupremeCourtScraper) Consume(ctx context.Context, message jetstream.Msg) error {
	log.Info().Msg("Processing Indonesia Supreme Court message from queue")

	var req messaging.ScrapeRequest
	if err := json.Unmarshal(message.Data(), &req); err != nil {
		log.Error().Err(err).Msg("Failed to unmarshal scrape request")
		return err
	}

	// Use WithHeartbeat for long-running scraping operations
	return s.BaseScraper.WithHeartbeat(ctx, message, func(ctx context.Context) error {
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
	}, 30*time.Second) // Send heartbeat every 30 seconds
}

func (s *IndonesiaSupremeCourtScraper) ScrapePage(ctx context.Context, page *rod.Page, url repository.UrlFrontier, jobID string) (repository.Extraction, error) {
	log.Info().Str("url", url.Url).Msg("Scraping Singapore Supreme Court page")

	return repository.Extraction{}, common.ErrNotImplemented
}

func (s *IndonesiaSupremeCourtScraper) scrapeJob(pagePool *rod.Pool[rod.Page], ctx context.Context, urlFrontier repository.UrlFrontier, jobID string) (repository.Extraction, error) {
	log.Info().Str("url", urlFrontier.Url).Msg("Scraping Indonesia Supreme Court page (not yet implemented)")

	// TODO: Implement actual scraping logic, including saving extraction.
	return repository.Extraction{}, common.ErrNotImplemented
}
