package isc

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"strconv"
	"time"

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
	"github.com/samber/lo"
)

// IndonesiaSupremeCourtCrawler is a crawler for the Indonesia Supreme Court
type IndonesiaSupremeCourtCrawler struct {
	crawler.BaseCrawler
	Config      crawler.IndonesiaSupremeCourtConfig
	browser     *rod.Browser
	workManager *work.WorkManager
}

// NewMahkamahAgungCrawler creates a newIndonesiaSupremeCourtCrawler
func NewIndonesiaSupremeCourtCrawler(db *db.DB, config crawler.IndonesiaSupremeCourtConfig, baseConfig crawler.BaseCrawlerConfig, broker *messaging.NatsBroker) (*IndonesiaSupremeCourtCrawler, error) {
	// This is just a stub - no actual implementation
	return &IndonesiaSupremeCourtCrawler{
		BaseCrawler: crawler.BaseCrawler{
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

func (c *IndonesiaSupremeCourtCrawler) Setup(ctx context.Context) error {

	log.Info().Msg("Setting up Indonesia Supreme Court crawler")
	// In a real implementation, this would initialize any necessary resources

	browser := rod.New()

	err := browser.Connect()
	if err != nil {
		log.Err(err).Msgf("Error connecting to browser")
		return err
	}
	return nil
}

func (c *IndonesiaSupremeCourtCrawler) Teardown(ctx context.Context) error {
	log.Info().Msg("Tearing down Indonesia Supreme Court crawler")
	err := c.browser.Close()
	if err != nil {
		log.Err(err).Msgf("Error closing browser")
		return err
	}
	return nil
}

// CrawlAll crawls all pages from the Indonesia Supreme Court
func (c *IndonesiaSupremeCourtCrawler) CrawlAll(ctx context.Context, jobID string) error {

	l := log.Info().
		Str("dataSourceID", c.BaseCrawler.Config.DataSource.ID).
		Str("jobID", jobID)

	l.Msgf("Start Crawling all judgments dataSourceName=%s", c.BaseCrawler.Config.DataSource.Name)

	startUrl, err := newStartURLCrawler(c.BaseCrawler.Config, c.Config)
	if err != nil {
		log.Err(err).Msg("Error creating URL crawler")
		return err
	}

	pagePool := rod.NewPagePool(c.BaseCrawler.Config.MaxConcurrency)
	defer pagePool.Cleanup(func(p *rod.Page) {
		err := p.Close()
		if err != nil {
			log.Error().Err(err).Msg("Error closing page")
		}
	})

	rpLast := c.browser.MustPage()
	defer rpLast.Close()
	lastPage, err := c.getLastPage(ctx, rpLast, startUrl.constructURL())
	if err != nil {
		return fmt.Errorf("failed to get last page: %w", err)
	}

	lastPageInt, totalResult := lastPage.Unpack()

	log.Info().Msg("Total result: " + strconv.Itoa(totalResult))

	urlList := generateUrls(startUrl, startUrl.currentPage, lastPageInt)
	// TODO: process urlList
	_ = urlList

	// Create a new context with cancellation
	ctx, cancel := context.WithCancel(ctx)
	defer cancel() // Ensure all resources are cleaned up

	// Create type-safe worker pool for crawling tasks (no meaningful return value)
	poolConfig := work.PoolConfig{
		NumWorkers:      c.BaseCrawler.Config.MaxConcurrency,
		TaskChannelSize: c.BaseCrawler.Config.MaxConcurrency * 2,
		ResultChanSize:  c.BaseCrawler.Config.MaxConcurrency * 2,
		WorkManager:     nil, // Don't use work manager for individual tasks
	}
	workerPool, err := work.NewWorkerPoolWithConfig[[]repository.UrlFrontier](poolConfig)
	if err != nil {
		return err
	}
	// Start the job in work manager
	if err := c.workManager.Start(ctx, jobID); err != nil {
		log.Error().Err(err).Str("dataSourceID", c.BaseCrawler.Config.DataSource.ID).Str("jobID", jobID).Msg("Failed to start work in manager")
		return fmt.Errorf("failed to start work manager for job %s: %w", jobID, err)
	}
	defer func() {
		if err := c.workManager.Complete(ctx, jobID); err != nil {
			log.Error().Err(err).Str("dataSourceID", c.BaseCrawler.Config.DataSource.ID).Str("jobID", jobID).Msg("Failed to complete work in manager")
		}
	}()

	workerPool.Start(ctx, jobID)
	defer workerPool.Stop()

	// Process results in a separate goroutine
	go func() {
		defer func() {
			log.Debug().Msg("Result processing goroutine completed")
		}()

		for result := range workerPool.Results() {
			if result.IsSuccess() {
				l.
					Str("taskID", result.TaskID).
					Dur("duration", result.Duration).
					Msg("Crawling task completed successfully")

				// Get the batch of UrlFrontier objects from the result
				urlFrontiers := result.Result
				if len(urlFrontiers) > 0 {
					// Save the batch to the database
					savedFrontiers, err := c.BaseCrawler.SaveUrlFrontierBatch(ctx, urlFrontiers)
					if err != nil {
						log.Error().
							Err(err).
							Str("dataSourceID", c.BaseCrawler.Config.DataSource.ID).
							Str("jobID", jobID).
							Str("taskID", result.TaskID).
							Int("numFrontiers", len(urlFrontiers)).
							Msg("Failed to save URL frontier batch")
					} else {
						l.
							Str("taskID", result.TaskID).
							Int("numSaved", len(savedFrontiers)).
							Msg("Successfully saved URL frontier batch")
					}
				} else {
					log.Debug().
						Str("jobID", jobID).
						Str("taskID", result.TaskID).
						Msg("Crawling task returned no UrlFrontiers to save")
				}
			} else {
				log.Error().
					Str("dataSourceID", c.BaseCrawler.Config.DataSource.ID).
					Str("jobID", jobID).
					Str("taskID", result.TaskID).
					Err(result.Error).
					Dur("duration", result.Duration).
					Msg("Crawling task failed")
			}
		}
	}()

	for _, url := range urlList {
		// Before processing each URL, verify that the job is still marked as running.
		running, err := c.workManager.IsRunning(ctx, jobID)
		if err != nil {
			log.Error().Err(err).Str("jobID", jobID).Msg("Failed to query WorkManager job state")
		} else if !running {
			log.Info().Str("jobID", jobID).Msg("Job has been cancelled – stopping CrawlAll URL loop")
			cancel()
			break
		}

		// Capture url in closure
		crawlURL := url

		// Create a cleaner task ID based on URL
		taskID := fmt.Sprintf("indonesia-supreme-court-crawler-page-%d", url.currentPage)

		// Create a simple task using the helper function
		task, err := work.NewTask(
			func(ctx context.Context) ([]repository.UrlFrontier, error) {
				// Check context before starting work
				select {
				case <-ctx.Done():
					return nil, ctx.Err()
				default:
					if c.Config.MinDelayMs > 0 && c.Config.MaxDelayMs > 0 {
						delay := rand.Intn(c.Config.MaxDelayMs-c.Config.MinDelayMs) + c.Config.MinDelayMs
						log.Debug().Int("ms", delay).Msg("Delaying request")
						time.Sleep(time.Duration(delay) * time.Millisecond)
					}
					return c.crawlingJob(&pagePool, ctx, crawlURL)
				}
			},
			work.WithID[[]repository.UrlFrontier](taskID),
			work.WithErrorHandler[[]repository.UrlFrontier](func(err error) {
				log.Err(err).
					Str("url", crawlURL.constructURL()).
					Str("jobID", jobID).
					Str("taskID", taskID).
					Msg("Error crawling URL")
			}),
			work.WithTimeout[[]repository.UrlFrontier](180*time.Second), // timeout parameter
		)

		if err != nil {
			log.Error().
				Err(err).
				Str("dataSourceID", c.BaseCrawler.Config.DataSource.ID).
				Str("jobID", jobID).
				Str("taskID", taskID).
				Str("url", crawlURL.constructURL()).
				Msg("Failed to create crawling task")
			continue
		}

		// Add task with non-blocking behavior
		if err := workerPool.AddTaskNonBlocking(task); err != nil {
			log.Warn().
				Err(err).
				Str("dataSourceID", c.BaseCrawler.Config.DataSource.ID).
				Str("jobID", jobID).
				Str("taskID", taskID).
				Str("url", crawlURL.constructURL()).
				Msg("Failed to add crawling task - queue might be full")

			// If non-blocking fails, try with context-aware blocking
			if err := workerPool.AddTask(ctx, task); err != nil {
				log.Error().
					Err(err).
					Str("dataSourceID", c.BaseCrawler.Config.DataSource.ID).
					Str("jobID", jobID).
					Str("url", crawlURL.constructURL()).
					Str("taskID", taskID).
					Msg("Failed to add crawling task - stopping")
				break
			}
		}
	}

	defer func() {

		// Print final statistics
		stats := workerPool.Stats()
		l.
			Int64("tasksCompleted", stats.TasksCompleted).
			Int64("tasksQueued", stats.TasksQueued).
			Int64("activeWorkers", stats.ActiveWorkers).
			Int("totalUrlsProcessed", len(urlList)).
			Msg("Crawling session completed")
	}()

	l.
		Int("totalTasks", len(urlList)).
		Msg("All tasks queued, waiting for completion")

	return nil
}

// CrawlByKeyword crawls pages from the Indonesia Supreme Court based on a search term
func (c *IndonesiaSupremeCourtCrawler) CrawlByKeyword(ctx context.Context, keyword string, jobID string) error {
	l := log.Info().
		Str("dataSourceID", c.BaseCrawler.Config.DataSource.ID).
		Str("keyword", keyword).
		Str("jobID", jobID)

	l.Msgf("Start Crawling by keyword dataSourceID=%s", c.BaseCrawler.Config.DataSource.ID)

	// Note: The work manager start is initiated later, just before the worker pool
	// begins processing tasks. This avoids duplicate start calls that can
	// trigger the "work is already running" error.

	// Validate keyword
	if keyword == "" {
		return fmt.Errorf("keyword cannot be empty")
	}

	// Create URL crawler with keyword search
	l.Msg("Creating URL crawler")
	startUrl, err := newStartURLCrawler(c.BaseCrawler.Config, c.Config)
	if err != nil {
		log.Err(err).Msg("Error creating URL crawler")
		return err
	}

	// Set the search phrase to the keyword
	startUrl.searchPhrase = keyword

	l.Msg("Creating page pool")
	pagePool := rod.NewPagePool(c.BaseCrawler.Config.MaxConcurrency)
	defer pagePool.Cleanup(func(p *rod.Page) {
		err := p.Close()
		if err != nil {
			log.Error().Err(err).Msg("Error closing page")
		}
	})

	l.Msg("Creating new page for getting last page number")
	rpLast := c.browser.MustPage()
	defer rpLast.Close()
	l.Msg("Getting last page number")
	lastPage, err := c.getLastPage(ctx, rpLast, startUrl.constructURL())
	if err != nil {
		return fmt.Errorf("failed to get last page for keyword search: %w", err)
	}

	lastPageInt, totalResult := lastPage.Unpack()

	l.Msg("Total result: " + strconv.Itoa(totalResult))

	urlList := generateUrls(startUrl, startUrl.currentPage, lastPageInt)

	// Create a new context with cancellation
	ctx, cancel := context.WithCancel(ctx)
	defer cancel() // Ensure all resources are cleaned up

	// Create type-safe worker pool for crawling tasks
	poolConfig := work.PoolConfig{
		NumWorkers: c.BaseCrawler.Config.MaxConcurrency,

		WorkManager: nil,
	}
	workerPool, err := work.NewWorkerPoolWithConfig[[]repository.UrlFrontier](poolConfig)
	if err != nil {
		return err
	}

	// Start the job in work manager
	if err := c.workManager.Start(ctx, jobID); err != nil {
		log.Error().Err(err).Str("dataSourceID", c.BaseCrawler.Config.DataSource.ID).Str("jobID", jobID).Msg("Failed to start work in manager")
		return fmt.Errorf("failed to start work manager for job %s: %w", jobID, err)
	}
	defer func() {
		if err := c.workManager.Complete(ctx, jobID); err != nil {
			log.Error().Err(err).Str("dataSourceID", c.BaseCrawler.Config.DataSource.ID).Str("jobID", jobID).Msg("Failed to complete work in manager")
		}
	}()

	workerPool.Start(ctx, jobID)
	defer workerPool.Stop()

	// Process results in a separate goroutine
	go func() {
		defer func() {
			log.Debug().Msg("Keyword search result processing goroutine completed")
		}()

		for result := range workerPool.Results() {
			if result.IsSuccess() {
				l.
					Str("taskID", result.TaskID).
					Dur("duration", result.Duration).
					Msg("Keyword crawling task completed successfully")

				// Get the batch of UrlFrontier objects from the result
				urlFrontiers := result.Result
				if len(urlFrontiers) > 0 {
					// Save the batch to the database
					savedFrontiers, err := c.BaseCrawler.SaveUrlFrontierBatch(ctx, urlFrontiers)
					if err != nil {
						log.Error().
							Err(err).
							Str("dataSourceID", c.BaseCrawler.Config.DataSource.ID).
							Str("jobID", jobID).
							Str("taskID", result.TaskID).
							Str("keyword", keyword).
							Int("numFrontiers", len(urlFrontiers)).
							Msg("Failed to save URL frontier batch for keyword search")
					} else {
						l.
							Str("taskID", result.TaskID).
							Int("numSaved", len(savedFrontiers)).
							Msg("Successfully saved URL frontier batch for keyword search")
					}
				} else {
					log.Debug().
						Str("taskID", result.TaskID).
						Str("keyword", keyword).
						Msg("Keyword crawling task returned no UrlFrontiers to save")
				}
			} else {
				log.Error().
					Str("dataSourceID", c.BaseCrawler.Config.DataSource.ID).
					Str("jobID", jobID).
					Str("taskID", result.TaskID).
					Str("keyword", keyword).
					Err(result.Error).
					Dur("duration", result.Duration).
					Msg("Keyword crawling task failed")
			}
		}
	}()

	for _, url := range urlList {
		// Before processing each URL, verify that the job is still marked as running.
		running, err := c.workManager.IsRunning(ctx, jobID)
		if err != nil {
			log.Error().Err(err).Str("jobID", jobID).Msg("Failed to query WorkManager job state")
		} else if !running {
			log.Info().Str("jobID", jobID).Msg("Job has been cancelled - stopping CrawlByKeyword URL loop")
			cancel()
			break
		}

		// Capture url in closure
		crawlURL := url

		// Create a cleaner task ID based on URL and keyword
		taskID := fmt.Sprintf("indonesia-supreme-court-keyword-crawler-page-%d-keyword-%s", url.currentPage, keyword)

		// Create a simple task using the helper function
		task, err := work.NewTask(
			func(ctx context.Context) ([]repository.UrlFrontier, error) {
				// Check context before starting work
				select {
				case <-ctx.Done():
					return nil, ctx.Err()
				default:
					if c.Config.MinDelayMs > 0 && c.Config.MaxDelayMs > 0 {
						delay := rand.Intn(c.Config.MaxDelayMs-c.Config.MinDelayMs) + c.Config.MinDelayMs
						log.Debug().Int("ms", delay).Msg("Delaying request")
						time.Sleep(time.Duration(delay) * time.Millisecond)
					}
					return c.crawlingJob(&pagePool, ctx, crawlURL)
				}
			},
			work.WithID[[]repository.UrlFrontier](taskID),
			work.WithErrorHandler[[]repository.UrlFrontier](func(err error) {
				log.Err(err).
					Str("url", crawlURL.constructURL()).
					Str("taskID", taskID).
					Str("keyword", keyword).
					Msg("Error crawling URL for keyword search")
			}),
			work.WithTimeout[[]repository.UrlFrontier](180*time.Second), // timeout parameter
		)

		if err != nil {
			log.Error().
				Err(err).
				Str("url", crawlURL.constructURL()).
				Str("taskID", taskID).
				Str("keyword", keyword).
				Msg("Failed to create keyword crawling task")
			continue
		}

		// Add task with non-blocking behavior
		if err := workerPool.AddTaskNonBlocking(task); err != nil {
			log.Warn().
				Err(err).
				Str("url", crawlURL.constructURL()).
				Str("taskID", taskID).
				Str("keyword", keyword).
				Msg("Failed to add keyword crawling task - queue might be full")

			// If non-blocking fails, try with context-aware blocking
			if err := workerPool.AddTask(ctx, task); err != nil {
				log.Error().
					Err(err).
					Str("url", crawlURL.constructURL()).
					Str("taskID", taskID).
					Str("keyword", keyword).
					Msg("Failed to add keyword crawling task - stopping")
				break
			}
		}
	}

	// Give tasks time to complete before checking final stats
	l.
		Int("totalTasks", len(urlList)).
		Msg("All keyword search tasks queued, waiting for completion")

	defer func() {

		// Print final statistics
		stats := workerPool.Stats()
		log.Info().
			Int64("tasksCompleted", stats.TasksCompleted).
			Int64("tasksQueued", stats.TasksQueued).
			Int64("activeWorkers", stats.ActiveWorkers).
			Int("totalUrlsProcessed", len(urlList)).
			Str("keyword", keyword).
			Msg("Keyword crawling session completed")
	}()

	return nil
}

// CrawlByURL crawls a specific URL from the Indonesia Supreme Court
func (c *IndonesiaSupremeCourtCrawler) CrawlByURL(ctx context.Context, url string, jobID string) error {
	l := log.Info().
		Str("url", url).
		Str("dataSourceID", c.BaseCrawler.Config.DataSource.ID).
		Str("jobID", jobID)

	l.Msgf("Start Crawling by URL dataSourceID=%s", c.BaseCrawler.Config.DataSource.ID)

	// Validate URL
	if url == "" {
		return fmt.Errorf("URL cannot be empty")
	}

	pagePool := rod.NewPagePool(1) // Only need one page for single URL
	defer pagePool.Cleanup(func(p *rod.Page) {
		err := p.Close()
		if err != nil {
			log.Error().Err(err).Msg("Error closing page")
		}
	})

	// Create a new context with cancellation
	ctx, cancel := context.WithCancel(ctx)
	defer cancel() // Ensure all resources are cleaned up

	// Create type-safe worker pool for single URL crawling task
	poolConfig := work.PoolConfig{
		NumWorkers:  1,
		WorkManager: nil,
	}
	workerPool, err := work.NewWorkerPoolWithConfig[[]repository.UrlFrontier](poolConfig)
	if err != nil {
		return err
	}

	// Start the job in work manager
	if err := c.workManager.Start(ctx, jobID); err != nil {
		log.Error().Err(err).Str("dataSourceID", c.BaseCrawler.Config.DataSource.ID).Str("jobID", jobID).Msg("Failed to start work in manager")
		return fmt.Errorf("failed to start work manager for job %s: %w", jobID, err)
	}
	defer func() {
		if err := c.workManager.Complete(ctx, jobID); err != nil {
			log.Error().Err(err).Str("dataSourceID", c.BaseCrawler.Config.DataSource.ID).Str("jobID", jobID).Msg("Failed to complete work in manager")
		}
	}()

	workerPool.Start(ctx, jobID)
	defer workerPool.Stop()

	go func() {
		defer func() {
			log.Debug().Msg("Single URL result processing goroutine completed")
		}()

		for result := range workerPool.Results() {
			if result.IsSuccess() {
				l.
					Str("taskID", result.TaskID).
					Dur("duration", result.Duration).
					Msg("Single URL crawling task completed successfully")

				// Get the batch of UrlFrontier objects from the result
				urlFrontiers := result.Result
				if len(urlFrontiers) > 0 {
					// Save the batch to the database
					savedFrontiers, err := c.BaseCrawler.SaveUrlFrontierBatch(ctx, urlFrontiers)
					if err != nil {
						log.Error().
							Err(err).
							Str("taskID", result.TaskID).
							Str("url", url).
							Int("numFrontiers", len(urlFrontiers)).
							Msg("Failed to save URL frontier batch for single URL")
						return
					} else {
						l.
							Str("taskID", result.TaskID).
							Int("numSaved", len(savedFrontiers)).
							Msg("Successfully saved URL frontier batch for single URL")
					}
				} else {
					log.Debug().
						Str("taskID", result.TaskID).
						Str("url", url).
						Msg("Single URL crawling task returned no UrlFrontiers to save")
				}
				// Signal success by not sending anything to errorsChan
			} else {
				log.Error().
					Str("taskID", result.TaskID).
					Str("url", url).
					Err(result.Error).
					Dur("duration", result.Duration).
					Msg("Single URL crawling task failed")
			}
		}
	}()

	// Create a task ID for the single URL
	// Before we proceed, confirm the job has not been cancelled.
	running, err := c.workManager.IsRunning(ctx, jobID)
	if err != nil {
		log.Error().Err(err).Str("jobID", jobID).Msg("Failed to query WorkManager job state")
	} else if !running {
		log.Info().Str("jobID", jobID).Msg("Job has been cancelled – aborting CrawlByURL")
		cancel()
		return ctx.Err()
	}

	taskID := fmt.Sprintf("indonesia-supreme-court-single-url-crawler-%s", url)

	urlCrawler, err := newStartURLCrawler(c.BaseCrawler.Config, c.Config)
	if err != nil {
		return err
	}

	// Create a simple task using the helper function
	task, err := work.NewTask(
		func(ctx context.Context) ([]repository.UrlFrontier, error) {
			// Check context before starting work
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			default:
				if c.Config.MinDelayMs > 0 && c.Config.MaxDelayMs > 0 {
					delay := rand.Intn(c.Config.MaxDelayMs-c.Config.MinDelayMs) + c.Config.MinDelayMs
					log.Debug().Int("ms", delay).Msg("Delaying request")
					time.Sleep(time.Duration(delay) * time.Millisecond)
				}
				return c.crawlingJob(&pagePool, ctx, urlCrawler)
			}
		},
		work.WithID[[]repository.UrlFrontier](taskID),
		work.WithErrorHandler[[]repository.UrlFrontier](func(err error) {
			log.Err(err).
				Str("url", url).
				Str("taskID", taskID).
				Msg("Error crawling single URL")
		}),
		work.WithTimeout[[]repository.UrlFrontier](180*time.Second), // timeout parameter
	)

	if err != nil {
		log.Error().
			Err(err).
			Str("url", url).
			Str("taskID", taskID).
			Msg("Failed to create single URL crawling task")
		return err
	}

	// Add task with blocking behavior since we need to wait for the result
	if err := workerPool.AddTask(ctx, task); err != nil {
		log.Error().
			Err(err).
			Str("url", url).
			Str("taskID", taskID).
			Msg("Failed to add single URL crawling task")
		return err
	}

	// Wait for the result or context cancellation
	defer func() {
		// Print final statistics
		stats := workerPool.Stats()
		log.Info().
			Int64("tasksCompleted", stats.TasksCompleted).
			Int64("tasksQueued", stats.TasksQueued).
			Int64("activeWorkers", stats.ActiveWorkers).
			Str("url", url).
			Msg("Single URL crawling session completed")
	}()

	return nil
}

// ExtractElements extracts URL frontiers from a page
func (c *IndonesiaSupremeCourtCrawler) ExtractElements(ctx context.Context, element *rod.Element) (repository.UrlFrontier, error) {
	// Not implemented as per requirements
	log.Error().Msg("ExtractElements method not implemented")
	return repository.UrlFrontier{}, common.ErrNotImplemented
}

func (c *IndonesiaSupremeCourtCrawler) CrawlPage(ctx context.Context, page *rod.Page, url string) ([]repository.UrlFrontier, error) {
	log.Error().Msg("Navigate method not implemented")
	return []repository.UrlFrontier{}, common.ErrNotImplemented
}

// Consume processes a message from a queue
func (c *IndonesiaSupremeCourtCrawler) Consume(ctx context.Context, message []byte) error {
	l := log.Info().Str("dataSourceID", c.BaseCrawler.Config.DataSource.ID)

	var msg messaging.CrawlRequest
	if err := json.Unmarshal(message, &msg); err != nil {
		log.Error().Str("dataSourceID", c.BaseCrawler.Config.DataSource.ID).Err(err).Msg("Failed to unmarshal crawl request")
		return err
	}

	l.Msgf("Consuming message: %+v", msg)

	switch msg.Type {
	case constants.CrawlAllAction:
		return c.CrawlAll(ctx, msg.ID)
	case constants.CrawlByKeywordAction:
		return c.CrawlByKeyword(ctx, msg.Payload.Keyword, msg.ID)
	case constants.CrawlByURLAction:
		return c.CrawlByURL(ctx, msg.Payload.URL, msg.ID)
	default:
		log.Warn().Str("type", string(msg.Type)).Msg("Unknown crawl request type")
		return fmt.Errorf("unknown crawl request type: %s", msg.Type)
	}
}

func (c *IndonesiaSupremeCourtCrawler) createPage() (*rod.Page, error) {
	incognito, err := c.browser.Incognito()
	if err != nil {
		log.Error().Str("dataSourceID", c.BaseCrawler.Config.DataSource.ID).Err(err).Msg("Error creating incognito page")
		return nil, err
	}

	return incognito.MustPage(), nil
}

func (c *IndonesiaSupremeCourtCrawler) crawlingJob(pagePool *rod.Pool[rod.Page], ctx context.Context, urlPage urlCrawler) ([]repository.UrlFrontier, error) {
	result := []repository.UrlFrontier{}
	page, err := pagePool.Get(c.createPage)
	if err != nil {
		log.Error().Str("dataSourceID", c.BaseCrawler.Config.DataSource.ID).Err(err).Msg("Error getting page")
		return result, err
	}
	defer pagePool.Put(page)
	return c.CrawlPage(ctx, page, urlPage.constructURL())
}

func (c *IndonesiaSupremeCourtCrawler) getLastPage(ctx context.Context, rp *rod.Page, url string) (lo.Tuple2[int, int], error) {
	lastPage := 0

	l := log.Info().Str("url", url).Str("dataSourceID", c.BaseCrawler.Config.DataSource.ID)
	l.Msg("Getting last page")

	if err := rp.Context(ctx).Navigate(url); err != nil {
		if err == context.Canceled {
			return lo.Tuple2[int, int]{}, err
		}
		log.Error().Str("url", url).Str("dataSourceID", c.BaseCrawler.Config.DataSource.ID).Err(err).Msg("Error navigating to url")
		return lo.Tuple2[int, int]{}, err
	}
	l.Msg("Navigation completed")
	totalResult := 0
	totalResultElement, err := rp.Elements("a.page-link")
	if err != nil {
		log.Error().Err(err).Msg("Error getting total result element")
		return lo.Tuple2[int, int]{}, err
	}

	for _, element := range totalResultElement {
		attr := element.MustAttribute("href")
		if attr == nil {
			continue
		}
		isLast := element.MustText() == "Last"
		if isLast {
			lastPageNumber := element.MustAttribute("data-ci-pagination-page")

			lastPage, err = strconv.Atoi(*lastPageNumber)
			if err != nil {
				log.Error().Err(err).Msg("Error converting last page number to integer")
				continue
			}
		}
	}

	totalResult = lastPage * 20

	return lo.Tuple2[int, int]{
		A: lastPage,
		B: totalResult,
	}, nil
}
