package crawler

import (
	"time"

	"github.com/adryanev/go-http-service-template/common"
	"github.com/adryanev/go-http-service-template/repository"
)

// CrawlerFactory creates crawler instances based on type
type CrawlerFactory struct {
	service CrawlerService
}

// NewCrawlerFactory creates a new crawler factory
func NewCrawlerFactory(service CrawlerService) *CrawlerFactory {
	return &CrawlerFactory{
		service: service,
	}
}

// CreateCrawler creates a crawler instance based on the specified type and domain
func (f *CrawlerFactory) CreateCrawler(crawlerType common.CrawlerType, dataSource repository.DataSource) (Crawler, error) {
	return GetCrawlerByType(crawlerType, f.service, dataSource)
}

// Factory functions for specific crawlers implemented in their respective packages
// These functions avoid import cycles by keeping the factory implementation
// independent of the specific crawler packages

// NewIndonesiaSupremeCourtCrawler creates an Indonesia Supreme Court crawler
func NewIndonesiaSupremeCourtCrawler(service CrawlerService, dataSource repository.DataSource) Crawler {
	config := NewCrawlerConfig(common.IndonesiaSupremeCourt, dataSource)
	baseCrawler := NewBaseCrawler(config, service)

	// Initialize crawler with the base crawler
	crawler := &IndonesiaSupremeCourtCrawler{
		baseCrawler: baseCrawler,
	}

	return crawler
}

// NewSingaporeSupremeCourtCrawler creates a Singapore Supreme Court crawler
func NewSingaporeSupremeCourtCrawler(service CrawlerService, dataSource repository.DataSource) Crawler {
	config := NewCrawlerConfig(common.SingaporeSupremeCourt, dataSource)
	baseCrawler := NewBaseCrawler(config, service)

	// Set custom browser options if needed
	baseCrawler.BrowserOpts.WaitAfterLoad = 3 * time.Second

	// Initialize crawler with the base crawler
	crawler := &SingaporeSupremeCourtCrawler{
		baseCrawler: baseCrawler,
	}

	return crawler
}

// NewLKPPBlacklistCrawler creates a LKPP Blacklist crawler
func NewLKPPBlacklistCrawler(service CrawlerService, dataSource repository.DataSource) Crawler {
	config := NewCrawlerConfig(common.LKPPBlacklist, dataSource)
	baseCrawler := NewBaseCrawler(config, service)

	// Set custom browser options if needed
	baseCrawler.BrowserOpts.WaitAfterLoad = 5 * time.Second // longer wait for dynamic content

	// Initialize crawler with the base crawler
	crawler := &LKPPBlacklistCrawler{
		baseCrawler: baseCrawler,
	}

	return crawler
}
