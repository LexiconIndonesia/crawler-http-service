package crawler

import (
	"fmt"

	"github.com/adryanev/go-http-service-template/common"
	"github.com/adryanev/go-http-service-template/repository"
)

// GetCrawlerByType creates a new crawler instance based on the specified type
func GetCrawlerByType(crawlerType common.CrawlerType, service CrawlerService, dataSource repository.DataSource) (Crawler, error) {
	switch crawlerType {
	case common.IndonesiaSupremeCourt:
		return NewIndonesiaSupremeCourtCrawler(service, dataSource), nil
	case common.SingaporeSupremeCourt:
		return NewSingaporeSupremeCourtCrawler(service, dataSource), nil
	case common.LKPPBlacklist:
		return NewLKPPBlacklistCrawler(service, dataSource), nil
	default:
		return nil, fmt.Errorf("unsupported crawler type: %s", crawlerType)
	}
}
