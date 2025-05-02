package indonesia_supreme_court

import (
	"context"

	"github.com/adryanev/go-http-service-template/common"
	crawler "github.com/adryanev/go-http-service-template/crawlers"
	"github.com/adryanev/go-http-service-template/repository"
	"github.com/go-rod/rod"
)

// CrawlerAdapter adapts the Indonesia Supreme Court crawler to the common crawler interface
type CrawlerAdapter struct {
	*crawler.BaseCrawler
	internalCrawler *IndonesiaSupremeCourtCrawler
}

// NewCrawlerAdapter creates a new adapter for the Indonesia Supreme Court crawler
func NewCrawlerAdapter(service crawler.CrawlerService, dataSource repository.DataSource) *CrawlerAdapter {
	config := crawler.NewCrawlerConfig(common.IndonesiaSupremeCourt, dataSource)
	baseCrawler := crawler.NewBaseCrawler(config, service)
	internalCrawler := NewIndonesiaSupremeCourtCrawler()

	return &CrawlerAdapter{
		BaseCrawler:     baseCrawler,
		internalCrawler: internalCrawler,
	}
}

// Setup initializes the crawler
func (c *CrawlerAdapter) Setup(ctx context.Context) error {
	c.internalCrawler.Setup()
	return nil
}

// Teardown cleans up resources
func (c *CrawlerAdapter) Teardown(ctx context.Context) error {
	c.internalCrawler.Teardown()
	return nil
}

// CrawlAll crawls all available URLs
func (c *CrawlerAdapter) CrawlAll(ctx context.Context) error {
	return c.internalCrawler.CrawlAll(ctx)
}

// CrawlByKeyword crawls URLs related to the given keyword
func (c *CrawlerAdapter) CrawlByKeyword(ctx context.Context, keyword string) error {
	// The original implementation doesn't have this method, so we'll provide a default implementation
	return c.BaseCrawler.CrawlByKeyword(ctx, keyword)
}

// CrawlByURL crawls a specific URL
func (c *CrawlerAdapter) CrawlByURL(ctx context.Context, url string) error {
	return c.internalCrawler.Crawl(ctx, url)
}

// Consume handles messages from a message broker
func (c *CrawlerAdapter) Consume(ctx context.Context, message []byte) error {
	// Adapt the consume method - this is a simplified version
	// In a real implementation, you would need to convert the message to the expected format
	return nil
}

// ExtractElements extracts elements from a page
func (c *CrawlerAdapter) ExtractElements(ctx context.Context, page *rod.Page) ([]crawler.Item, error) {
	// This is a simplified implementation
	// In a real implementation, you would implement proper extraction logic
	var items []crawler.Item

	// Example extraction
	elements, err := page.Elements("div.putusan-list")
	if err != nil {
		return nil, err
	}

	for _, element := range elements {
		title, err := element.Text()
		if err != nil {
			continue
		}

		item := crawler.Item{
			Title:   title,
			URL:     page.MustInfo().URL,
			Content: "",
			Metadata: map[string]interface{}{
				"source": "Indonesia Supreme Court",
			},
		}

		items = append(items, item)
	}

	return items, nil
}
