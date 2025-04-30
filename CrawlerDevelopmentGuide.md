# Crawler Development Guide

This guide explains how to develop and extend the crawler functionality in the project.

## Project Overview

This project is a web crawler service built using Go, designed to collect data from various websites by implementing custom crawlers for each data source. The system:

- Uses Go-Rod for web browser automation
- Stores crawled URLs and data in PostgreSQL
- Uses NATS JetStream for message passing and queue management
- Follows a modular architecture for easy extension

## Architecture

The crawler system consists of several key components:

1. **Crawler Interface**: Base interface that all crawlers implement
2. **BaseCrawler**: Provides common functionality for web crawlers
3. **Specific Crawlers**: Custom implementations for different websites
4. **CrawlerFactory**: Creates crawler instances based on type
5. **CrawlerService**: Provides database operations for crawlers
6. **NATS Handlers**: Manages message-based operations

## Developing a New Crawler

### 1. Create a New Crawler Directory

Create a directory for your crawler under the `crawlers/` directory:

```bash
mkdir crawlers/your-crawler-name
```

### 2. Define Crawler Type

Add your crawler type to the common package:

```go
// In common/types.go or similar file
const (
    // Existing types
    IndonesiaSupremeCourt CrawlerType = "indonesia-supreme-court"
    SingaporeSupremeCourt CrawlerType = "singapore-supreme-court"
    LKPPBlacklist        CrawlerType = "lkpp-blacklist"

    // Your new crawler type
    YourCrawlerName      CrawlerType = "your-crawler-name"
)
```

### 3. Implement the Crawler

Create the following files in your crawler directory:

#### a. crawler.go

This file implements the core crawler functionality:

```go
package your_crawler_name

import (
    "context"

    crawler "github.com/adryanev/go-http-service-template/crawlers"
    "github.com/go-rod/rod"
)

// YourCrawler implements the Crawler interface
type YourCrawler struct {
    *crawler.BaseCrawler
}

// ExtractElements implements the main scraping logic
func (c *YourCrawler) ExtractElements(ctx context.Context, page *rod.Page) ([]crawler.Item, error) {
    // Implement your extraction logic here
    // Example:
    items := []crawler.Item{}

    // Select elements on the page
    elements, err := page.Elements("your-css-selector")
    if err != nil {
        return nil, err
    }

    // Process each element
    for _, el := range elements {
        title, err := el.Text()
        if err != nil {
            continue
        }

        url, err := el.Attribute("href")
        if err != nil {
            continue
        }

        items = append(items, crawler.Item{
            URL:     *url,
            Title:   title,
            Content: "",
            Metadata: map[string]interface{}{
                "source": "your-crawler-name",
                // Add other metadata
            },
        })
    }

    return items, nil
}

// Optionally override other methods if needed
func (c *YourCrawler) CrawlByKeyword(ctx context.Context, keyword string) error {
    // Implement keyword search logic
    // Example: construct a search URL with the keyword
    searchURL := fmt.Sprintf("https://example.com/search?q=%s", keyword)
    return c.CrawlByURL(ctx, searchURL)
}
```

#### b. models.go (Optional)

Define any domain-specific models:

```go
package your_crawler_name

// Define custom data structures specific to your crawler
type YourCustomData struct {
    Field1 string
    Field2 int
    Field3 []string
}
```

### 4. Register Your Crawler in the Factory

Add a factory function in `crawlers/crawler_factory.go`:

```go
// Add to the existing file

// NewYourCrawlerNameCrawler creates a Your Crawler
func NewYourCrawlerNameCrawler(service CrawlerService) Crawler {
    config := NewCrawlerConfig(common.YourCrawlerName, "example.com")
    baseCrawler := NewBaseCrawler(config, service)

    // Set custom browser options if needed
    baseCrawler.BrowserOpts.WaitAfterLoad = 3 * time.Second

    return &your_crawler_name.YourCrawler{
        BaseCrawler: baseCrawler,
    }
}
```

### 5. Update the CreateCrawlerByType Function

Update the function in `crawlers/create_crawler.go` to include your crawler:

```go
// Add to the CreateCrawlerByType function
func CreateCrawlerByType(crawlerType common.CrawlerType, service CrawlerService) (Crawler, error) {
    switch crawlerType {
    // Existing cases
    case common.IndonesiaSupremeCourt:
        return NewIndonesiaSupremeCourtCrawler(service), nil
    case common.SingaporeSupremeCourt:
        return NewSingaporeSupremeCourtCrawler(service), nil
    case common.LKPPBlacklist:
        return NewLKPPBlacklistCrawler(service), nil

    // Your new crawler
    case common.YourCrawlerName:
        return NewYourCrawlerNameCrawler(service), nil

    default:
        return nil, fmt.Errorf("unsupported crawler type: %s", crawlerType)
    }
}
```

### 6. Register the Crawler to Listen to NATS Messages

Update the `RegisterCrawlers` function in `crawlers/nats_handlers.go`:

```go
// Add your crawler type to the list
crawlerTypes := []common.CrawlerType{
    common.IndonesiaSupremeCourt,
    common.SingaporeSupremeCourt,
    common.LKPPBlacklist,
    common.YourCrawlerName, // Add your crawler type here
}
```

## Testing Your Crawler

### 1. Manual Testing

You can manually test your crawler by sending a message to NATS:

```go
// Example code to test your crawler
msg := crawler.CrawlMessage{
    CrawlerType: string(common.YourCrawlerName),
    Action:      "crawl_keyword",
    Keyword:     "test keyword",
}

// Convert to JSON
data, err := json.Marshal(msg)
if err != nil {
    log.Error().Err(err).Msg("Failed to marshal message")
    return
}

// Publish to NATS
subject := fmt.Sprintf("crawler.%s", common.YourCrawlerName)
err = natsClient.Publish(subject, data)
if err != nil {
    log.Error().Err(err).Msg("Failed to publish message")
    return
}
```

### 2. Writing Unit Tests

Create a test file for your crawler:

```go
package your_crawler_name_test

import (
    "context"
    "testing"

    "github.com/adryanev/go-http-service-template/common"
    "github.com/adryanev/go-http-service-template/crawlers"
    "github.com/go-rod/rod"
    "github.com/go-rod/rod/lib/launcher"
    "github.com/stretchr/testify/assert"
)

func TestYourCrawler_ExtractElements(t *testing.T) {
    // Setup mock service
    mockService := &MockCrawlerService{}

    // Create the crawler
    crawler, err := crawlers.CreateCrawlerByType(common.YourCrawlerName, mockService)
    assert.NoError(t, err)

    // Set up the browser
    ctx := context.Background()
    err = crawler.Setup(ctx)
    assert.NoError(t, err)
    defer crawler.Teardown(ctx)

    // Create a test page or use a mock
    // ...

    // Test extraction logic
    // ...
}
```

## Advanced Crawler Features

### 1. Handling Authentication

If the website requires authentication:

```go
// Example authentication method
func (c *YourCrawler) authenticate(ctx context.Context, page *rod.Page) error {
    // Navigate to login page
    err := page.Navigate("https://example.com/login")
    if err != nil {
        return err
    }

    // Wait for the page to load
    err = page.WaitLoad()
    if err != nil {
        return err
    }

    // Fill in login form
    err = page.MustElement("#username").Input("your_username")
    if err != nil {
        return err
    }

    err = page.MustElement("#password").Input("your_password")
    if err != nil {
        return err
    }

    // Click login button
    err = page.MustElement("#login-button").Click()
    if err != nil {
        return err
    }

    // Wait for authentication to complete
    return page.WaitLoad()
}
```

### 2. Handling Pagination

For websites with paginated results:

```go
// Example pagination handling
func (c *YourCrawler) crawlWithPagination(ctx context.Context, baseURL string, maxPages int) error {
    var allItems []crawler.Item

    for i := 1; i <= maxPages; i++ {
        pageURL := fmt.Sprintf("%s?page=%d", baseURL, i)

        page, err := c.Navigate(ctx, pageURL)
        if err != nil {
            return err
        }

        items, err := c.ExtractElements(ctx, page)
        if err != nil {
            page.Close()
            return err
        }

        allItems = append(allItems, items...)

        // Check if there's a next page
        hasNextPage, err := page.Has(".next-page")
        page.Close()

        if err != nil || !hasNextPage {
            break
        }
    }

    // Process all items
    // ...

    return nil
}
```

### 3. Handling JavaScript-Heavy Sites

For sites with extensive JavaScript:

```go
// Configure the base crawler with special options
func NewYourJSHeavyCrawler(service CrawlerService) Crawler {
    config := NewCrawlerConfig(common.YourCrawlerName, "js-heavy-site.com")
    baseCrawler := NewBaseCrawler(config, service)

    // Increase wait time for JavaScript rendering
    baseCrawler.BrowserOpts.WaitAfterLoad = 5 * time.Second

    return &your_crawler_name.YourCrawler{
        BaseCrawler: baseCrawler,
    }
}

// In your extraction logic:
func (c *YourCrawler) ExtractElements(ctx context.Context, page *rod.Page) ([]crawler.Item, error) {
    // Wait for specific elements to appear
    err := page.WaitElementsMoreThan("your-selector", 1)
    if err != nil {
        return nil, err
    }

    // Wait for network idle
    err = page.WaitIdle(5 * time.Second)
    if err != nil {
        return nil, err
    }

    // Continue with extraction
    // ...
}
```

## Understanding Key Components

### CrawlerService

The CrawlerService provides database operations for URL frontiers:

- `UpsertUrl`: Add or update URL frontiers
- `UpdateFrontierStatuses`: Update URL frontier statuses
- `GetUrlFrontierByUrl`: Get a URL frontier by URL
- `GetUrlFrontierById`: Get a URL frontier by ID
- `GetUnscrappedUrlFrontiers`: Get URL frontiers that haven't been crawled

### NATS Message Structure

Messages sent to crawlers should follow this structure:

```json
{
  "crawler_type": "your-crawler-name",
  "action": "crawl_keyword",
  "keyword": "search term",
  "url": "https://example.com/specific-page"
}
```

Supported actions:
- `crawl_all`: Crawl all URLs in the frontier
- `crawl_url`: Crawl a specific URL
- `crawl_keyword`: Crawl URLs related to a keyword
- `consume`: Process a raw message

## Best Practices

1. **Error Handling**: Properly handle and log errors
2. **Respect Robots.txt**: Check and follow robots.txt rules
3. **Rate Limiting**: Implement delays between requests
4. **Idempotency**: Ensure crawling operations can be safely retried
5. **Logging**: Include detailed logging for debugging
6. **Resource Management**: Close browser pages and connections
7. **Validation**: Validate extracted data before storing

## Conclusion

The crawler system in this project provides a flexible foundation for developing custom web crawlers. By following this guide, you can create new crawlers that integrate with the existing system for various data sources.

For additional help, check the implementations of existing crawlers in the `crawlers/` directory.
