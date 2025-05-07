package indonesia_supreme_court

import (
	"context"

	"github.com/LexiconIndonesia/crawler-http-service/common/crawler"
	"github.com/LexiconIndonesia/crawler-http-service/repository"
	"github.com/go-rod/rod"
	"github.com/rs/zerolog/log"
)

// MahkamahAgungCrawler is a crawler for the Indonesia Supreme Court
type MahkamahAgungCrawler struct {
	crawler.BaseCrawler
	Config crawler.MahkamahAgungConfig
}

// NewMahkamahAgungCrawler creates a new MahkamahAgungCrawler
func NewMahkamahAgungCrawler(config crawler.MahkamahAgungConfig, baseConfig crawler.BaseCrawlerConfig, broker crawler.MessageBroker) (*MahkamahAgungCrawler, error) {
	// This is just a stub - no actual implementation
	return &MahkamahAgungCrawler{
		BaseCrawler: crawler.BaseCrawler{
			Config:        baseConfig,
			MessageBroker: broker,
		},
		Config: config,
	}, nil
}

// CrawlAll crawls all pages from the Indonesia Supreme Court
func (c *MahkamahAgungCrawler) CrawlAll(ctx context.Context) error {
	// Not implemented as per requirements
	log.Error().Msg("CrawlAll method not implemented")
	return crawler.ErrNotImplemented
}

// CrawlByKeyword crawls pages from the Indonesia Supreme Court based on a search term
func (c *MahkamahAgungCrawler) CrawlByKeyword(ctx context.Context, keyword string) error {
	// Not implemented as per requirements
	log.Error().Msg("CrawlByKeyword method not implemented")
	return crawler.ErrNotImplemented
}

// CrawlByURL crawls a specific URL from the Indonesia Supreme Court
func (c *MahkamahAgungCrawler) CrawlByURL(ctx context.Context, url string) error {
	// Not implemented as per requirements
	log.Error().Msg("CrawlByURL method not implemented")
	return crawler.ErrNotImplemented
}

// ExtractElements extracts URL frontiers from a page
func (c *MahkamahAgungCrawler) ExtractElements(ctx context.Context, page *rod.Page) ([]repository.UrlFrontier, error) {
	// Not implemented as per requirements
	log.Error().Msg("ExtractElements method not implemented")
	return nil, crawler.ErrNotImplemented
}
