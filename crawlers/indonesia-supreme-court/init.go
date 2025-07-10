package isc

import (
	"github.com/LexiconIndonesia/crawler-http-service/common/crawler"
	"github.com/LexiconIndonesia/crawler-http-service/common/db"
	"github.com/LexiconIndonesia/crawler-http-service/common/messaging"
	"github.com/LexiconIndonesia/crawler-http-service/repository"
)

// init registers the MahkamahAgung crawler with the crawler registry
func init() {
	// Register the MahkamahAgung crawler creator function
	crawler.RegisterCrawler("indonesia-supreme-court", CreateIndonesiaSupremeCourt)
	crawler.RegisterScraper("indonesia-supreme-court", CreateIndonesiaSupremeCourtScraper)
}

// CreateIndonesiaSupremeCourt creates a MahkamahAgungCrawler
func CreateIndonesiaSupremeCourt(db *db.DB, dataSource repository.DataSource, baseConfig crawler.BaseCrawlerConfig, broker *messaging.NatsBroker) (crawler.Crawler, error) {
	// generate config
	config, err := crawler.UnmarshalIndonesiaSupremeCourtConfig(dataSource.Config)
	if err != nil {
		return nil, err
	}

	// Create and return the actual crawler implementation
	return NewIndonesiaSupremeCourtCrawler(db, config, baseConfig, broker)
}

func CreateIndonesiaSupremeCourtScraper(db *db.DB, dataSource repository.DataSource, baseConfig crawler.BaseScraperConfig, broker *messaging.NatsBroker) (crawler.Scraper, error) {
	// generate config
	config, err := crawler.UnmarshalIndonesiaSupremeCourtConfig(dataSource.Config)
	if err != nil {
		return nil, err
	}

	// Create and return the actual crawler implementation
	return NewIndonesiaSupremeCourtScraper(db, config, baseConfig, broker)
}
