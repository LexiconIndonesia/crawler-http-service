package crawler

import (
	"github.com/LexiconIndonesia/crawler-http-service/common/messaging"
	"github.com/LexiconIndonesia/crawler-http-service/repository"
)

// CrawlerFactory creates crawlers based on data source type
type CrawlerFactory struct {
	MessageBroker messaging.MessageBroker
}

// NewCrawlerFactory creates a new crawler factory
func NewCrawlerFactory(broker messaging.MessageBroker) *CrawlerFactory {
	return &CrawlerFactory{
		MessageBroker: broker,
	}
}

// CrawlerCreator is a function type for creating crawlers
type CrawlerCreator func(dataSource repository.DataSource, baseConfig BaseCrawlerConfig, broker messaging.MessageBroker) (Crawler, error)
type ScraperCreator func(dataSource repository.DataSource, baseConfig BaseScraperConfig, broker messaging.MessageBroker) (Scraper, error)

// ScraperFactory creates scrapers based on data source type
type ScraperFactory struct {
	MessageBroker messaging.MessageBroker
}

// NewScraperFactory creates a new scraper factory
func NewScraperFactory(broker messaging.MessageBroker) *ScraperFactory {
	return &ScraperFactory{
		MessageBroker: broker,
	}
}
