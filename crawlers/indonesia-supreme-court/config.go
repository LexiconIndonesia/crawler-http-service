package indonesia_supreme_court

import (
	"github.com/adryanev/go-http-service-template/common"
	crawler "github.com/adryanev/go-http-service-template/crawlers"
)

const (
	CRAWLER_DOMAIN string = "putusan3.mahkamahagung.go.id"
)

var (
	Config = crawler.NewCrawlerConfig(common.IndonesiaSupremeCourt, CRAWLER_DOMAIN)
)
