package crawler

import (
	"github.com/LexiconIndonesia/crawler-http-service/common/db"
	"github.com/LexiconIndonesia/crawler-http-service/common/messaging"
	"github.com/LexiconIndonesia/crawler-http-service/repository"
)

// CrawlerFactory creates crawlers based on data source type
type CrawlerFactory struct {
	MessageBroker *messaging.NatsBroker
}

// NewCrawlerFactory creates a new crawler factory
func NewCrawlerFactory(broker *messaging.NatsBroker) *CrawlerFactory {
	return &CrawlerFactory{
		MessageBroker: broker,
	}
}

// CrawlerCreator is a function type for creating crawlers
type CrawlerCreator func(db *db.DB, dataSource repository.DataSource, baseConfig BaseCrawlerConfig, broker *messaging.NatsBroker) (Crawler, error)
type ScraperCreator func(db *db.DB, dataSource repository.DataSource, baseConfig BaseScraperConfig, broker *messaging.NatsBroker) (Scraper, error)

// ScraperFactory creates scrapers based on data source type
type ScraperFactory struct {
	MessageBroker *messaging.NatsBroker
}

// NewScraperFactory creates a new scraper factory
func NewScraperFactory(broker *messaging.NatsBroker) *ScraperFactory {
	return &ScraperFactory{
		MessageBroker: broker,
	}
}
