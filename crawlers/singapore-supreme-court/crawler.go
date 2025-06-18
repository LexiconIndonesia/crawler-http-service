package ssc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	stdUrl "net/url"
	"regexp"
	"strconv"
	"time"

	"github.com/LexiconIndonesia/crawler-http-service/common/constants"
	"github.com/LexiconIndonesia/crawler-http-service/common/crawler"
	"github.com/LexiconIndonesia/crawler-http-service/common/db"
	"github.com/LexiconIndonesia/crawler-http-service/common/messaging"
	"github.com/LexiconIndonesia/crawler-http-service/common/services"
	"github.com/LexiconIndonesia/crawler-http-service/common/storage"
	"github.com/LexiconIndonesia/crawler-http-service/common/work"
	"github.com/LexiconIndonesia/crawler-http-service/repository"
	"github.com/go-rod/rod"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rs/zerolog/log"
	"github.com/samber/lo"
)

// SingaporeSupremeCourtCrawler is a crawler for the Singapore E-Litigation system
type SingaporeSupremeCourtCrawler struct {
	crawler.BaseCrawler
	Config      crawler.SingaporeSupremeCourtConfig
	browser     *rod.Browser
	workManager *work.WorkManager
}

// NewELitigationCrawler creates a new SingaporeSupremeCourtCrawler
func NewSingaporeSupremeCourtCrawler(db *db.DB, config crawler.SingaporeSupremeCourtConfig, baseConfig crawler.BaseCrawlerConfig, broker *messaging.NatsBroker) (*SingaporeSupremeCourtCrawler, error) {
	// Create the crawler
	return &SingaporeSupremeCourtCrawler{
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

// Setup initializes the crawler
func (c *SingaporeSupremeCourtCrawler) Setup(ctx context.Context) error {
	log.Info().Msgf("Setting Up Singapore Supreme Court crawler")
	browser := rod.New()

	err := browser.Connect()
	if err != nil {
		log.Err(err).Msgf("Error connecting to browser")
		return err
	}

	c.browser = browser
	return nil
}

// Teardown cleans up resources
func (c *SingaporeSupremeCourtCrawler) Teardown(ctx context.Context) error {
	log.Info().Msg("Tearing down Singapore Supreme Court crawler")
	err := c.browser.Close()
	if err != nil {
		log.Err(err).Msgf("Error closing browser")
		return err
	}
	return nil
}

// CrawlAll crawls all available judgments
func (c *SingaporeSupremeCourtCrawler) CrawlAll(ctx context.Context, jobID string) error {
	l := log.Info().
		Str("dataSource", c.BaseCrawler.Config.DataSource.Name).
		Str("jobID", jobID)

	l.Msg("Start Crawling all judgments")

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
	lastPage, err := getLastPage(ctx, rpLast, startUrl.constructURL())
	if err != nil {
		return fmt.Errorf("failed to get last page: %w", err)
	}

	lastPageInt, totalResult := lastPage.Unpack()

	log.Info().Msg("Total result: " + strconv.Itoa(totalResult))

	urlList := generateUrls(startUrl, startUrl.currentPage, lastPageInt)

	// Create a new context with cancellation
	ctx, cancel := context.WithCancel(ctx)
	defer cancel() // Ensure all resources are cleaned up

	// Create type-safe worker pool for crawling tasks (no meaningful return value)
	poolConfig := work.PoolConfig{
		NumWorkers:  c.BaseCrawler.Config.MaxConcurrency,
		WorkManager: c.workManager,
	}
	workerPool, err := work.NewWorkerPoolWithConfig[[]repository.UrlFrontier](poolConfig)
	if err != nil {
		return err
	}

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
					Str("jobID", jobID).
					Str("taskID", result.TaskID).
					Err(result.Error).
					Dur("duration", result.Duration).
					Msg("Crawling task failed")
			}
		}
	}()

	for _, url := range urlList {
		// Capture url in closure
		crawlURL := url

		// Create a cleaner task ID based on URL
		taskID := fmt.Sprintf("singapore-supreme-court-crawler-page-%d", url.currentPage)

		// Create a simple task using the helper function
		task, err := work.NewTask[[]repository.UrlFrontier](
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
					return c.crawlingJob(&pagePool, ctx, crawlURL.constructURL())
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
				Str("jobID", jobID).
				Str("taskID", taskID).
				Str("url", crawlURL.constructURL()).
				Msg("Failed to add crawling task - queue might be full")

			// If non-blocking fails, try with context-aware blocking
			if err := workerPool.AddTask(ctx, task); err != nil {
				log.Error().
					Err(err).
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

// CrawlByKeyword crawls judgments by keyword
func (c *SingaporeSupremeCourtCrawler) CrawlByKeyword(ctx context.Context, keyword, jobID string) error {
	l := log.Info().
		Str("dataSource", c.BaseCrawler.Config.DataSource.Name).
		Str("keyword", keyword).
		Str("jobID", jobID)

	l.Msg("Start Crawling by keyword")

	// Update the work manager to signal that the work has started
	// The jobID is used as the workID
	if err := c.workManager.Start(ctx, jobID); err != nil {
		log.Error().Err(err).Str("jobID", jobID).Msg("Failed to start work in manager")
		return fmt.Errorf("failed to start work manager for job %s: %w", jobID, err)
	}

	// Validate keyword
	if keyword == "" {
		return fmt.Errorf("keyword cannot be empty")
	}

	// Create URL crawler with keyword search
	startUrl, err := newStartURLCrawler(c.BaseCrawler.Config, c.Config)
	if err != nil {
		log.Err(err).Msg("Error creating URL crawler")
		return err
	}

	// Set the search phrase to the keyword
	startUrl.searchPhrase = keyword

	pagePool := rod.NewPagePool(c.BaseCrawler.Config.MaxConcurrency)
	defer pagePool.Cleanup(func(p *rod.Page) {
		err := p.Close()
		if err != nil {
			log.Error().Err(err).Msg("Error closing page")
		}
	})

	rpLast := c.browser.MustPage()
	defer rpLast.Close()
	lastPage, err := getLastPage(ctx, rpLast, startUrl.constructURL())
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
		NumWorkers:  c.BaseCrawler.Config.MaxConcurrency,
		WorkManager: c.workManager,
	}
	workerPool, err := work.NewWorkerPoolWithConfig[[]repository.UrlFrontier](poolConfig)
	if err != nil {
		return err
	}

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
					Str("taskID", result.TaskID).
					Str("keyword", keyword).
					Err(result.Error).
					Dur("duration", result.Duration).
					Msg("Keyword crawling task failed")
			}
		}
	}()

	for _, url := range urlList {
		// Capture url in closure
		crawlURL := url

		// Create a cleaner task ID based on URL and keyword
		taskID := fmt.Sprintf("singapore-supreme-court-keyword-crawler-page-%d-keyword-%s", url.currentPage, keyword)

		// Create a simple task using the helper function
		task, err := work.NewTask[[]repository.UrlFrontier](
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
					return c.crawlingJob(&pagePool, ctx, crawlURL.constructURL())
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

// CrawlByURL crawls a specific URL
func (c *SingaporeSupremeCourtCrawler) CrawlByURL(ctx context.Context, url, jobID string) error {
	l := log.Info().
		Str("url", url).
		Str("dataSource", c.BaseCrawler.Config.DataSource.Name).
		Str("jobID", jobID)

	l.Msg("Start Crawling by URL")

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
		WorkManager: c.workManager,
	}
	workerPool, err := work.NewWorkerPoolWithConfig[[]repository.UrlFrontier](poolConfig)
	if err != nil {
		return err
	}

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
	taskID := fmt.Sprintf("singapore-supreme-court-single-url-crawler-%s", c.BaseCrawler.GenerateID(url))

	// Create a simple task using the helper function
	task, err := work.NewTask[[]repository.UrlFrontier](
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
				return c.crawlingJob(&pagePool, ctx, url)
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

func (c *SingaporeSupremeCourtCrawler) ExtractElements(ctx context.Context, element *rod.Element) (repository.UrlFrontier, error) {
	log.Debug().Msg("Extracting elements from Singapore Supreme Court page")

	// Check context before processing
	select {
	case <-ctx.Done():
		return repository.UrlFrontier{}, ctx.Err()
	default:
	}
	link := element.MustElement("a.h5.gd-heardertext").MustAttribute("href")
	if link == nil {
		log.Error().Msg("link is empty")
		return repository.UrlFrontier{}, errors.New("link is empty")
	}
	if !isDetailPage(*link) {
		log.Error().Msg("link is not a detail page")
		return repository.UrlFrontier{}, errors.New("link is not a detail page")
	}

	title, err := getTitle(element)
	if err != nil {
		log.Error().Err(err).Msg("Error getting title")
		title = ""
	}
	caseNumbers, err := getCaseNumbers(element)
	if err != nil {
		log.Error().Err(err).Msg("Error getting case numbers")
		caseNumbers = []string{}
	}
	citationNumber, err := getCitationNumber(element)
	if err != nil {
		log.Error().Err(err).Msg("Error getting citation number")
		citationNumber = ""
	}
	categories, err := getCategories(element)
	if err != nil {
		log.Error().Err(err).Msg("Error getting categories")
		categories = []string{}
	}

	decisionDate, err := getDecisionDate(element)
	if err != nil {
		log.Error().Err(err).Msg("Error getting decision date")
		decisionDate = ""
	}

	metadata := UrlFrontierMetadata{
		Title:          title,
		CaseNumbers:    caseNumbers,
		CitationNumber: citationNumber,
		DecisionDate:   decisionDate,
		Categories:     categories,
	}

	metadataBytes, err := metadata.ToJson()
	if err != nil {
		log.Error().Err(err).Msg("Error converting metadata to JSON")
		metadataBytes = []byte{}
	}

	detailUrls := repository.UrlFrontier{
		DataSourceID: c.BaseCrawler.Config.DataSource.ID,
		Domain:       c.BaseCrawler.Config.DataSource.BaseUrl.String,
		Keyword:      pgtype.Text{String: "", Valid: true},
		Url:          fmt.Sprintf("%s%s", c.BaseCrawler.Config.DataSource.BaseUrl.String, *link),
		Metadata:     metadataBytes,
	}

	return detailUrls, nil
}

func (c *SingaporeSupremeCourtCrawler) CrawlPage(ctx context.Context, page *rod.Page, url string) ([]repository.UrlFrontier, error) {
	log.Info().Str("url", url).Msg("Navigating to E-Litigation URL")
	log.Info().Msg("Crawling URL: " + url)

	res := []repository.UrlFrontier{}
	// Check context before starting
	select {
	case <-ctx.Done():
		return res, ctx.Err()
	default:
	}

	rpCtx := page.Context(ctx)
	wait := rpCtx.MustWaitNavigation()
	err := rpCtx.Navigate(url)
	if err != nil {
		log.Error().Err(err).Msg("Error navigating to url")
		return res, err
	}
	wait()

	// Check context after navigation
	select {
	case <-ctx.Done():
		return res, ctx.Err()
	default:
	}

	page = rpCtx.MustWaitStable()

	elements, err := page.Elements("#listview > div.row > div.card.col-12")
	if err != nil {
		log.Error().Err(err).Msg("Error getting elements")
		return res, err
	}

	log.Info().Msgf("Found %d elements", len(elements))

	// Process each element found
	processedCount := 0
	for _, element := range elements {
		// Check context during element processing
		select {
		case <-ctx.Done():
			return res, ctx.Err()
		default:
		}

		crawlerResult, err := c.ExtractElements(ctx, element)
		if err != nil {
			log.Error().Err(err).Msg("Error extracting elements")
			continue
		}

		// Only process if we actually got a valid result
		if crawlerResult.Url != "" {
			processedCount++
			log.Info().Msg("Crawler result: " + crawlerResult.Url)

			// Here you would typically save the URL frontier or process it further
			// For example: c.SaveUrlFrontier(ctx, crawlerResult)
		}
	}

	log.Info().
		Str("url", url).
		Int("elementsFound", len(elements)).
		Int("elementsProcessed", processedCount).
		Msg("Crawling page completed")

	return res, nil
}

// Consume processes a message from a queue
func (c *SingaporeSupremeCourtCrawler) Consume(ctx context.Context, message []byte) error {
	var msg messaging.CrawlRequest
	if err := json.Unmarshal(message, &msg); err != nil {
		log.Error().Err(err).Msg("Failed to unmarshal crawl request")
		return err
	}

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

func (c *SingaporeSupremeCourtCrawler) createPage() (*rod.Page, error) {
	incognito, err := c.browser.Incognito()
	if err != nil {
		log.Error().Err(err).Msg("Error creating incognito page")
		return nil, err
	}

	return incognito.MustPage(), nil
}

func (c *SingaporeSupremeCourtCrawler) crawlingJob(pagePool *rod.Pool[rod.Page], ctx context.Context, urlPage string) ([]repository.UrlFrontier, error) {
	result := []repository.UrlFrontier{}
	page, err := pagePool.Get(c.createPage)
	if err != nil {
		log.Error().Err(err).Msg("Error getting page")
		return result, err
	}
	defer pagePool.Put(page)
	return c.CrawlPage(ctx, page, urlPage)
}

func getLastPage(ctx context.Context, rp *rod.Page, url string) (lo.Tuple2[int, int], error) {
	lastPage := 0

	if err := rp.Context(ctx).Navigate(url); err != nil {
		if err == context.Canceled {
			return lo.Tuple2[int, int]{}, err
		}
		log.Error().Err(err).Msg("Error navigating to url")
		return lo.Tuple2[int, int]{}, err
	}
	totalResult := 0
	totalResultElement, err := rp.Element("#listview > div.row.justify-content-between.align-items-center > div.gd-csummary")
	if err != nil {
		log.Error().Err(err).Msg("Error getting total result element")
		return lo.Tuple2[int, int]{}, err
	}
	res, err := totalResultElement.HTML()
	if err != nil {
		log.Error().Err(err).Msg("Error getting total result element")
		return lo.Tuple2[int, int]{}, err
	}
	totalResult, err = strconv.Atoi(regexp.MustCompile(`\d+`).FindString(res))
	if err != nil {
		log.Error().Err(err).Msg("Error converting total result to integer")
		return lo.Tuple2[int, int]{}, err
	}

	elements, err := rp.Elements("#listview > div.row.justify-content-end > div > ul > li.page-item.page-link> a")
	if err != nil {
		log.Error().Err(err).Msg("Error getting elements")
		return lo.Tuple2[int, int]{}, err
	}
	for _, element := range elements {
		if element.MustHTML() == "" {
			continue
		}
		href := element.MustAttribute("href")
		if href == nil {
			continue
		}
		u, err := stdUrl.Parse(*href)
		if err != nil {
			log.Error().Err(err).Msg("Error parsing URL")
			continue
		}
		lp := u.Query().Get("CurrentPage")
		lpInt, err := strconv.Atoi(lp)
		if err != nil {
			log.Error().Err(err).Msg("Error converting LP to integer")
			continue
		}
		if lastPage < lpInt {
			lastPage = lpInt
		}
	}

	return lo.Tuple2[int, int]{
		A: lastPage,
		B: totalResult,
	}, nil
}

func isDetailPage(link string) bool {
	checker, err := regexp.Compile("/gd/s")
	if err != nil {
		log.Error().Err(err).Msg("Regex Compile Error")
		return false
	}

	return checker.MatchString(link)
}
