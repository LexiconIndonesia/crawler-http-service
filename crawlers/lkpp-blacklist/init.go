package lkpp_blacklist

import (
	"github.com/LexiconIndonesia/crawler-http-service/common/crawler"
)

// init registers the LKPP blacklist crawler with the crawler registry
func init() {
	// Register the LKPP blacklist crawler creator function
	crawler.RegisterCrawler("lkpp_blacklist", CreateLKPPBlacklistCrawler)
}

// CreateLKPPBlacklistCrawler creates a LKPP blacklist crawler
func CreateLKPPBlacklistCrawler(config crawler.DataSourceConfig, baseConfig crawler.BaseCrawlerConfig, broker crawler.MessageBroker) (crawler.Crawler, error) {
	// First check if we have a specific LKPPBlacklistConfig
	lkppConfig, ok := config.(LKPPBlacklistConfig)
	if ok {
		// If we have a specific config, use it
		return NewLKPPBlacklistCrawler(lkppConfig, baseConfig, broker)
	}

	// If not, create a default config
	// We can't directly convert from another config type, so we'll create a new one with defaults
	lkppConfig = LKPPBlacklistConfig{
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
