package crawler

import (
	"fmt"

	"github.com/adryanev/go-http-service-template/common"
)

// GetCrawlerByType creates a new crawler instance based on the specified type
func GetCrawlerByType(crawlerType common.CrawlerType, service CrawlerService) (Crawler, error) {
	switch crawlerType {
	case common.IndonesiaSupremeCourt:
		config := NewCrawlerConfig(common.IndonesiaSupremeCourt, "mahkamahagung.go.id")
		baseCrawler := NewBaseCrawler(config, service)
		return &IndonesiaSupremeCourtCrawler{BaseCrawler: baseCrawler}, nil
	case common.SingaporeSupremeCourt:
		config := NewCrawlerConfig(common.SingaporeSupremeCourt, "supremecourt.gov.sg")
		baseCrawler := NewBaseCrawler(config, service)
		return &SingaporeSupremeCourtCrawler{BaseCrawler: baseCrawler}, nil
	case common.LKPPBlacklist:
		config := NewCrawlerConfig(common.LKPPBlacklist, "blacklist.lkpp.go.id")
		baseCrawler := NewBaseCrawler(config, service)
		return &LKPPBlacklistCrawler{BaseCrawler: baseCrawler}, nil
	default:
		return nil, fmt.Errorf("unsupported crawler type: %s", crawlerType)
	}
}
