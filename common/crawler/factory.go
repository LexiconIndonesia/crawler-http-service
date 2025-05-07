package crawler

import (
	"context"
	"encoding/json"
	"fmt"
)

// CrawlerFactory creates crawlers based on data source type
type CrawlerFactory struct {
	MessageBroker MessageBroker
}

// NewCrawlerFactory creates a new crawler factory
func NewCrawlerFactory(broker MessageBroker) *CrawlerFactory {
	return &CrawlerFactory{
		MessageBroker: broker,
	}
}

// CreateCrawler creates a crawler based on data source type and config
func (f *CrawlerFactory) CreateCrawler(ctx context.Context, dataSourceType string, configType string, configData json.RawMessage) (Crawler, error) {
	// Not implemented as per requirements
	return nil, fmt.Errorf("CreateCrawler method not implemented for type: %s", dataSourceType)
}

// ScraperFactory creates scrapers based on data source type
type ScraperFactory struct {
	MessageBroker MessageBroker
}

// NewScraperFactory creates a new scraper factory
func NewScraperFactory(broker MessageBroker) *ScraperFactory {
	return &ScraperFactory{
		MessageBroker: broker,
	}
}

// CreateScraper creates a scraper based on data source type and config
func (f *ScraperFactory) CreateScraper(ctx context.Context, dataSourceType string, configType string, configData json.RawMessage) (Scraper, error) {
	// Not implemented as per requirements
	return nil, fmt.Errorf("CreateScraper method not implemented for type: %s", dataSourceType)
}
