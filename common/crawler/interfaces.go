package crawler

import (
	"context"

	"github.com/LexiconIndonesia/crawler-http-service/repository"
	"github.com/go-rod/rod"
)

// Crawler defines the interface for web crawling operations
type Crawler interface {
	// Setup initializes the crawler, including browser configuration
	Setup(ctx context.Context) error

	// Teardown cleans up resources used by the crawler
	Teardown(ctx context.Context) error

	// CrawlAll crawls all pages from the configured data source
	CrawlAll(ctx context.Context, jobID string) error

	// CrawlByKeyword crawls pages based on a search term
	CrawlByKeyword(ctx context.Context, keyword string, jobID string) error

	// CrawlByURL crawls a specific URL
	CrawlByURL(ctx context.Context, url string, jobID string) error

	// Consume processes a message from a queue
	Consume(ctx context.Context, message []byte) error

	// ExtractElements extracts URL frontiers from a page
	ExtractElements(ctx context.Context, element *rod.Element) (repository.UrlFrontier, error)

	// Crawl a page
	CrawlPage(ctx context.Context, page *rod.Page, url string) ([]repository.UrlFrontier, error)
}

// Scraper defines the interface for web scraping operations
type Scraper interface {
	// Setup initializes the scraper
	Setup(ctx context.Context) error

	// Teardown cleans up resources used by the scraper
	Teardown(ctx context.Context) error

	// ScrapeAll scrapes all pending URLs for a data source
	ScrapeAll(ctx context.Context, jobID string) error

	// ScrapeByURLFrontierID scrapes a specific URL frontier by ID
	ScrapeByURLFrontierID(ctx context.Context, id string, jobID string) error

	// Consume processes a message from a queue
	Consume(ctx context.Context, message []byte) error

	// Scrape a Page
	ScrapePage(ctx context.Context, page *rod.Page, url repository.UrlFrontier, jobID string) (repository.Extraction, error)
}
