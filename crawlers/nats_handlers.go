package crawler

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/adryanev/go-http-service-template/common"
	"github.com/adryanev/go-http-service-template/common/db"
	"github.com/adryanev/go-http-service-template/common/messaging"
	"github.com/adryanev/go-http-service-template/repository"
	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog/log"
)

// CrawlMessage represents the structure of a NATS message for crawling
type CrawlMessage struct {
	CrawlerType  string `json:"crawler_type"`
	Action       string `json:"action"`         // "crawl_all", "crawl_url", "crawl_keyword"
	URL          string `json:"url"`            // Used for crawl_url
	Keyword      string `json:"keyword"`        // Used for crawl_keyword
	DataSourceID string `json:"data_source_id"` // Data source identifier
}

// RegisterCrawlers sets up all available crawlers to listen to NATS messages
func RegisterCrawlers(natsClient *messaging.NatsClient, dbConn *db.DB) error {
	if natsClient == nil {
		return fmt.Errorf("nats client is nil")
	}

	// Initialize the crawler service
	crawlerService := NewCrawlerService(dbConn)

	// Set up NATS subscriptions for each crawler type
	crawlerTypes := []common.CrawlerType{
		common.IndonesiaSupremeCourt,
		common.SingaporeSupremeCourt,
		common.LKPPBlacklist,
		// Add any other crawler types here
	}

	// Register each crawler type
	for _, crawlerType := range crawlerTypes {
		subject := fmt.Sprintf("crawler.%s", crawlerType)
		log.Info().Str("subject", subject).Msg("Registering crawler to listen on NATS")

		// Create a handler specific to this crawler type
		handler := createCrawlerHandler(crawlerType, crawlerService)

		// Subscribe to the specific subject for this crawler
		_, err := messaging.SubscribeToSubject(natsClient, subject, handler)
		if err != nil {
			return fmt.Errorf("failed to subscribe crawler %s: %w", crawlerType, err)
		}
	}

	// Also subscribe to a general crawler subject for dynamic crawler selection
	_, err := messaging.SubscribeToSubject(natsClient, "crawler.*", handleGenericCrawlerMessage(crawlerService))
	if err != nil {
		return fmt.Errorf("failed to subscribe to generic crawler subject: %w", err)
	}

	return nil
}

// createCrawlerHandler creates a message handler for a specific crawler type
func createCrawlerHandler(crawlerType common.CrawlerType, service CrawlerService) messaging.MessageHandler {
	return func(msg *nats.Msg) error {
		log.Info().
			Str("subject", msg.Subject).
			Str("crawler", string(crawlerType)).
			Msg("Received crawler message")

		// Parse the message
		var crawlMsg CrawlMessage
		if err := json.Unmarshal(msg.Data, &crawlMsg); err != nil {
			return fmt.Errorf("failed to unmarshal message: %w", err)
		}

		// Create a data source
		dataSource := repository.DataSource{
			ID: crawlMsg.DataSourceID,
		}
		if dataSource.ID == "" {
			// Use default data source ID based on crawler type
			dataSource.ID = string(crawlerType)
		}

		// Create crawler instance
		crawler, err := GetCrawlerByType(crawlerType, service, dataSource)
		if err != nil {
			return fmt.Errorf("failed to create crawler: %w", err)
		}

		// Create a context for the crawler operations
		ctx := context.Background()

		// Setup crawler
		if err := crawler.Setup(ctx); err != nil {
			return fmt.Errorf("failed to setup crawler: %w", err)
		}
		defer crawler.Teardown(ctx)

		// Execute the appropriate crawling action
		switch crawlMsg.Action {
		case "crawl_all":
			return crawler.CrawlAll(ctx)
		case "crawl_url":
			return crawler.CrawlByURL(ctx, crawlMsg.URL)
		case "crawl_keyword":
			return crawler.CrawlByKeyword(ctx, crawlMsg.Keyword)
		case "consume":
			return crawler.Consume(ctx, msg.Data)
		default:
			return fmt.Errorf("unknown action: %s", crawlMsg.Action)
		}
	}
}

// handleGenericCrawlerMessage handles messages sent to the generic crawler.* subject
func handleGenericCrawlerMessage(service CrawlerService) messaging.MessageHandler {
	return func(msg *nats.Msg) error {
		log.Info().
			Str("subject", msg.Subject).
			Msg("Received generic crawler message")

		// Parse the message
		var crawlMsg CrawlMessage
		if err := json.Unmarshal(msg.Data, &crawlMsg); err != nil {
			return fmt.Errorf("failed to unmarshal message: %w", err)
		}

		// Determine crawler type from the message
		crawlerType := common.CrawlerType(crawlMsg.CrawlerType)
		if crawlerType == "" {
			// Try to extract from subject if not in message
			parts := strings.Split(msg.Subject, ".")
			if len(parts) >= 2 {
				crawlerType = common.CrawlerType(parts[1])
			}
		}

		if crawlerType == "" {
			return fmt.Errorf("crawler type not specified")
		}

		// Create a data source
		dataSource := repository.DataSource{
			ID: crawlMsg.DataSourceID,
		}
		if dataSource.ID == "" {
			// Use default data source ID based on crawler type
			dataSource.ID = string(crawlerType)
		}

		// Create crawler instance
		crawler, err := GetCrawlerByType(crawlerType, service, dataSource)
		if err != nil {
			return fmt.Errorf("failed to create crawler: %w", err)
		}

		// Create a context for the crawler operations
		ctx := context.Background()

		// Setup crawler
		if err := crawler.Setup(ctx); err != nil {
			return fmt.Errorf("failed to setup crawler: %w", err)
		}
		defer crawler.Teardown(ctx)

		// Execute the appropriate crawling action
		switch crawlMsg.Action {
		case "crawl_all":
			return crawler.CrawlAll(ctx)
		case "crawl_url":
			return crawler.CrawlByURL(ctx, crawlMsg.URL)
		case "crawl_keyword":
			return crawler.CrawlByKeyword(ctx, crawlMsg.Keyword)
		case "consume":
			return crawler.Consume(ctx, msg.Data)
		default:
			return fmt.Errorf("unknown action: %s", crawlMsg.Action)
		}
	}
}
