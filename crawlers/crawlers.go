package crawler

import (
	"github.com/adryanev/go-http-service-template/common"
)

// This file contains the main crawler implementations

// CreateCrawlerByType creates a new crawler instance based on the specified type
// This function is maintained for backward compatibility
func CreateCrawlerByType(crawlerType common.CrawlerType, service CrawlerService) (Crawler, error) {
	return GetCrawlerByType(crawlerType, service)
}
