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

// CrawlerCreator is a function type for creating crawlers
type CrawlerCreator func(config DataSourceConfig, baseConfig BaseCrawlerConfig, broker MessageBroker) (Crawler, error)

// CreateCrawler creates a crawler based on data source type and config
func (f *CrawlerFactory) CreateCrawler(ctx context.Context, dataSourceType string, configType string, configData json.RawMessage) (Crawler, error) {
	// Load the configuration based on the config type
	sourceConfig, err := LoadDataSourceConfig(configData, configType)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// Validate the configuration
	if err := sourceConfig.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	// Create base crawler config
	baseConfig := DefaultBaseCrawlerConfig()
	baseConfig.DataSourceID = dataSourceType

	// Dynamically get the crawler creator function
	// This approach avoids import cycles by using dependency injection
	crawlerRegistry := GetCrawlerRegistry()
	creator, exists := crawlerRegistry[dataSourceType]

	if !exists {
		return nil, fmt.Errorf("unsupported data source type: %s", dataSourceType)
	}

	return creator(sourceConfig, baseConfig, f.MessageBroker)
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
