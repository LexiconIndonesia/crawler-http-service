package lkpp_blacklist

import (
	"github.com/LexiconIndonesia/crawler-http-service/common/crawler"
	"github.com/LexiconIndonesia/crawler-http-service/common/messaging"
	"github.com/LexiconIndonesia/crawler-http-service/repository"
)

// init registers the LKPP blacklist crawler with the crawler registry
func init() {
	// Register the LKPP blacklist crawler creator function
	crawler.RegisterCrawler("lkpp-blacklist", CreateLKPPBlacklistCrawler)
}

// CreateLKPPBlacklistCrawler creates a LKPP blacklist crawler
func CreateLKPPBlacklistCrawler(dataSource repository.DataSource, baseConfig crawler.BaseCrawlerConfig, broker messaging.MessageBroker) (crawler.Crawler, error) {

	// TODO: Create the real config
	lkppConfig := LKPPBlacklistConfig{
		BaseConfig: crawler.BaseConfig{
			PaginationSelector: ".pagination",
			DetailLinkSelector: ".detail-link",
			MaxPages:           10,
			Delay:              1000,
		},
		BaseURL:         "https://blacklist.lkpp.go.id",
		SearchFormURL:   "https://blacklist.lkpp.go.id/search",
		CompanySelector: ".company-item",
	}

	// Create and return the actual crawler implementation
	return NewLKPPBlacklistCrawler(lkppConfig, baseConfig, broker)
}
