package crawler

import (
	"time"

	"github.com/adryanev/go-http-service-template/common"
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
func (f *CrawlerFactory) CreateCrawler(crawlerType common.CrawlerType, domain string) (Crawler, error) {
	return CreateCrawlerByType(crawlerType, f.service)
}

// Factory functions for specific crawlers implemented in their respective packages
// These functions avoid import cycles by keeping the factory implementation
// independent of the specific crawler packages

// NewIndonesiaSupremeCourtCrawler creates an Indonesia Supreme Court crawler
func NewIndonesiaSupremeCourtCrawler(service CrawlerService) Crawler {
	config := NewCrawlerConfig(common.IndonesiaSupremeCourt, "mahkamahagung.go.id")
	baseCrawler := NewBaseCrawler(config, service)

	return &IndonesiaSupremeCourtCrawler{
		BaseCrawler: baseCrawler,
	}
}

// NewSingaporeSupremeCourtCrawler creates a Singapore Supreme Court crawler
func NewSingaporeSupremeCourtCrawler(service CrawlerService) Crawler {
	config := NewCrawlerConfig(common.SingaporeSupremeCourt, "supremecourt.gov.sg")
	baseCrawler := NewBaseCrawler(config, service)

	// Set custom browser options if needed
	baseCrawler.BrowserOpts.WaitAfterLoad = 3 * time.Second

	return &SingaporeSupremeCourtCrawler{
		BaseCrawler: baseCrawler,
	}
}

// NewLKPPBlacklistCrawler creates a LKPP Blacklist crawler
func NewLKPPBlacklistCrawler(service CrawlerService) Crawler {
	config := NewCrawlerConfig(common.LKPPBlacklist, "blacklist.lkpp.go.id")
	baseCrawler := NewBaseCrawler(config, service)

	// Set custom browser options if needed
	baseCrawler.BrowserOpts.WaitAfterLoad = 5 * time.Second // longer wait for dynamic content

	return &LKPPBlacklistCrawler{
		BaseCrawler: baseCrawler,
	}
}
