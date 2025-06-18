package crawler

import (
	"context"
	"fmt"
	"maps"
	"sync"

	"github.com/LexiconIndonesia/crawler-http-service/common/constants"
	"github.com/LexiconIndonesia/crawler-http-service/common/db"
	"github.com/LexiconIndonesia/crawler-http-service/common/messaging"
	"github.com/LexiconIndonesia/crawler-http-service/repository"
	"github.com/nats-io/nats.go"
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
	if err != nil {
		return fmt.Errorf("failed to get active data sources: %w", err)
	}

	datasourceMap := make(map[string]repository.DataSource)
	for _, ds := range dataSources {
		datasourceMap[ds.Name] = ds
	}

	// Register all scrapers to listen to NATS messages
	for name, creator := range GetScraperRegistry() {

		dataSource, ok := datasourceMap[name]
		if !ok {
			continue // or log a warning
		}

		baseScraperConfig := DefaultBaseScraperConfig()
		baseScraperConfig.DataSource = dataSource

		// Create a new scraper
		scraper, err := creator(dbConn, dataSource, baseScraperConfig, natsClient)
		if err != nil {
			return fmt.Errorf("failed to create scraper: %w", err)
		}

		if err := scraper.Setup(ctx); err != nil {
			return fmt.Errorf("failed to setup scraper %s: %w", name, err)
		}

		// Subscribe to scrape topics
		scrapeAllTopic := fmt.Sprintf("%s.%s", dataSource.ID, constants.ScrapeAllAction)
		_, err = messaging.SubscribeToQueueGroup(natsClient, scrapeAllTopic, "scraper-workers", func(msg *nats.Msg) error {
			return scraper.Consume(ctx, msg.Data)
		})
		if err != nil {
			return fmt.Errorf("failed to subscribe to topic %s: %w", scrapeAllTopic, err)
		}

		scrapeByIDTopic := fmt.Sprintf("%s.%s", dataSource.ID, constants.ScrapeByIDAction)
		_, err = messaging.SubscribeToQueueGroup(natsClient, scrapeByIDTopic, "scraper-workers", func(msg *nats.Msg) error {
			return scraper.Consume(ctx, msg.Data)
		})
		if err != nil {
			return fmt.Errorf("failed to subscribe to topic %s: %w", scrapeByIDTopic, err)
		}
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
