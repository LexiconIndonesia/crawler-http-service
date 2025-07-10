package crawler

import (
	"context"

	"github.com/LexiconIndonesia/crawler-http-service/repository"
	"github.com/go-rod/rod"
	"github.com/nats-io/nats.go/jetstream"
)

// HEARTBEATING USAGE:
//
// The Crawler and Scraper interfaces now accept jetstream.Msg for heartbeating support.
// For long-running tasks, implementations should use the WithHeartbeat utility from
// BaseCrawler/BaseScraper:
//
//    return c.BaseCrawler.WithHeartbeat(ctx, message, func(ctx context.Context) error {
//        // Your long-running operation here
//        return c.CrawlAll(ctx, jobID)
//    }, 30*time.Second)
//
// The registry now calls msg.Ack() AFTER processing is complete, not before.
//
// Benefits:
// - Prevents message timeout during long-running operations
// - JetStream knows the message is still being processed
// - Automatic retry if consumer dies without completing the task

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

	// Consume processes a message from a queue with heartbeating support
	Consume(ctx context.Context, message jetstream.Msg) error

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

	// Consume processes a message from a queue with heartbeating support
	Consume(ctx context.Context, message jetstream.Msg) error

	// Scrape a Page
	ScrapePage(ctx context.Context, page *rod.Page, url repository.UrlFrontier, jobID string) (repository.Extraction, error)
}
