package lkpp_blacklist

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	crawler "github.com/adryanev/go-http-service-template/crawlers"
	"github.com/go-rod/rod"
	"github.com/rs/zerolog/log"
)

// LKPPBlacklistCrawler implements a crawler for the LKPP Blacklist
type LKPPBlacklistCrawler struct {
	*crawler.BaseCrawler
}

// NewCrawler creates a new LKPP Blacklist crawler
func NewCrawler(service crawler.CrawlerService) *LKPPBlacklistCrawler {
	baseCrawler := crawler.NewBaseCrawler(Config, service)

	// Set custom browser options
	baseCrawler.BrowserOpts.WaitAfterLoad = 5 * time.Second // longer wait for dynamic content

	return &LKPPBlacklistCrawler{
		BaseCrawler: baseCrawler,
	}
}

// ExtractElements implements the specific extraction logic for LKPP Blacklist
func (c *LKPPBlacklistCrawler) ExtractElements(ctx context.Context, page *rod.Page) ([]crawler.Item, error) {
	log.Info().Str("crawler", c.Config.Name).Msg("Extracting elements from LKPP Blacklist")

	var items []crawler.Item

	// Wait for page to be fully loaded
	err := page.WaitLoad()
	if err != nil {
		return nil, fmt.Errorf("failed to wait for page load: %w", err)
	}

	// Wait for table to be loaded (dynamic content)
	err = waitForSelector(page, "table.blacklist-table, table.table-blacklist")
	if err != nil {
		return nil, fmt.Errorf("failed to find blacklist table: %w", err)
	}

	// Example: extract blacklist table rows
	// Note: These selectors need to be adjusted based on the actual page structure
	rows, err := page.Elements("table.blacklist-table tbody tr, table.table-blacklist tbody tr")
	if err != nil {
		return nil, fmt.Errorf("failed to find table rows: %w", err)
	}

	for i, row := range rows {
		// Extract cells
		cells, err := row.Elements("td")
		if err != nil || len(cells) < 4 {
			log.Warn().Int("row", i).Err(err).Msg("Skipping row with insufficient cells")
			continue
		}

		// Extract company name (assume it's in the second column)
		companyName, err := cells[1].Text()
		if err != nil {
			log.Warn().Int("row", i).Err(err).Msg("Failed to extract company name")
			continue
		}

		// Extract blacklist reason (assume it's in the third column)
		reason, err := cells[2].Text()
		if err != nil {
			log.Warn().Int("row", i).Err(err).Msg("Failed to extract blacklist reason")
			continue
		}

		// Extract blacklist period (assume it's in the fourth column)
		period, err := cells[3].Text()
		if err != nil {
			log.Warn().Int("row", i).Err(err).Msg("Failed to extract blacklist period")
			continue
		}

		// Get more details if available
		var details string
		detailsButton, err := row.Element("a.details-button, button.details")
		if err == nil {
			// Click to show details
			detailsButton.MustClick()

			// Wait for details to load
			time.Sleep(1 * time.Second)

			// Try to find the details content
			detailsElement, err := page.Element("div.blacklist-details, div.modal-body")
			if err == nil {
				details, _ = detailsElement.Text()
			}

			// Close details modal if it exists
			closeButton, err := page.Element("button.close, a.close-modal")
			if err == nil {
				closeButton.MustClick()
			}
		}

		// Create item
		item := crawler.Item{
			URL:     page.MustInfo().URL,
			Title:   companyName,
			Content: reason,
			Metadata: map[string]interface{}{
				"source":     "LKPP Blacklist",
				"period":     period,
				"details":    details,
				"row_number": i + 1,
			},
		}

		items = append(items, item)
	}

	// Take a screenshot of the page for reference
	screenshotDir := "screenshots/lkpp-blacklist"
	if err := os.MkdirAll(screenshotDir, 0755); err == nil {
		timestamp := time.Now().Format("20060102-150405")
		filename := filepath.Join(screenshotDir, fmt.Sprintf("page-%s.png", timestamp))

		data := page.MustScreenshot()
		_ = os.WriteFile(filename, data, 0644)
	}

	log.Info().
		Str("crawler", c.Config.Name).
		Int("items_count", len(items)).
		Msg("Extracted items from LKPP Blacklist")

	return items, nil
}

// CrawlByKeyword implements keyword-based search for LKPP Blacklist
func (c *LKPPBlacklistCrawler) CrawlByKeyword(ctx context.Context, keyword string) error {
	log.Info().
		Str("crawler", c.Config.Name).
		Str("keyword", keyword).
		Msg("Crawling LKPP Blacklist by keyword")

	// Form the search URL with the given keyword
	searchURL := fmt.Sprintf("https://%s/daftar-hitam/pencarian?q=%s",
		CRAWLER_DOMAIN, keyword)

	// Navigate to search URL
	page, err := c.Navigate(ctx, searchURL)
	if err != nil {
		return fmt.Errorf("failed to navigate to search URL: %w", err)
	}
	defer page.Close()

	// Wait for search results to load
	err = page.WaitLoad()
	if err != nil {
		return fmt.Errorf("failed to wait for search results: %w", err)
	}

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

	// Process pagination if available
	paginationElements, err := page.Elements("ul.pagination li a")
	if err == nil && len(paginationElements) > 0 {
		log.Info().Msg("Found pagination, processing additional pages")

		// Get the number of pages from pagination
		var lastPage int
		for _, el := range paginationElements {
			text, err := el.Text()
			if err == nil {
				num, err := strconv.Atoi(strings.TrimSpace(text))
				if err == nil && num > lastPage {
					lastPage = num
				}
			}
		}

		if lastPage > 1 {
			// Process additional pages
			for i := 2; i <= lastPage; i++ {
				pageURL := fmt.Sprintf("%s&page=%d", searchURL, i)

				// Navigate to page
				pagePage, err := c.Navigate(ctx, pageURL)
				if err != nil {
					log.Error().Err(err).Int("page", i).Msg("Failed to navigate to page")
					continue
				}

				// Extract items from this page
				pageItems, err := c.ExtractElements(ctx, pagePage)
				if err != nil {
					log.Error().Err(err).Int("page", i).Msg("Failed to extract items from page")
					pagePage.Close()
					continue
				}

				// Close the page when done
				pagePage.Close()

				log.Info().
					Int("page", i).
					Int("items_count", len(pageItems)).
					Msg("Extracted items from pagination page")
			}
		}
	}

	return nil
}

// Helper function to wait for an element to appear
func waitForSelector(page *rod.Page, selector string) error {
	timeout := 10 * time.Second
	start := time.Now()

	for {
		if time.Since(start) > timeout {
			return fmt.Errorf("timeout waiting for element: %s", selector)
		}

		// Try to find the element
		_, err := page.Element(selector)
		if err == nil {
			return nil
		}

		// Wait a bit before trying again
		time.Sleep(500 * time.Millisecond)
	}
}
