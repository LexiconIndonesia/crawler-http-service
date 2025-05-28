package crawler

import (
	"context"

	"github.com/LexiconIndonesia/crawler-http-service/common/models"

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
	CrawlAll(ctx context.Context) error

	// CrawlByKeyword crawls pages based on a search term
	CrawlByKeyword(ctx context.Context, keyword string) error

	// CrawlByURL crawls a specific URL
	CrawlByURL(ctx context.Context, url string) error

	// Consume processes a message from a queue
	Consume(ctx context.Context, message []byte) error

	// ExtractElements extracts URL frontiers from a page
	ExtractElements(ctx context.Context, page *rod.Element) (repository.UrlFrontier, error)

	// Crawl a page
	CrawlPage(ctx context.Context, page *rod.Page, url string) error
}

// Scraper defines the interface for web scraping operations
type Scraper interface {
	// Setup initializes the scraper
	Setup(ctx context.Context) error

	// Teardown cleans up resources used by the scraper
	Teardown(ctx context.Context) error

	// ScrapeAll scrapes all pending URLs for a data source
	ScrapeAll(ctx context.Context) error

	// ScrapeByUrlFrontierID scrapes a specific URL frontier by ID
	ScrapeByUrlFrontierID(ctx context.Context, id string) error

	// Consume processes a message from a queue
	Consume(ctx context.Context, message []byte) error

	// ExtractElements extracts data from a page
	ExtractElements(ctx context.Context, page *rod.Page) ([]repository.Extraction, error)

	// ExtractArtifactsFromPage extracts and downloads artifacts from a page
	ExtractArtifactsFromPage(ctx context.Context, page *rod.Page) ([]models.ExtractionArtifact, error)

	// Navigate navigates to a URL and returns the page
	Navigate(ctx context.Context, url string) (*rod.Page, error)
}
