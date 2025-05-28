package singapore_supreme_court

import (
	"context"
	"fmt"
	stdUrl "net/url"
	"regexp"
	"strconv"
	"sync"

	"github.com/LexiconIndonesia/crawler-http-service/common/crawler"
	"github.com/LexiconIndonesia/crawler-http-service/common/messaging"
	"github.com/LexiconIndonesia/crawler-http-service/repository"
	"github.com/go-rod/rod"
	"github.com/rs/zerolog/log"
	"github.com/samber/lo"
)

// SingaporeSupremeCourtCrawler is a crawler for the Singapore E-Litigation system
type SingaporeSupremeCourtCrawler struct {
	crawler.BaseCrawler
	Config  crawler.SingaporeSupremeCourtConfig
	browser *rod.Browser
}

// NewELitigationCrawler creates a new SingaporeSupremeCourtCrawler
func NewSingaporeSupremeCourtCrawler(config crawler.SingaporeSupremeCourtConfig, baseConfig crawler.BaseCrawlerConfig, broker messaging.MessageBroker) (*SingaporeSupremeCourtCrawler, error) {

	// Create the crawler
	return &SingaporeSupremeCourtCrawler{
		BaseCrawler: crawler.BaseCrawler{
			Config:        baseConfig,
			MessageBroker: broker,
		},
		Config: config,
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
func (c *SingaporeSupremeCourtCrawler) CrawlAll(ctx context.Context) error {
	log.Info().Msg("Crawling all judgments")
	startUrl, err := newStartUrlCrawler(c.BaseCrawler.Config, c.Config)

	if err != nil {
		log.Err(err).Msg("Error creating URL crawler")
		return err
	}

	pagePool := rod.NewPagePool(7)
	defer pagePool.Cleanup(func(p *rod.Page) {
		err := p.Close()
		if err != nil {
			log.Error().Err(err).Msg("Error closing page")
		}
	})

	rpLast := c.browser.MustPage()
	defer rpLast.Close()
	lastPage, err := getLastPage(ctx, rpLast, startUrl.constructUrl())
	if err != nil {
		return fmt.Errorf("failed to get last page: %w", err)
	}

	lastPageInt, totalResult := lastPage.Unpack()

	log.Info().Msg("Total result: " + strconv.Itoa(totalResult))

	urlList := generateUrls(startUrl, startUrl.currentPage, lastPageInt)

	chunks := lo.Chunk(urlList, 7)

	var crawlErrors []error
	errChan := make(chan error, len(urlList)) // Channel to collect errors

	// Create a new context with cancellation
	ctx, cancel := context.WithCancel(ctx)
	defer cancel() // Ensure all resources are cleaned up

	for _, urls := range chunks {
		// Check if context is cancelled before starting new chunk
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		wg := sync.WaitGroup{}

		// Process all URLs in the chunk concurrently
		for _, url := range urls {
			wg.Add(1)
			go func(url string) {
				defer wg.Done()

				// Create a new context for this goroutine
				select {
				case <-ctx.Done():
					errChan <- ctx.Err()
					return
				default:
					if err := c.crawlingJob(&pagePool, ctx, url); err != nil {
						errChan <- fmt.Errorf("error crawling %s: %w", url, err)
						// Optional: cancel other goroutines if you want to stop on first error
						// cancel()
					}
				}
			}(url)
		}

		go func() {
			wg.Wait()
			close(errChan)
		}()
	}

	// Collect any errors that occurred
	for err := range errChan {
		if err != nil {
			if err == context.Canceled {
				return fmt.Errorf("crawling was cancelled: %w", err)
			}
			crawlErrors = append(crawlErrors, err)
		}
	}

	// If there were any errors, return them combined
	if len(crawlErrors) > 0 {
		return fmt.Errorf("encountered %d errors during crawling: %v", len(crawlErrors), crawlErrors)
	}

	return nil
}

// CrawlByKeyword crawls judgments by keyword
func (c *SingaporeSupremeCourtCrawler) CrawlByKeyword(ctx context.Context, keyword string) error {
	log.Info().Str("keyword", keyword).Msg("Crawling judgments by keyword")

	return crawler.ErrNotImplemented
}

// CrawlByURL crawls a specific URL
func (c *SingaporeSupremeCourtCrawler) CrawlByURL(ctx context.Context, url string) error {
	return nil
}

func (c *SingaporeSupremeCourtCrawler) ExtractElements(ctx context.Context, page *rod.Element) (repository.UrlFrontier, error) {
	log.Info().Msg("Extracting elements from E-Litigation page")

	// In a real implementation, this would extract URLs or IDs from API responses

	return repository.UrlFrontier{}, crawler.ErrNotImplemented
}

func (c *SingaporeSupremeCourtCrawler) CrawlPage(ctx context.Context, page *rod.Page, url string) error {
	log.Info().Str("url", url).Msg("Navigating to E-Litigation URL")
	log.Info().Msg("Crawling URL: " + url)

	// Check context before starting
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	rpCtx := page.Context(ctx)
	wait := rpCtx.MustWaitNavigation()
	err := rpCtx.Navigate(url)
	if err != nil {
		log.Error().Err(err).Msg("Error navigating to url")
		return err
	}
	wait()

	// Check context after navigation
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	page = rpCtx.MustWaitStable()

	elements, err := page.Elements("#listview > div.row > div.card.col-12")
	if err != nil {
		log.Error().Err(err).Msg("Error getting elements")
		return err
	}

	log.Info().Msgf("Found %d elements", len(elements))

	detailUrls := []repository.UrlFrontier{}
	for _, element := range elements {

		crawlerResult, err := c.ExtractElements(ctx, element)

		if err != nil {
			log.Error().Err(err).Msg("Error getting element content")
			continue
		}

		detailUrls = append(detailUrls, crawlerResult)
		log.Info().Msg("Crawler result: " + crawlerResult.Url)
	}

	log.Info().Msgf("Crawling url: %s done!", url)
	return nil

	return nil
}

// Consume processes a message from a queue
func (c *SingaporeSupremeCourtCrawler) Consume(ctx context.Context, message []byte) error {
	log.Info().Msg("Processing E-Litigation message from queue")

	// In a real implementation, this would:
	// 1. Unmarshal the message (likely a URL frontier or instruction)
	// 2. Process it according to the crawler's logic
	// 3. Create URL frontiers or perform other actions

	return crawler.ErrNotImplemented
}

func (c *SingaporeSupremeCourtCrawler) createPage() (*rod.Page, error) {
	incognito, err := c.browser.Incognito()
	if err != nil {
		log.Error().Err(err).Msg("Error creating incognito page")
		return nil, err
	}

	return incognito.MustPage(), nil

}

func (c *SingaporeSupremeCourtCrawler) crawlingJob(pagePool *rod.Pool[rod.Page], ctx context.Context, urlPage string) error {
	page, err := pagePool.Get(c.createPage)
	if err != nil {
		log.Error().Err(err).Msg("Error getting page")
		return err
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
