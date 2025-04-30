package crawler

import (
	"context"

	"github.com/go-rod/rod"
)

// Crawler is the interface that defines the common behavior for all crawlers
type Crawler interface {

	// Setup performs necessary setup before crawling
	Setup(ctx context.Context) error

	// Teardown performs cleanup after crawling
	Teardown(ctx context.Context) error

	// CrawlAll crawls all available URLs
	CrawlAll(ctx context.Context) error

	// CrawlByKeyword crawls URLs related to the given keyword
	CrawlByKeyword(ctx context.Context, keyword string) error

	// CrawlByURL crawls a specific URL
	CrawlByURL(ctx context.Context, url string) error

	// Consume handles messages from a message broker
	Consume(ctx context.Context, message []byte) error

	// ExtractElements is implemented by specific crawlers to extract data from a page
	ExtractElements(ctx context.Context, page *rod.Page) ([]Item, error)

	// Navigate goes to a specific URL
	Navigate(ctx context.Context, url string) (*rod.Page, error)
}
