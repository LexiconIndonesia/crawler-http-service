package open_sanction

import (
	"github.com/adryanev/go-http-service-template/common"
	crawler "github.com/adryanev/go-http-service-template/crawlers"
)

const (
	CRAWLER_DOMAIN string = "www.opensanctions.org"
)

var (
	Config = crawler.NewCrawlerConfig(common.OpenSanction, CRAWLER_DOMAIN)
)
