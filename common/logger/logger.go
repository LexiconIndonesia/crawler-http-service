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

	// Since zerolog.Event doesn't expose its fields directly, we'll use
	// the base LogEvent structure and enhance it as needed
	logEvent := LogEvent{
		Message:   msg,
		EventType: level.String(),
	}

	// This is done asynchronously to not block the logging
	go func() {
		if err := h.logToDatabase(logEvent); err != nil {
			// Log the error but don't use the hook to avoid potential infinite recursion
			// Use a simple log to avoid recursion
			log.Error().Err(err).Msg("Failed to log to database via hook")
		}
	}()
}

// logToDatabase stores the log in the database with improved error handling
func (h *CrawlerLogHook) logToDatabase(event LogEvent) error {
	// Skip logging if both data source ID and URL frontier ID are empty to avoid foreign key constraint violations
	if event.DataSourceID == "" {
		return nil
	}

	// Create a context with a shorter timeout to avoid blocking
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Check if database is available first
	if err := h.db.Ping(ctx); err != nil {
		// Database is not available, log to console only
		log.Warn().Err(err).Msg("Database unavailable, skipping log storage")
		return nil // Don't return error to avoid infinite recursion
	}

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

	var jobID pgtype.Text
	if event.JobID != "" {
		jobID = pgtype.Text{
			String: event.JobID,
			Valid:  true,
		}
	} else {
		jobID = pgtype.Text{Valid: false}
	}

	// Create message if provided
	message := pgtype.Text{
		String: event.Message,
		Valid:  event.Message != "",
	}

	// Insert into database
	logParams := repository.CreateCrawlerLogParams{
		ID:           logID,
		DataSourceID: event.DataSourceID,
		JobID:        jobID,
		EventType:    event.EventType,
		Message:      message,
		Details:      detailsJSON,
		CreatedAt:    time.Now(),
	}

	_, err := h.db.Queries.CreateCrawlerLog(ctx, logParams)
	if err != nil {
		// Check if it's a context timeout or connection issue
		if ctx.Err() == context.DeadlineExceeded {
			log.Warn().Msg("Database log write timed out, continuing without storage")
			return nil // Don't return error to avoid infinite recursion
		}
		return err
	}

	return nil
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

		if match := extractField(msg, "jobID"); match != "" {
			logEvent.JobID = match
		}

		// Extract any other details that might be in JSON format within the message
		details := extractJSONDetails(msg)
		if details != nil {
			logEvent.Details = details
		}

		if err := h.logToDatabase(logEvent); err != nil {
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
	DataSourceID string
	JobID        string
	EventType    string
	Message      string
	Details      interface{}
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
	// Skip logging if both data source ID and URL frontier ID are empty to avoid foreign key constraint violations
	if event.DataSourceID == "" {
		return nil
	}

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

	var jobID pgtype.Text
	if event.JobID != "" {
		jobID = pgtype.Text{
			String: event.JobID,
			Valid:  true,
		}
	} else {
		jobID = pgtype.Text{Valid: false}
	}

	// Create message if provided
	message := pgtype.Text{
		String: event.Message,
		Valid:  event.Message != "",
	}

	// Insert into database with retry mechanism
	logParams := repository.CreateCrawlerLogParams{
		ID:           logID,
		DataSourceID: event.DataSourceID,
		JobID:        jobID,
		EventType:    event.EventType,
		Message:      message,
		Details:      detailsJSON,
		CreatedAt:    time.Now(),
	}

	// Try to insert with retries
	var err error
	for attempt := 1; attempt <= 3; attempt++ {
		// Create a context with timeout for each attempt
		insertCtx, cancel := context.WithTimeout(ctx, 3*time.Second)

		_, err = s.db.Queries.CreateCrawlerLog(insertCtx, logParams)
		cancel()

		if err == nil {
			break // Success, exit retry loop
		}

		// Check if it's a context timeout or connection issue
		if insertCtx.Err() == context.DeadlineExceeded {
			log.Warn().Int("attempt", attempt).Msg("Database log write timed out, retrying...")
			time.Sleep(time.Duration(attempt) * 100 * time.Millisecond) // Exponential backoff
			continue
		}

		// For other errors, don't retry
		break
	}

	if err != nil {
		log.Error().Err(err).Msg("Failed to insert log into database after retries")
		// Don't return error to avoid breaking the application flow
		// Just log to console and continue
	}

	// Also log to console for visibility
	logEntry := log.Info()

	if event.DataSourceID != "" {
		logEntry = logEntry.Str("dataSourceID", event.DataSourceID)
	}

	if event.JobID != "" {
		logEntry = logEntry.Str("jobID", event.JobID)
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
		DataSourceID: dataSourceID,
		EventType:    "error",
		Message:      message,
		Details:      detailMap,
	})
}

// CheckDatabaseHealth checks if the database is available for logging
func (s *LogService) CheckDatabaseHealth(ctx context.Context) error {
	// Create a short timeout for health check
	healthCtx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()

	return s.db.Ping(healthCtx)
}

// GetDatabaseStats returns basic statistics about the database connection
func (s *LogService) GetDatabaseStats() map[string]interface{} {
	stats := s.db.Pool.Stat()
	return map[string]interface{}{
		"total_connections":        stats.TotalConns(),
		"idle_connections":         stats.IdleConns(),
		"acquired_connections":     stats.AcquiredConns(),
		"constructing_connections": stats.ConstructingConns(),
	}
}
