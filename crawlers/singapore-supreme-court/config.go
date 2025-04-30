package singapore_supreme_court

import (
	"github.com/adryanev/go-http-service-template/common"
	crawler "github.com/adryanev/go-http-service-template/crawlers"
)

const (
	CRAWLER_DOMAIN string = "supremecourt.gov.sg"
)

var (
	Config = crawler.NewCrawlerConfig(common.SingaporeSupremeCourt, CRAWLER_DOMAIN)
)
