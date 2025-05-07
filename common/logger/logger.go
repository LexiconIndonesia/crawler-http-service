package logger

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/LexiconIndonesia/crawler-http-service/common/db"
	"github.com/LexiconIndonesia/crawler-http-service/repository"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// CrawlerLogHook implements zerolog.Hook interface
// for storing logs in the database
type CrawlerLogHook struct {
	db *db.DB
}

// NewCrawlerLogHook creates a new log hook
func NewCrawlerLogHook(db *db.DB) *CrawlerLogHook {
	return &CrawlerLogHook{
		db: db,
	}
}

// Run implements zerolog.Hook.Run
func (h *CrawlerLogHook) Run(e *zerolog.Event, level zerolog.Level, msg string) {
	// Skip if level is too low
	if level < zerolog.InfoLevel {
		return
	}

	// Create a new context with 5 second timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Since zerolog.Event doesn't expose its fields directly, we'll use
	// the base LogEvent structure and enhance it as needed
	logEvent := LogEvent{
		Message:   msg,
		EventType: level.String(),
	}

	// This is done asynchronously to not block the logging
	go func() {
		if err := h.logToDatabase(ctx, logEvent); err != nil {
			// Log the error but don't use the hook to avoid potential infinite recursion
			log.Error().Err(err).Msg("Failed to log to database via hook")
		}
	}()
}

// logToDatabase stores the log in the database
func (h *CrawlerLogHook) logToDatabase(ctx context.Context, event LogEvent) error {
	// Generate a unique ID for the log
	logID := uuid.New().String()

	// Convert details to JSON
	var detailsJSON json.RawMessage
	if event.Details != nil {
		var err error
		detailsJSON, err = json.Marshal(event.Details)
		if err != nil {
			log.Error().Err(err).Msg("Failed to marshal log details")
			detailsJSON = json.RawMessage("{}")
		}
	} else {
		detailsJSON = json.RawMessage("{}")
	}

	// Create URL frontier ID if provided
	var urlFrontierID pgtype.Text
	if event.URLFrontierID != "" {
		urlFrontierID = pgtype.Text{
			String: event.URLFrontierID,
			Valid:  true,
		}
	} else {
		urlFrontierID = pgtype.Text{Valid: false}
	}

	// Create message if provided
	message := pgtype.Text{
		String: event.Message,
		Valid:  event.Message != "",
	}

	// Insert into database
	logParams := repository.CreateCrawlerLogParams{
		ID:            logID,
		DataSourceID:  event.DataSourceID,
		UrlFrontierID: urlFrontierID,
		EventType:     event.EventType,
		Message:       message,
		Details:       detailsJSON,
		CreatedAt:     time.Now(),
	}

	return h.db.Queries.CreateCrawlerLog(ctx, logParams)
}

// ContextualCrawlerLogHook is an extension of the basic hook that can extract context
type ContextualCrawlerLogHook struct {
	*CrawlerLogHook
}

// NewContextualCrawlerLogHook creates a new contextual log hook
func NewContextualCrawlerLogHook(db *db.DB) *ContextualCrawlerLogHook {
	return &ContextualCrawlerLogHook{
		CrawlerLogHook: NewCrawlerLogHook(db),
	}
}

// Run implements zerolog.Hook.Run with context extraction
func (h *ContextualCrawlerLogHook) Run(e *zerolog.Event, level zerolog.Level, msg string) {
	// Skip if level is too low
	if level < zerolog.InfoLevel {
		return
	}

	// Create a new context with 5 second timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// This is done asynchronously to not block the logging
	go func() {
		// Create a base log event
		logEvent := LogEvent{
			Message:   msg,
			EventType: level.String(),
		}

		// We can't directly access zerolog.Event fields, but we can prepare for specific patterns
		// Look for known fields in the message that might indicate data source or URL frontier ID

		// Try to extract dataSourceID - often logged as "dataSourceID=xyz"
		if match := extractField(msg, "dataSourceID"); match != "" {
			logEvent.DataSourceID = match
		}

		// Try to extract urlFrontierID - often logged as "urlFrontierID=xyz"
		if match := extractField(msg, "urlFrontierID"); match != "" {
			logEvent.URLFrontierID = match
		}

		// Extract any other details that might be in JSON format within the message
		details := extractJSONDetails(msg)
		if details != nil {
			logEvent.Details = details
		}

		if err := h.logToDatabase(ctx, logEvent); err != nil {
			// Log the error but don't use the hook to avoid potential infinite recursion
			log.Error().Err(err).Msg("Failed to log to database via hook")
		}
	}()
}

// Helper function to extract fields from messages
func extractField(msg, fieldName string) string {
	// Very simple extraction - in a real implementation, you'd use regex
	searchStr := fieldName + "="
	idx := strings.Index(msg, searchStr)
	if idx >= 0 {
		start := idx + len(searchStr)
		end := strings.IndexAny(msg[start:], " ,\n\t")
		if end < 0 {
			return msg[start:]
		}
		return msg[start : start+end]
	}
	return ""
}

// Helper function to extract JSON details from a message
func extractJSONDetails(msg string) interface{} {
	// Look for JSON objects in the message
	start := strings.Index(msg, "{")
	end := strings.LastIndex(msg, "}")

	if start >= 0 && end > start {
		jsonStr := msg[start : end+1]
		var result interface{}
		if err := json.Unmarshal([]byte(jsonStr), &result); err == nil {
			return result
		}
	}

	return nil
}

// LogService provides structured logging to database
type LogService struct {
	db *db.DB
}

// LogEvent represents a log event
type LogEvent struct {
	DataSourceID  string
	URLFrontierID string
	EventType     string
	Message       string
	Details       interface{}
}

// InitializeLogging sets up global zerolog configuration with database hooks
func InitializeLogging(db *db.DB) {
	// Create and add the database hook to the global logger
	hook := NewContextualCrawlerLogHook(db)
	log.Logger = log.Logger.Hook(hook)
}

// NewLogService creates a new log service
func NewLogService(db *db.DB) *LogService {
	// No need to initialize hooks here anymore as it should be done at startup
	// This allows hook to be set up once for all logging
	return &LogService{
		db: db,
	}
}

// Log creates a log entry in the database
func (s *LogService) Log(ctx context.Context, event LogEvent) error {
	// Generate a unique ID for the log
	logID := uuid.New().String()

	// Convert details to JSON
	var detailsJSON json.RawMessage
	if event.Details != nil {
		var err error
		detailsJSON, err = json.Marshal(event.Details)
		if err != nil {
			log.Error().Err(err).Msg("Failed to marshal log details")
			detailsJSON = json.RawMessage("{}")
		}
	} else {
		detailsJSON = json.RawMessage("{}")
	}

	// Create URL frontier ID if provided
	var urlFrontierID pgtype.Text
	if event.URLFrontierID != "" {
		urlFrontierID = pgtype.Text{
			String: event.URLFrontierID,
			Valid:  true,
		}
	} else {
		urlFrontierID = pgtype.Text{Valid: false}
	}

	// Create message if provided
	message := pgtype.Text{
		String: event.Message,
		Valid:  event.Message != "",
	}

	// Insert into database
	logParams := repository.CreateCrawlerLogParams{
		ID:            logID,
		DataSourceID:  event.DataSourceID,
		UrlFrontierID: urlFrontierID,
		EventType:     event.EventType,
		Message:       message,
		Details:       detailsJSON,
		CreatedAt:     time.Now(),
	}

	if err := s.db.Queries.CreateCrawlerLog(ctx, logParams); err != nil {
		log.Error().Err(err).Msg("Failed to insert log into database")
		return err
	}

	// Also log to console for visibility
	logEntry := log.Info()

	if event.DataSourceID != "" {
		logEntry = logEntry.Str("dataSourceID", event.DataSourceID)
	}

	if event.URLFrontierID != "" {
		logEntry = logEntry.Str("urlFrontierID", event.URLFrontierID)
	}

	logEntry.
		Str("eventType", event.EventType).
		Str("message", event.Message).
		Interface("details", event.Details).
		Msg("Crawler event")

	return nil
}

// Error logs an error event
func (s *LogService) Error(ctx context.Context, dataSourceID, urlFrontierID, message string, err error, details interface{}) error {
	// Create detail map that includes error
	detailMap := map[string]interface{}{
		"error": err.Error(),
	}

	// Add additional details if provided
	if details != nil {
		if detailsMap, ok := details.(map[string]interface{}); ok {
			for k, v := range detailsMap {
				detailMap[k] = v
			}
		} else {
			detailMap["additional"] = details
		}
	}

	return s.Log(ctx, LogEvent{
		DataSourceID:  dataSourceID,
		URLFrontierID: urlFrontierID,
		EventType:     "error",
		Message:       message,
		Details:       detailMap,
	})
}

// CrawlStart logs the start of a crawl operation
func (s *LogService) CrawlStart(ctx context.Context, dataSourceID, keyword string) error {
	return s.Log(ctx, LogEvent{
		DataSourceID: dataSourceID,
		EventType:    "crawl.started",
		Message:      "Crawl operation started",
		Details: map[string]interface{}{
			"keyword": keyword,
		},
	})
}

// CrawlComplete logs the completion of a crawl operation
func (s *LogService) CrawlComplete(ctx context.Context, dataSourceID string, resultsCount int) error {
	return s.Log(ctx, LogEvent{
		DataSourceID: dataSourceID,
		EventType:    "crawl.completed",
		Message:      "Crawl operation completed",
		Details: map[string]interface{}{
			"results_count": resultsCount,
		},
	})
}

// ExtractStart logs the start of an extraction operation
func (s *LogService) ExtractStart(ctx context.Context, dataSourceID, urlFrontierID, url string) error {
	return s.Log(ctx, LogEvent{
		DataSourceID:  dataSourceID,
		URLFrontierID: urlFrontierID,
		EventType:     "extract.started",
		Message:       "Extraction started",
		Details: map[string]interface{}{
			"url": url,
		},
	})
}

// ExtractComplete logs the completion of an extraction operation
func (s *LogService) ExtractComplete(ctx context.Context, dataSourceID, urlFrontierID, extractionID string, changed bool, version int) error {
	statusText := "unchanged"
	if changed {
		statusText = "changed"
	}

	return s.Log(ctx, LogEvent{
		DataSourceID:  dataSourceID,
		URLFrontierID: urlFrontierID,
		EventType:     "extract.completed",
		Message:       "Extraction completed, content " + statusText,
		Details: map[string]interface{}{
			"extraction_id": extractionID,
			"changed":       changed,
			"version":       version,
		},
	})
}

// GetLogsByDataSource retrieves logs for a specific data source with pagination
func (s *LogService) GetLogsByDataSource(ctx context.Context, dataSourceID string, limit, offset int) ([]repository.CrawlerLog, error) {
	// This would need a custom query in the repository
	// For demonstration purposes, we'll return nil and an error
	return nil, nil
}

// GetLogsByURLFrontier retrieves logs for a specific URL frontier with pagination
func (s *LogService) GetLogsByURLFrontier(ctx context.Context, urlFrontierID string, limit, offset int) ([]repository.CrawlerLog, error) {
	// This would need a custom query in the repository
	// For demonstration purposes, we'll return nil and an error
	return nil, nil
}
