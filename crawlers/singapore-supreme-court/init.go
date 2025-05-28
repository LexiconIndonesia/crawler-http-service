package singapore_supreme_court

import (
	"github.com/LexiconIndonesia/crawler-http-service/common/crawler"
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
func CreateCrawler(dataSource repository.DataSource, baseConfig crawler.BaseCrawlerConfig, broker messaging.MessageBroker) (crawler.Crawler, error) {

	log.Info().Msgf("Creating Singapore Supreme Court crawler for data source %s", dataSource.Name)
	config, err := crawler.UnmarshalSingaporeSupremeCourtConfig(dataSource.Config)
	if err != nil {
		log.Error().Err(err).Msg("Failed to unmarshal config")
		return nil, err

	}

	// Create and return the actual crawler implementation
	return NewSingaporeSupremeCourtCrawler(config, baseConfig, broker)
}

func CreateScraper(dataSource repository.DataSource, baseConfig crawler.BaseScraperConfig, broker messaging.MessageBroker) (crawler.Scraper, error) {

	log.Info().Msgf("Creating Singapore Supreme Court scraper for data source %s", dataSource.Name)
	config, err := crawler.UnmarshalSingaporeSupremeCourtConfig(dataSource.Config)
	if err != nil {
		log.Error().Err(err).Msg("Failed to unmarshal config")
		return nil, err

	}

	// Create and return the actual scraper implementation
	return NewSingaporeSupremeCourtScraper(config, baseConfig, broker)
}
