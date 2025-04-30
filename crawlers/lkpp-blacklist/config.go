package lkpp_blacklist

import (
	"github.com/adryanev/go-http-service-template/common"
	crawler "github.com/adryanev/go-http-service-template/crawlers"
)

const (
	CRAWLER_DOMAIN string = "blacklist.lkpp.go.id"
)

var (
	Config = crawler.NewCrawlerConfig(common.LKPPBlacklist, CRAWLER_DOMAIN)
)
