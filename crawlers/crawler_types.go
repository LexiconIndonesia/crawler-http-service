package crawler

import (
	"context"

	"github.com/go-rod/rod"
)

// IndonesiaSupremeCourtCrawler implements the Crawler interface for the Indonesia Supreme Court website
type IndonesiaSupremeCourtCrawler struct {
	baseCrawler *BaseCrawler
	browser     *rod.Browser
}

// Ensure IndonesiaSupremeCourtCrawler implements the Crawler interface
var _ Crawler = (*IndonesiaSupremeCourtCrawler)(nil)

// Setup performs initialization for the Indonesia Supreme Court crawler
func (c *IndonesiaSupremeCourtCrawler) Setup(ctx context.Context) error {
	return c.baseCrawler.Setup(ctx)
}

// Teardown cleans up resources for the Indonesia Supreme Court crawler
func (c *IndonesiaSupremeCourtCrawler) Teardown(ctx context.Context) error {
	return c.baseCrawler.Teardown(ctx)
}

// CrawlAll crawls all available URLs for the Indonesia Supreme Court
func (c *IndonesiaSupremeCourtCrawler) CrawlAll(ctx context.Context) error {
	return c.baseCrawler.CrawlAll(ctx)
}

// CrawlByKeyword crawls URLs related to the given keyword for the Indonesia Supreme Court
func (c *IndonesiaSupremeCourtCrawler) CrawlByKeyword(ctx context.Context, keyword string) error {
	return c.baseCrawler.CrawlByKeyword(ctx, keyword)
}

// CrawlByURL crawls a specific URL for the Indonesia Supreme Court
func (c *IndonesiaSupremeCourtCrawler) CrawlByURL(ctx context.Context, url string) error {
	return c.baseCrawler.CrawlByURL(ctx, url)
}

// ExtractElements extracts data from a page for the Indonesia Supreme Court
func (c *IndonesiaSupremeCourtCrawler) ExtractElements(ctx context.Context, page *rod.Page) ([]Item, error) {
	return c.baseCrawler.ExtractElements(ctx, page)
}

// Navigate goes to a specific URL for the Indonesia Supreme Court
func (c *IndonesiaSupremeCourtCrawler) Navigate(ctx context.Context, url string) (*rod.Page, error) {
	return c.baseCrawler.Navigate(ctx, url)
}

// Consume handles messages from a message broker for the Indonesia Supreme Court
func (c *IndonesiaSupremeCourtCrawler) Consume(ctx context.Context, message []byte) error {
	return c.baseCrawler.Consume(ctx, message)
}

// SingaporeSupremeCourtCrawler implements the Crawler interface for the Singapore Supreme Court website
type SingaporeSupremeCourtCrawler struct {
	baseCrawler *BaseCrawler
}

// Ensure SingaporeSupremeCourtCrawler implements the Crawler interface
var _ Crawler = (*SingaporeSupremeCourtCrawler)(nil)

// Setup performs initialization for the Singapore Supreme Court crawler
func (c *SingaporeSupremeCourtCrawler) Setup(ctx context.Context) error {
	return c.baseCrawler.Setup(ctx)
}

// Teardown cleans up resources for the Singapore Supreme Court crawler
func (c *SingaporeSupremeCourtCrawler) Teardown(ctx context.Context) error {
	return c.baseCrawler.Teardown(ctx)
}

// CrawlAll crawls all available URLs for the Singapore Supreme Court
func (c *SingaporeSupremeCourtCrawler) CrawlAll(ctx context.Context) error {
	return c.baseCrawler.CrawlAll(ctx)
}

// CrawlByKeyword crawls URLs related to the given keyword for the Singapore Supreme Court
func (c *SingaporeSupremeCourtCrawler) CrawlByKeyword(ctx context.Context, keyword string) error {
	return c.baseCrawler.CrawlByKeyword(ctx, keyword)
}

// CrawlByURL crawls a specific URL for the Singapore Supreme Court
func (c *SingaporeSupremeCourtCrawler) CrawlByURL(ctx context.Context, url string) error {
	return c.baseCrawler.CrawlByURL(ctx, url)
}

// ExtractElements extracts data from a page for the Singapore Supreme Court
func (c *SingaporeSupremeCourtCrawler) ExtractElements(ctx context.Context, page *rod.Page) ([]Item, error) {
	return c.baseCrawler.ExtractElements(ctx, page)
}

// Navigate goes to a specific URL for the Singapore Supreme Court
func (c *SingaporeSupremeCourtCrawler) Navigate(ctx context.Context, url string) (*rod.Page, error) {
	return c.baseCrawler.Navigate(ctx, url)
}

// Consume handles messages from a message broker for the Singapore Supreme Court
func (c *SingaporeSupremeCourtCrawler) Consume(ctx context.Context, message []byte) error {
	return c.baseCrawler.Consume(ctx, message)
}

// LKPPBlacklistCrawler implements the Crawler interface for the LKPP Blacklist website
type LKPPBlacklistCrawler struct {
	baseCrawler *BaseCrawler
}

// Ensure LKPPBlacklistCrawler implements the Crawler interface
var _ Crawler = (*LKPPBlacklistCrawler)(nil)

// Setup performs initialization for the LKPP Blacklist crawler
func (c *LKPPBlacklistCrawler) Setup(ctx context.Context) error {
	return c.baseCrawler.Setup(ctx)
}

// Teardown cleans up resources for the LKPP Blacklist crawler
func (c *LKPPBlacklistCrawler) Teardown(ctx context.Context) error {
	return c.baseCrawler.Teardown(ctx)
}

// CrawlAll crawls all available URLs for the LKPP Blacklist
func (c *LKPPBlacklistCrawler) CrawlAll(ctx context.Context) error {
	return c.baseCrawler.CrawlAll(ctx)
}

// CrawlByKeyword crawls URLs related to the given keyword for the LKPP Blacklist
func (c *LKPPBlacklistCrawler) CrawlByKeyword(ctx context.Context, keyword string) error {
	return c.baseCrawler.CrawlByKeyword(ctx, keyword)
}

// CrawlByURL crawls a specific URL for the LKPP Blacklist
func (c *LKPPBlacklistCrawler) CrawlByURL(ctx context.Context, url string) error {
	return c.baseCrawler.CrawlByURL(ctx, url)
}

// ExtractElements extracts data from a page for the LKPP Blacklist
func (c *LKPPBlacklistCrawler) ExtractElements(ctx context.Context, page *rod.Page) ([]Item, error) {
	return c.baseCrawler.ExtractElements(ctx, page)
}

// Navigate goes to a specific URL for the LKPP Blacklist
func (c *LKPPBlacklistCrawler) Navigate(ctx context.Context, url string) (*rod.Page, error) {
	return c.baseCrawler.Navigate(ctx, url)
}

// Consume handles messages from a message broker for the LKPP Blacklist
func (c *LKPPBlacklistCrawler) Consume(ctx context.Context, message []byte) error {
	return c.baseCrawler.Consume(ctx, message)
}
