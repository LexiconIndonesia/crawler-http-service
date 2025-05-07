package indonesia_supreme_court

import (
	"github.com/LexiconIndonesia/crawler-http-service/common/crawler"
)

// init registers the MahkamahAgung crawler with the crawler registry
func init() {
	// Register the MahkamahAgung crawler creator function
	crawler.RegisterCrawler("mahkamahagung", CreateMahkamahAgungCrawler)
}

// CreateMahkamahAgungCrawler creates a MahkamahAgungCrawler
func CreateMahkamahAgungCrawler(config crawler.DataSourceConfig, baseConfig crawler.BaseCrawlerConfig, broker crawler.MessageBroker) (crawler.Crawler, error) {
	// Convert config to the specific type
	mahkamahAgungConfig, ok := config.(crawler.MahkamahAgungConfig)
	if !ok {
		return nil, crawler.ErrInvalidConfig
	}

	// Create and return the actual crawler implementation
	return NewMahkamahAgungCrawler(mahkamahAgungConfig, baseConfig, broker)
}
