package crawler

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"sync"
	"time"

	"github.com/LexiconIndonesia/crawler-http-service/common/config"
	"github.com/LexiconIndonesia/crawler-http-service/common/db"
	"github.com/LexiconIndonesia/crawler-http-service/common/messaging"
	"github.com/LexiconIndonesia/crawler-http-service/repository"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/rs/zerolog/log"
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

func RegisterCrawlers(ctx context.Context, natsClient *messaging.NatsBroker, dbConn *db.DB, gcsConfig config.GCSConfig) error {

	// get all datasources
	dataSources, err := dbConn.Queries.GetActiveDataSources(ctx)

	datasourceMap := make(map[string]repository.DataSource)

	if err != nil {
		return fmt.Errorf("failed to get active data sources: %w", err)
	}

	for _, ds := range dataSources {
		datasourceMap[ds.Name] = ds
	}

	dsNames := make([]string, 0, len(datasourceMap))
	for k := range datasourceMap {
		dsNames = append(dsNames, k)
	}
	log.Info().Strs("datasources_from_db", dsNames).Msg("Active data sources")

	regNames := make([]string, 0, len(GetCrawlerRegistry()))
	for k := range GetCrawlerRegistry() {
		regNames = append(regNames, k)
	}
	log.Info().Strs("registry_crawlers", regNames).Msg("Crawler registry contents")

	// Register all crawlers to listen to NATS messages
	for name, creator := range GetCrawlerRegistry() {

		log.Info().Str("crawler_name", name).Msg("Processing crawler from registry")

		dataSource, exists := datasourceMap[name]
		if !exists {
			log.Warn().Str("crawler_name", name).Msg("Skipping crawler registration - data source not found in database")
			continue
		}

		baseCrawlerConfig := DefaultBaseCrawlerConfig()
		baseCrawlerConfig.DataSource = dataSource

		// Create a new crawler
		crawler, err := creator(dbConn, dataSource, baseCrawlerConfig, natsClient)
		if err != nil {
			return fmt.Errorf("failed to create crawler: %w", err)
		}

		if err := crawler.Setup(ctx); err != nil {
			return fmt.Errorf("failed to setup crawler %s: %w", name, err)
		}

		consumer, err := messaging.GetJetStreamConsumer(natsClient, "crawler-workers", fmt.Sprintf("%s.frontier", dataSource.Name))
		if err != nil {
			log.Error().Err(err).Str("crawler_name", name).Msg("Failed to get JetStream consumer")
			continue
		}
		if consumer == nil {
			log.Warn().Str("crawler_name", name).Msg("Skipping crawler registration - consumer is nil")
			continue
		}

		go func(c Crawler) {
			for {
				select {
				case <-ctx.Done():
					log.Info().Str("crawler_name", name).Msg("Context cancelled, stopping consumer.")
					return
				default:
				}

				batch, err := consumer.Fetch(5, jetstream.FetchMaxWait(5*time.Second))
				if err != nil {
					if errors.Is(err, nats.ErrTimeout) || errors.Is(err, context.DeadlineExceeded) {
						continue
					}
					log.Error().Err(err).Str("crawler_name", name).Msg("Failed to fetch messages")
					time.Sleep(1 * time.Second)
					continue
				}

				for msg := range batch.Messages() {
					log.Info().Str("crawler_name", name).Msg("Consuming message")

					// Process the message first, then acknowledge
					err = c.Consume(ctx, msg)
					if err != nil {
						log.Error().Err(err).Str("crawler_name", name).Msg("Failed to consume message")
						// On error, we still acknowledge to avoid reprocessing bad messages
					}

					// Acknowledge the message after processing
					err := msg.Ack()
					if err != nil {
						log.Error().Err(err).Str("crawler_name", name).Msg("Failed to acknowledge message")
					} else {
						log.Info().Str("crawler_name", name).Msg("Acknowledged message")
					}
				}
				if batch.Error() != nil {
					log.Error().Err(batch.Error()).Str("crawler_name", name).Msg("Error during message batch processing")
				}
			}
		}(crawler)
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

func RegisterScrapers(ctx context.Context, natsClient *messaging.NatsBroker, dbConn *db.DB, gcsConfig config.GCSConfig) error {

	// get all datasources
	dataSources, err := dbConn.Queries.GetActiveDataSources(ctx)
	if err != nil {
		return fmt.Errorf("failed to get active data sources: %w", err)
	}

	datasourceMap := make(map[string]repository.DataSource)
	for _, ds := range dataSources {
		datasourceMap[ds.Name] = ds
	}

	dsNames := make([]string, 0, len(datasourceMap))
	for k := range datasourceMap {
		dsNames = append(dsNames, k)
	}
	log.Info().Strs("datasources_from_db", dsNames).Msg("Active data sources for scrapers")

	regNames := make([]string, 0, len(GetScraperRegistry()))
	for k := range GetScraperRegistry() {
		regNames = append(regNames, k)
	}
	log.Info().Strs("registry_scrapers", regNames).Msg("Scraper registry contents")

	// Register all scrapers to listen to NATS messages
	for name, creator := range GetScraperRegistry() {

		log.Info().Str("scraper_name", name).Msg("Processing scraper from registry")

		dataSource, ok := datasourceMap[name]
		if !ok {
			log.Warn().Str("scraper_name", name).Msg("Skipping scraper registration - data source not found in database")
			continue
		}

		if gcsConfig.Bucket == "" {
			log.Error().Msg("Storage bucket is not set, skipping scraper registration")
			return fmt.Errorf("storage bucket is not set, skipping scraper registration")
		}

		baseScraperConfig := DefaultBaseScraperConfigWithBucket(gcsConfig.Bucket)
		baseScraperConfig.DataSource = dataSource

		// Create a new scraper
		scraper, err := creator(dbConn, dataSource, baseScraperConfig, natsClient)
		if err != nil {
			return fmt.Errorf("failed to create scraper: %w", err)
		}

		if err := scraper.Setup(ctx); err != nil {
			return fmt.Errorf("failed to setup scraper %s: %w", name, err)
		}

		consumer, err := messaging.GetJetStreamConsumer(natsClient, "scraper-workers", fmt.Sprintf("%s.extraction", dataSource.Name))
		if err != nil {
			log.Error().Err(err).Str("scraper_name", name).Msg("Failed to get JetStream consumer")
			continue
		}
		if consumer == nil {
			log.Warn().Str("scraper_name", name).Msg("Skipping scraper registration - consumer is nil")
			continue
		}

		go func(s Scraper) {
			for {
				select {
				case <-ctx.Done():
					log.Info().Str("scraper_name", name).Msg("Context cancelled, stopping consumer.")
					return
				default:
				}

				batch, err := consumer.Fetch(5, jetstream.FetchMaxWait(5*time.Second))
				if err != nil {
					if errors.Is(err, nats.ErrTimeout) || errors.Is(err, context.DeadlineExceeded) {
						continue
					}
					log.Error().Err(err).Str("scraper_name", name).Msg("Failed to fetch messages")
					time.Sleep(1 * time.Second)
					continue
				}

				for msg := range batch.Messages() {
					log.Info().Str("scraper_name", name).Msg("Consuming message")

					// Process the message first, then acknowledge
					err = s.Consume(ctx, msg)
					if err != nil {
						log.Error().Err(err).Str("scraper_name", name).Msg("Failed to consume message")
						// On error, we still acknowledge to avoid reprocessing bad messages
					}

					// Acknowledge the message after processing
					err := msg.Ack()
					if err != nil {
						log.Error().Err(err).Str("scraper_name", name).Msg("Failed to acknowledge message")
					} else {
						log.Info().Str("scraper_name", name).Msg("Acknowledged message")
					}
				}
				if batch.Error() != nil {
					log.Error().Err(batch.Error()).Str("scraper_name", name).Msg("Error during message batch processing")
				}
			}
		}(scraper)
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
