package ssc

import (
	"github.com/LexiconIndonesia/crawler-http-service/common/crawler"
	"github.com/LexiconIndonesia/crawler-http-service/common/db"
	"github.com/LexiconIndonesia/crawler-http-service/common/messaging"
	"github.com/LexiconIndonesia/crawler-http-service/repository"
	"github.com/rs/zerolog/log"
)

// init registers the E-Litigation crawler with the crawler registry
func init() {
	// Register the E-Litigation crawler creator function
	crawler.RegisterCrawler("singapore-supreme-court", CreateCrawler)
	crawler.RegisterScraper("singapore-supreme-court", CreateScraper)
}

// CreateSingaporeSupremeCourtCrawler creates an E-Litigation crawler
func CreateCrawler(db *db.DB, dataSource repository.DataSource, baseConfig crawler.BaseCrawlerConfig, broker *messaging.NatsBroker) (crawler.Crawler, error) {
	log.Info().Msgf("Creating Singapore Supreme Court crawler for data source %s", dataSource.Name)
	config, err := crawler.UnmarshalSingaporeSupremeCourtConfig(dataSource.Config)
	if err != nil {
		log.Error().Err(err).Msg("Failed to unmarshal config")
		return nil, err

	}

	// Create and return the actual crawler implementation
	return NewSingaporeSupremeCourtCrawler(db, config, baseConfig, broker)
}

func CreateScraper(db *db.DB, dataSource repository.DataSource, baseConfig crawler.BaseScraperConfig, broker *messaging.NatsBroker) (crawler.Scraper, error) {
	log.Info().Msgf("Creating Singapore Supreme Court scraper for data source %s", dataSource.Name)
	config, err := crawler.UnmarshalSingaporeSupremeCourtConfig(dataSource.Config)
	if err != nil {
		log.Error().Err(err).Msg("Failed to unmarshal config")
		return nil, err

	}

	// Create and return the actual scraper implementation
	return NewSingaporeSupremeCourtScraper(db, config, baseConfig, broker)
}
