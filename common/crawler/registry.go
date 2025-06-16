package crawler

import (
	"context"
	"fmt"
	"maps"
	"sync"

	"github.com/LexiconIndonesia/crawler-http-service/common/db"
	"github.com/LexiconIndonesia/crawler-http-service/common/messaging"
	"github.com/LexiconIndonesia/crawler-http-service/repository"
)

var (
	crawlerRegistry     = make(map[string]CrawlerCreator)
	crawlerRegistryLock sync.RWMutex

	scraperRegistry     = make(map[string]ScraperCreator)
	scraperRegistryLock sync.RWMutex
)

// RegisterCrawler registers a crawler creator function for a specific data source type
func RegisterCrawler(name string, creator CrawlerCreator) {
	crawlerRegistryLock.Lock()
	defer crawlerRegistryLock.Unlock()
	crawlerRegistry[name] = creator
}

func RegisterCrawlers(ctx context.Context, natsClient *messaging.NatsBroker, dbConn *db.DB) error {

	// get all datasources
	dataSources, err := dbConn.Queries.GetActiveDataSources(ctx)

	datasourceMap := make(map[string]repository.DataSource)

	if err != nil {
		return fmt.Errorf("failed to get active data sources: %w", err)
	}

	for _, ds := range dataSources {
		datasourceMap[ds.Name] = ds
	}
	// Register all crawlers to listen to NATS messages
	for name, creator := range GetCrawlerRegistry() {

		dataSource := datasourceMap[name]

		baseCrawlerConfig := DefaultBaseCrawlerConfig()
		baseCrawlerConfig.DataSource = dataSource

		// Create a new crawler
		crawler, err := creator(dbConn, dataSource, baseCrawlerConfig, natsClient)
		if err != nil {
			return fmt.Errorf("failed to create crawler: %w", err)
		}

		crawler.Setup(ctx)
	}

	return nil

}

// GetCrawlerRegistry returns the crawler registry
func GetCrawlerRegistry() map[string]CrawlerCreator {
	crawlerRegistryLock.RLock()
	defer crawlerRegistryLock.RUnlock()

	// Create a copy to avoid race conditions
	registryCopy := make(map[string]CrawlerCreator, len(crawlerRegistry))
	maps.Copy(registryCopy, crawlerRegistry)

	return registryCopy
}

func RegisterScraper(name string, creator ScraperCreator) {
	scraperRegistryLock.Lock()
	defer scraperRegistryLock.Unlock()
	scraperRegistry[name] = creator
}

func RegisterScrapers(ctx context.Context, natsClient *messaging.NatsBroker, dbConn *db.DB) error {

	// get all datasources
	dataSources, err := dbConn.Queries.GetActiveDataSources(ctx)

	datasourceMap := make(map[string]repository.DataSource)

	if err != nil {
		return fmt.Errorf("failed to get active data sources: %w", err)
	}

	for _, ds := range dataSources {
		datasourceMap[ds.Name] = ds
	}
	// Register all crawlers to listen to NATS messages
	for name, creator := range GetScraperRegistry() {

		dataSource := datasourceMap[name]

		baseScraperConfig := DefaultBaseScraperConfig()
		baseScraperConfig.DataSource = dataSource

		// Create a new scraper
		scraper, err := creator(dbConn, dataSource, baseScraperConfig, natsClient)
		if err != nil {
			return fmt.Errorf("failed to create scraper: %w", err)
		}

		scraper.Setup(ctx)
	}

	return nil

}

// GetScraperRegistry returns the scraper registry
func GetScraperRegistry() map[string]ScraperCreator {
	scraperRegistryLock.RLock()
	defer scraperRegistryLock.RUnlock()

	// Create a copy to avoid race conditions
	registryCopy := make(map[string]ScraperCreator, len(scraperRegistry))
	maps.Copy(registryCopy, scraperRegistry)

	return registryCopy
}
