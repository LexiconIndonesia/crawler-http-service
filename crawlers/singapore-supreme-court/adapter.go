package singapore_supreme_court

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/adryanev/go-http-service-template/common"
	crawler "github.com/adryanev/go-http-service-template/crawlers"
	"github.com/go-rod/rod"
	"github.com/rs/zerolog/log"
)

// CrawlerAdapter adapts the Singapore Supreme Court crawler to the common crawler interface
type CrawlerAdapter struct {
	*crawler.BaseCrawler
}

// NewCrawlerAdapter creates a new adapter for the Singapore Supreme Court crawler
func NewCrawlerAdapter(service crawler.CrawlerService) *CrawlerAdapter {
	config := crawler.NewCrawlerConfig(common.SingaporeSupremeCourt, CRAWLER_DOMAIN)
	baseCrawler := crawler.NewBaseCrawler(config, service)

	// Set custom browser options if needed
	baseCrawler.BrowserOpts.WaitAfterLoad = 3 * time.Second

	return &CrawlerAdapter{
		BaseCrawler: baseCrawler,
	}
}

// ExtractElements implements the specific extraction logic for Singapore Supreme Court
func (c *CrawlerAdapter) ExtractElements(ctx context.Context, page *rod.Page) ([]crawler.Item, error) {
	log.Info().Str("crawler", c.Config.Name).Msg("Extracting elements from Singapore Supreme Court")

	var items []crawler.Item

	// Wait for page to be loaded (adjust selector as needed)
	err := page.WaitLoad()
	if err != nil {
		return nil, fmt.Errorf("failed to wait for page load: %w", err)
	}

	// Example: extract judgment items
	// Note: These selectors need to be adjusted based on the actual page structure
	judgmentElements, err := page.Elements("div.judgment-item, div.case-item")
	if err != nil {
		return nil, fmt.Errorf("failed to find judgment elements: %w", err)
	}

	for _, element := range judgmentElements {
		// Extract title
		titleElement, err := element.Element("h3.judgment-title, h3.case-title")
		if err != nil {
			log.Warn().Err(err).Msg("Failed to find title element, skipping item")
			continue
		}

		title, err := titleElement.Text()
		if err != nil {
			log.Warn().Err(err).Msg("Failed to extract title text, skipping item")
			continue
		}

		// Extract link to judgment (PDF)
		var pdfURL string
		linkElement, err := element.Element("a[href*='.pdf']")
		if err == nil {
			href, err := linkElement.Attribute("href")
			if err == nil && href != nil {
				pdfURL = *href
				if !strings.HasPrefix(pdfURL, "http") {
					// Handle relative URLs
					pdfURL = fmt.Sprintf("https://%s%s", CRAWLER_DOMAIN, pdfURL)
				}
			}
		}

		// Extract date
		var date string
		dateElement, err := element.Element("span.date, div.date")
		if err == nil {
			date, _ = dateElement.Text()
		}

		// Take a screenshot of the element (for debugging)
		screenshotDir := "screenshots/singapore-supreme-court"
		if err := os.MkdirAll(screenshotDir, 0755); err == nil {
			safeFilename := fmt.Sprintf("judgment-%d", time.Now().Unix())
			filename := filepath.Join(screenshotDir, safeFilename+".png")

			// Capture screenshot using MustScreenshot
			data := element.MustScreenshot()
			_ = os.WriteFile(filename, data, 0644)
		}

		// Create item
		item := crawler.Item{
			URL:     pdfURL,
			Title:   title,
			Content: "", // PDF link, so no direct content
			Metadata: map[string]interface{}{
				"source": "Singapore Supreme Court",
				"date":   date,
			},
		}

		items = append(items, item)
	}

	log.Info().
		Str("crawler", c.Config.Name).
		Int("items_count", len(items)).
		Msg("Extracted items from Singapore Supreme Court")

	return items, nil
}

// CrawlByKeyword implements keyword-based search for Singapore Supreme Court
func (c *CrawlerAdapter) CrawlByKeyword(ctx context.Context, keyword string) error {
	log.Info().
		Str("crawler", c.Config.Name).
		Str("keyword", keyword).
		Msg("Crawling Singapore Supreme Court by keyword")

	// Form the search URL with the given keyword
	// Example: https://www.supremecourt.gov.sg/news/judgments?keyword=corruption
	searchURL := fmt.Sprintf("https://www.%s/news/judgments?keyword=%s",
		CRAWLER_DOMAIN, keyword)

	// Navigate to search URL
	page, err := c.Navigate(ctx, searchURL)
	if err != nil {
		return fmt.Errorf("failed to navigate to search URL: %w", err)
	}
	defer page.Close()

	// Extract items from search results
	items, err := c.ExtractElements(ctx, page)
	if err != nil {
		return fmt.Errorf("failed to extract items from search results: %w", err)
	}

	log.Info().
		Str("crawler", c.Config.Name).
		Str("keyword", keyword).
		Int("items_count", len(items)).
		Msg("Extracted items from keyword search")

	return nil
}
