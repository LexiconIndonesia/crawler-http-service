package crawler

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/rs/zerolog/log"
)

// IndonesiaSupremeCourtCrawler implements a crawler for Indonesia Supreme Court
type IndonesiaSupremeCourtCrawler struct {
	*BaseCrawler
}

// ExtractElements implements the specific extraction logic for Indonesia Supreme Court
func (c *IndonesiaSupremeCourtCrawler) ExtractElements(ctx context.Context, page *rod.Page) ([]Item, error) {
	log.Info().Str("crawler", c.Config.Name).Msg("Extracting elements from Indonesia Supreme Court")

	var items []Item

	// Example implementation - this would need to be customized based on the actual page structure
	elements, err := page.Elements("div.putusan-list")
	if err != nil {
		return nil, err
	}

	for _, element := range elements {
		// Extract title
		titleElement, err := element.Element("h3.putusan-title")
		if err != nil {
			continue
		}
		title, err := titleElement.Text()
		if err != nil {
			continue
		}

		// Extract link
		linkElement, err := element.Element("a.putusan-link")
		if err != nil {
			continue
		}
		href, err := linkElement.Attribute("href")
		if err != nil || href == nil {
			continue
		}

		// Extract content summary
		contentElement, err := element.Element("div.putusan-summary")
		if err != nil {
			continue
		}
		content, err := contentElement.Text()
		if err != nil {
			continue
		}

		// Create item
		item := Item{
			URL:     *href,
			Title:   title,
			Content: content,
			Metadata: map[string]interface{}{
				"source": "Indonesia Supreme Court",
			},
		}

		items = append(items, item)
	}

	return items, nil
}

// SingaporeSupremeCourtCrawler implements a crawler for Singapore Supreme Court
type SingaporeSupremeCourtCrawler struct {
	*BaseCrawler
}

// ExtractElements implements the specific extraction logic for Singapore Supreme Court
func (c *SingaporeSupremeCourtCrawler) ExtractElements(ctx context.Context, page *rod.Page) ([]Item, error) {
	log.Info().Str("crawler", c.Config.Name).Msg("Extracting elements from Singapore Supreme Court")

	var items []Item

	// Wait for page to be loaded
	err := page.WaitLoad()
	if err != nil {
		return nil, fmt.Errorf("failed to wait for page load: %w", err)
	}

	// Example: extract judgment items
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
					pdfURL = fmt.Sprintf("https://www.supremecourt.gov.sg%s", pdfURL)
				}
			}
		}

		// Extract date
		var date string
		dateElement, err := element.Element("span.date, div.date")
		if err == nil {
			date, _ = dateElement.Text()
		}

		// Create item
		item := Item{
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

	// Take a screenshot for debugging
	screenshotDir := "screenshots/singapore-supreme-court"
	if err := os.MkdirAll(screenshotDir, 0755); err == nil {
		filename := filepath.Join(screenshotDir, "page-"+strconv.FormatInt(time.Now().Unix(), 10)+".png")
		data := page.MustScreenshot()
		_ = os.WriteFile(filename, data, 0644)
	}

	log.Info().
		Str("crawler", c.Config.Name).
		Int("items_count", len(items)).
		Msg("Extracted items from Singapore Supreme Court")

	return items, nil
}

// CrawlByKeyword implements keyword-based search for Singapore Supreme Court
func (c *SingaporeSupremeCourtCrawler) CrawlByKeyword(ctx context.Context, keyword string) error {
	log.Info().
		Str("crawler", c.Config.Name).
		Str("keyword", keyword).
		Msg("Crawling Singapore Supreme Court by keyword")

	// Form the search URL with the given keyword
	searchURL := fmt.Sprintf("https://www.supremecourt.gov.sg/news/judgments?keyword=%s", keyword)

	// Navigate to search URL
	page, err := c.Navigate(ctx, searchURL)
	if err != nil {
		return fmt.Errorf("failed to navigate to search URL: %w", err)
	}
	defer page.Close()

	// Extract items from search results
	_, err = c.ExtractElements(ctx, page)
	if err != nil {
		return fmt.Errorf("failed to extract items from search results: %w", err)
	}

	return nil
}

// LKPPBlacklistCrawler implements a crawler for LKPP Blacklist
type LKPPBlacklistCrawler struct {
	*BaseCrawler
}

// ExtractElements implements the specific extraction logic for LKPP Blacklist
func (c *LKPPBlacklistCrawler) ExtractElements(ctx context.Context, page *rod.Page) ([]Item, error) {
	log.Info().Str("crawler", c.Config.Name).Msg("Extracting elements from LKPP Blacklist")

	var items []Item

	// Wait for page to be fully loaded
	err := page.WaitLoad()
	if err != nil {
		return nil, fmt.Errorf("failed to wait for page load: %w", err)
	}

	// Wait for table to be loaded (dynamic content)
	// Use a simple polling approach
	var tableFound bool
	for i := 0; i < 20; i++ { // Try for up to 10 seconds
		_, err := page.Element("table.blacklist-table, table.table-blacklist")
		if err == nil {
			tableFound = true
			break
		}
		time.Sleep(500 * time.Millisecond)
	}

	if !tableFound {
		return nil, fmt.Errorf("blacklist table not found")
	}

	// Example: extract blacklist table rows
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
		item := Item{
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
	searchURL := fmt.Sprintf("https://blacklist.lkpp.go.id/daftar-hitam/pencarian?q=%s", keyword)

	// Navigate to search URL
	page, err := c.Navigate(ctx, searchURL)
	if err != nil {
		return fmt.Errorf("failed to navigate to search URL: %w", err)
	}
	defer page.Close()

	// Extract items from search results
	_, err = c.ExtractElements(ctx, page)
	if err != nil {
		return fmt.Errorf("failed to extract items from search results: %w", err)
	}

	return nil
}
