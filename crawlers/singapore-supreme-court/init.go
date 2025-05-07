package singapore_supreme_court

import (
	"github.com/LexiconIndonesia/crawler-http-service/common/crawler"
)

// init registers the E-Litigation crawler with the crawler registry
func init() {
	// Register the E-Litigation crawler creator function
	crawler.RegisterCrawler("elitigation", CreateELitigationCrawler)
}

// CreateELitigationCrawler creates an E-Litigation crawler
func CreateELitigationCrawler(config crawler.DataSourceConfig, baseConfig crawler.BaseCrawlerConfig, broker crawler.MessageBroker) (crawler.Crawler, error) {
	// Convert config to the specific type
	elitigationConfig, ok := config.(crawler.ElitigationSGConfig)
	if !ok {
		return nil, crawler.ErrInvalidConfig
	}

	// Create and return the actual crawler implementation
	return NewELitigationCrawler(elitigationConfig, baseConfig, broker)
}
