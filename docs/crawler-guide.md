# Crawler & Scraper Development Guide

## Overview

This guide provides comprehensive instructions for developing new crawlers and scrapers for the Lexicon Indonesia crawler system. The system is designed to be modular and extensible, allowing for easy addition of new data sources.

## Table of Contents

1. [Architecture](#architecture)
2. [Prerequisites](#prerequisites)
3. [Step-by-Step Implementation Guide](#step-by-step-implementation-guide)
4. [Configuration Management](#configuration-management)
5. [Testing Your Implementation](#testing-your-implementation)
6. [Debugging Tips](#debugging-tips)
7. [Deployment](#deployment)
8. [Best Practices](#best-practices)

## Architecture

The crawler system consists of two main components:

### Crawler
Responsible for:
- Visiting source websites
- Extracting URLs for detail pages
- Saving these URLs to the `url_frontiers` table
- Supporting different input types: `all`, `by_url`, and `by_keyword`

### Scraper
Responsible for:
- Processing entries in the `url_frontiers` table
- Extracting structured data from detail pages
- Downloading and storing artifacts (PDFs, HTML, etc.)
- Saving metadata to the `extractions` table

### Key Components

- **Interfaces**: `Crawler` and `Scraper` define the contract for implementations
- **Base Implementations**: `BaseCrawler` and `BaseScraper` provide common functionality
- **Site-specific Implementations**: Concrete implementations for each source
- **Factories**: `CrawlerFactory` and `ScraperFactory` for creating the appropriate implementations
- **Repositories**: Database access layer for URL frontiers and extractions
- **Storage**: GCS integration for artifact storage
- **Messaging**: NATS for asynchronous communication between components

## Prerequisites

Before you begin, ensure you have:

1. Go 1.18+ installed
2. Docker and Docker Compose for local development
3. Access to the project repository
4. Basic understanding of web crawling concepts
5. Familiarity with [Rod](https://github.com/go-rod/rod) for browser automation

## Step-by-Step Implementation Guide

### 1. Create the Directory Structure

Create a new directory under `crawlers/` for your site:

```
crawlers/
└── your-new-source/
    ├── crawler.go
    └── scraper.go
```

### 2. Define Configuration

Add your site-specific configuration to `common/crawler/config.go`:

```go
// YourNewSourceConfig represents configuration for your source
type YourNewSourceConfig struct {
    BaseConfig
    // Add your source-specific configuration fields
    APIKey string `json:"api_key"`
}

// Validate validates the YourNewSourceConfig
func (c YourNewSourceConfig) Validate() error {
    if c.DetailLinkSelector == "" {
        return errors.New("missing detail link selector")
    }
    // Add other validation logic
    return nil
}

// Update LoadDataSourceConfig function to include your new config type
func LoadDataSourceConfig(raw json.RawMessage, configType string) (DataSourceConfig, error) {
    switch configType {
    // Existing cases...
    case "yournewsource":
        var c YourNewSourceConfig
        if err := json.Unmarshal(raw, &c); err != nil {
            return nil, fmt.Errorf("failed to unmarshal your new source config: %w", err)
        }
        return c, nil
    default:
        return nil, fmt.Errorf("unknown config type: %s", configType)
    }
}
```

### 3. Implement the Crawler

Create the crawler implementation in `crawlers/your-new-source/crawler.go`:

```go
package your_new_source

import (
    "context"

    "github.com/LexiconIndonesia/crawler-http-service/common/crawler"
    "github.com/LexiconIndonesia/crawler-http-service/repository"
    "github.com/go-rod/rod"
    "github.com/rs/zerolog/log"
)

// YourNewSourceCrawler is a crawler for your new source
type YourNewSourceCrawler struct {
    crawler.BaseCrawler
    Config crawler.YourNewSourceConfig
}

// NewYourNewSourceCrawler creates a new YourNewSourceCrawler
func NewYourNewSourceCrawler(config crawler.YourNewSourceConfig, baseConfig crawler.BaseCrawlerConfig, broker crawler.MessageBroker) (*YourNewSourceCrawler, error) {
    return &YourNewSourceCrawler{
        BaseCrawler: crawler.BaseCrawler{
            Config:        baseConfig,
            MessageBroker: broker,
        },
        Config: config,
    }, nil
}

// CrawlAll crawls all pages from your new source
func (c *YourNewSourceCrawler) CrawlAll(ctx context.Context) error {
    // Not implemented as per requirements
    log.Error().Msg("CrawlAll method not implemented")
    return crawler.ErrNotImplemented
}

// CrawlByKeyword crawls pages based on a search term
func (c *YourNewSourceCrawler) CrawlByKeyword(ctx context.Context, keyword string) error {
    // Not implemented as per requirements
    log.Error().Msg("CrawlByKeyword method not implemented")
    return crawler.ErrNotImplemented
}

// CrawlByURL crawls a specific URL
func (c *YourNewSourceCrawler) CrawlByURL(ctx context.Context, url string) error {
    // Not implemented as per requirements
    log.Error().Msg("CrawlByURL method not implemented")
    return crawler.ErrNotImplemented
}

// ExtractElements extracts URL frontiers from a page
func (c *YourNewSourceCrawler) ExtractElements(ctx context.Context, page *rod.Page) ([]repository.UrlFrontier, error) {
    // Not implemented as per requirements
    log.Error().Msg("ExtractElements method not implemented")
    return nil, crawler.ErrNotImplemented
}
```

### 4. Implement the Scraper

Create the scraper implementation in `crawlers/your-new-source/scraper.go`:

```go
package your_new_source

import (
    "context"

    "github.com/LexiconIndonesia/crawler-http-service/common/crawler"
    "github.com/LexiconIndonesia/crawler-http-service/common/models"
    "github.com/LexiconIndonesia/crawler-http-service/repository"
    "github.com/go-rod/rod"
    "github.com/rs/zerolog/log"
)

// YourNewSourceScraper is a scraper for your new source
type YourNewSourceScraper struct {
    crawler.BaseScraper
    Config crawler.YourNewSourceConfig
}

// NewYourNewSourceScraper creates a new YourNewSourceScraper
func NewYourNewSourceScraper(config crawler.YourNewSourceConfig, baseConfig crawler.BaseScraperConfig, broker crawler.MessageBroker) (*YourNewSourceScraper, error) {
    return &YourNewSourceScraper{
        BaseScraper: crawler.BaseScraper{
            Config:        baseConfig,
            MessageBroker: broker,
        },
        Config: config,
    }, nil
}

// ScrapeAll scrapes all pending URLs for your new source
func (s *YourNewSourceScraper) ScrapeAll(ctx context.Context) error {
    // Not implemented as per requirements
    log.Error().Msg("ScrapeAll method not implemented")
    return crawler.ErrNotImplemented
}

// ScrapeByUrlFrontierID scrapes a specific URL frontier by ID
func (s *YourNewSourceScraper) ScrapeByUrlFrontierID(ctx context.Context, id string) error {
    // Not implemented as per requirements
    log.Error().Msg("ScrapeByUrlFrontierID method not implemented")
    return crawler.ErrNotImplemented
}

// ExtractElements extracts data from a page
func (s *YourNewSourceScraper) ExtractElements(ctx context.Context, page *rod.Page) ([]repository.Extraction, error) {
    // Not implemented as per requirements
    log.Error().Msg("ExtractElements method not implemented")
    return nil, crawler.ErrNotImplemented
}

// ExtractArtifactsFromPage extracts and downloads artifacts from a page
func (s *YourNewSourceScraper) ExtractArtifactsFromPage(ctx context.Context, page *rod.Page) ([]models.ExtractionArtifact, error) {
    // Not implemented as per requirements
    log.Error().Msg("ExtractArtifactsFromPage method not implemented")
    return nil, crawler.ErrNotImplemented
}
```

### 5. Update Factories

Add your new source to the factory implementations in `common/crawler/factory.go`:

```go
// In CrawlerFactory.CreateCrawler method
func (f *CrawlerFactory) CreateCrawler(ctx context.Context, dataSourceType string, configType string, configData json.RawMessage) (Crawler, error) {
    config, err := LoadDataSourceConfig(configData, configType)
    if err != nil {
        return nil, err
    }

    baseConfig := DefaultBaseCrawlerConfig()
    baseConfig.DataSourceID = dataSourceType

    switch configType {
    // Existing cases...
    case "yournewsource":
        yourConfig, ok := config.(YourNewSourceConfig)
        if !ok {
            return nil, fmt.Errorf("invalid config type for your new source")
        }
        return your_new_source.NewYourNewSourceCrawler(yourConfig, baseConfig, f.MessageBroker)
    default:
        return nil, fmt.Errorf("CreateCrawler method not implemented for type: %s", dataSourceType)
    }
}

// Similarly for ScraperFactory.CreateScraper method
```

## Configuration Management

### Data Source Configuration

Data sources are configured in the database's `data_sources` table with the following fields:

- `id`: Unique identifier
- `name`: Human-readable name
- `country`: Country code
- `source_type`: Type of source (e.g., "court", "regulatory", etc.)
- `base_url`: Base URL of the source
- `description`: Description of the source
- `config`: JSON configuration specific to the source (polymorphic)
- `is_active`: Whether the source is active

Example configuration for your new source:

```json
{
  "pagination_selector": ".pagination a",
  "detail_link_selector": ".case-item a.detail-link",
  "max_pages": 10,
  "delay_ms": 1000,
  "api_key": "your-api-key-if-needed"
}
```

## Testing Your Implementation

### Local Testing

1. Start the development environment:

```bash
docker-compose up -d
```

2. Run the application:

```bash
go run main.go
```

3. Test your crawler with a specific URL:

```bash
curl -X POST http://localhost:8080/api/v1/crawlers/{your-source-id}/crawl-url -d '{"url": "https://example.com/search"}'
```

### Testing APIs

Use the following API endpoints for testing:

#### Crawler Endpoints

- `POST /api/v1/crawlers/{source-id}/crawl-all`: Crawl all pages
- `POST /api/v1/crawlers/{source-id}/crawl-keyword`: Crawl based on keyword
- `POST /api/v1/crawlers/{source-id}/crawl-url`: Crawl specific URL

#### Scraper Endpoints

- `POST /api/v1/scrapers/{source-id}/scrape-all`: Scrape all pending URLs
- `POST /api/v1/scrapers/{source-id}/scrape/{frontier-id}`: Scrape specific URL frontier

## Debugging Tips

### Browser Debugging

Rod allows browser debugging. Add this to your crawler/scraper before navigation:

```go
// Enable browser debugging
page.MustSetViewport(1280, 800, 1, false).
    MustActivate().
    MustWaitLoad()
```

### Logging

Use the provided logging framework for consistent logging:

```go
log.Info().Str("url", url).Msg("Starting crawl")
log.Error().Err(err).Str("url", url).Msg("Failed to crawl")
```

### Common Issues

1. **Selector Issues**: Verify selectors using browser developer tools
2. **Timeouts**: Adjust timeout settings in the config
3. **Rate Limiting**: Implement delays between requests
4. **JavaScript Rendering**: Ensure JS execution is complete before extraction

## Deployment

The crawler system is deployed as a containerized service. Your new crawler will be included in the deployment once it's merged into the main branch.

## Best Practices

### General Best Practices

1. **Error Handling**: Implement robust error handling with contextual information
2. **Rate Limiting**: Respect the target site's rate limits
3. **Concurrency**: Use reasonable concurrency limits
4. **Idempotency**: Ensure crawling and scraping operations are idempotent
5. **Resource Management**: Properly close resources (browser pages, DB connections)

### Rod Best Practices

1. **Page Management**: Always close pages after use
2. **Selectors**: Use specific, stable CSS selectors
3. **Waiting**: Always wait for navigation and DOM elements to be ready
4. **Resource Usage**: Monitor CPU and memory usage

### Example: Using Rod Effectively

```go
// Wait for selectors to be ready
elements, err := page.Timeout(time.Second * 10).Element(c.Config.DetailLinkSelector)
if err != nil {
    return nil, fmt.Errorf("failed to find detail link elements: %w", err)
}

// Extract href attributes
var urls []string
for _, el := range elements.MustElements(c.Config.DetailLinkSelector) {
    href := el.MustAttribute("href")
    if href != nil {
        urls = append(urls, *href)
    }
}
```

## Conclusion

By following this guide, you should be able to implement a new crawler and scraper for any data source. The modular architecture allows for easy extension and maintenance. If you encounter any issues, refer to the debugging tips or reach out to the development team.

Remember, as per the requirements, actual crawling or scraping logic should not be implemented - just create the stubs with "not implemented" errors.
