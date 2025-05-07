# Web Crawler & Scraper System

This directory contains the implementation of the modular and extensible web crawler and scraper system.

## System Overview

The system consists of two main components:

1. **Crawler**: Responsible for discovering and saving URLs to be scraped
2. **Scraper**: Responsible for extracting data from the discovered URLs

## Architecture

The system is designed to be modular and extensible, with the following components:

- **Interfaces**: Core interfaces that define the behavior of crawlers and scrapers
- **Base Implementations**: Common functionality shared by all crawlers and scrapers
- **Site-specific Implementations**: Crawlers and scrapers for specific sites
- **Factories**: Factories for creating crawlers and scrapers
- **Repositories**: Interfaces for database access
- **Storage Service**: Interface for cloud storage
- **Message Broker**: Interface for asynchronous messaging

## Adding a New Site

To add support for a new site, follow these steps:

1. Create a new directory under `crawlers/` for your site
2. Define a site-specific configuration struct that implements the `DataSourceConfig` interface
3. Create a crawler struct that embeds `BaseCrawler` and implements the `Crawler` interface
4. Create a scraper struct that embeds `BaseScraper` and implements the `Scraper` interface
5. Update the factory to create instances of your crawler and scraper

### Example: Adding a new site "example-court"

1. Create the directory structure:

```
crawlers/
└── example-court/
    ├── crawler.go
    └── scraper.go
```

2. Define the site-specific configuration:

```go
// In common/crawler/config.go
type ExampleCourtConfig struct {
    BaseConfig
    APIKey string `json:"api_key"`
}

func (c ExampleCourtConfig) Validate() error {
    if c.APIKey == "" {
        return errors.New("API key is required")
    }
    return nil
}

// Update LoadDataSourceConfig function
func LoadDataSourceConfig(raw json.RawMessage, configType string) (DataSourceConfig, error) {
    switch configType {
    // Existing cases...
    case "examplecourt":
        var c ExampleCourtConfig
        return c, json.Unmarshal(raw, &c)
    default:
        return nil, fmt.Errorf("unknown config type: %s", configType)
    }
}
```

3. Create the crawler implementation:

```go
// In crawlers/example-court/crawler.go
package example_court

import (
    "context"

    "github.com/go-rod/rod"
    "github.com/LexiconIndonesia/crawler-http-service/common/crawler"
)

type ExampleCourtCrawler struct {
    crawler.BaseCrawler
    Config crawler.ExampleCourtConfig
    Utils  *crawler.RodUtils
}

func NewExampleCourtCrawler(config crawler.ExampleCourtConfig, baseConfig crawler.BaseCrawlerConfig, broker crawler.MessageBroker) (*ExampleCourtCrawler, error) {
    return &ExampleCourtCrawler{
        BaseCrawler: crawler.BaseCrawler{
            Config:        baseConfig,
            MessageBroker: broker,
        },
        Config: config,
    }, nil
}

// Implement the Crawler interface methods...
func (c *ExampleCourtCrawler) CrawlAll(ctx context.Context) error {
    // Not implemented as per requirements
    return crawler.ErrNotImplemented
}

// Implement other required methods...
```

4. Create the scraper implementation:

```go
// In crawlers/example-court/scraper.go
package example_court

import (
    "context"

    "github.com/go-rod/rod"
    "github.com/LexiconIndonesia/crawler-http-service/common/crawler"
)

type ExampleCourtScraper struct {
    crawler.BaseScraper
    Config crawler.ExampleCourtConfig
    Utils  *crawler.RodUtils
}

func NewExampleCourtScraper(config crawler.ExampleCourtConfig, baseConfig crawler.BaseScraperConfig, broker crawler.MessageBroker) (*ExampleCourtScraper, error) {
    return &ExampleCourtScraper{
        BaseScraper: crawler.BaseScraper{
            Config:        baseConfig,
            MessageBroker: broker,
        },
        Config: config,
    }, nil
}

// Implement the Scraper interface methods...
func (s *ExampleCourtScraper) ScrapeAll(ctx context.Context) error {
    // Not implemented as per requirements
    return crawler.ErrNotImplemented
}

// Implement other required methods...
```

5. Update the factory:

```go
// In common/crawler/factory.go
func (f *CrawlerFactory) CreateCrawler(ctx context.Context, dataSourceType string, configType string, configData json.RawMessage) (Crawler, error) {
    config, err := LoadDataSourceConfig(configData, configType)
    if err != nil {
        return nil, err
    }

    baseConfig := DefaultBaseCrawlerConfig()
    baseConfig.DataSourceID = dataSourceType

    switch configType {
    // Existing cases...
    case "examplecourt":
        exampleConfig, ok := config.(ExampleCourtConfig)
        if !ok {
            return nil, fmt.Errorf("invalid config type for examplecourt")
        }
        return example_court.NewExampleCourtCrawler(exampleConfig, baseConfig, f.MessageBroker)
    default:
        return nil, fmt.Errorf("unknown config type: %s", configType)
    }
}

// Do the same for ScraperFactory.CreateScraper
```

## Using NATS for Communication

The crawler and scraper communicate asynchronously using NATS Jetstream. This allows for:

- Decoupling of crawling and scraping
- Horizontal scaling of crawlers and scrapers
- Improved fault tolerance

### Message Types

- **crawler.url.discovered**: Published when a crawler discovers a URL
- **crawler.url.crawled**: Published when a crawler finishes crawling a URL
- **scraper.url.pending**: Published to request scraping of a URL
- **scraper.url.scraped**: Published when a scraper finishes scraping a URL

## Database Schema

The system uses the following tables:

- **data_sources**: Defines the different data sources
- **url_frontiers**: Stores URLs to be crawled/scraped
- **extractions**: Stores extracted data
- **extraction_versions**: Stores historical versions of extractions
- **crawler_logs**: Stores logs of crawler/scraper activities

## Error Handling and Retries

The system includes built-in error handling and retry mechanisms:

- Each URL has an `attempts` counter
- Failed URLs can be retried up to a configurable number of times
- Errors are logged with context information
- URLs that consistently fail are marked as failed
