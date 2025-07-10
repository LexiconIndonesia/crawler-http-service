package logger

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"time"

	"github.com/LexiconIndonesia/crawler-http-service/common/db"
	"github.com/LexiconIndonesia/crawler-http-service/repository"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// LogEvent represents a log event
type LogEvent struct {
	DataSourceID string
	JobID        string
	EventType    string
	Message      string
	Details      interface{}
}

// DBLogWriter is an io.Writer that writes logs to the database.
type DBLogWriter struct {
	db *db.DB
}

// NewDBLogWriter creates a new DBLogWriter.
func NewDBLogWriter(db *db.DB) *DBLogWriter {
	return &DBLogWriter{db: db}
}

// Write implements io.Writer. It's called by zerolog for each log event.
func (w *DBLogWriter) Write(p []byte) (n int, err error) {
	// Only process if the log event is a JSON object
	trimmed := bytes.TrimSpace(p)
	if len(trimmed) == 0 || trimmed[0] != '{' || trimmed[len(trimmed)-1] != '}' {
		// Not a JSON object, skip
		return len(p), nil
	}

	// Try to find valid JSON objects in the input
	decoder := json.NewDecoder(bytes.NewReader(trimmed))
	for decoder.More() {
		var event map[string]interface{}
		if err := decoder.Decode(&event); err != nil {
			// If we can't decode the JSON, it might be malformed or incomplete
			// Skip this without logging an error since it's common for log streams
			// to have mixed content
			break
		}

		// Process the successfully decoded JSON event
		w.processLogEvent(event)
	}

	return len(p), nil
}

// processLogEvent processes a single log event
func (w *DBLogWriter) processLogEvent(event map[string]interface{}) {
	// Extract fields for LogEvent
	logEvent := LogEvent{}
	if msg, ok := event["message"].(string); ok {
		logEvent.Message = msg
	}
	if level, ok := event["level"].(string); ok {
		logEvent.EventType = level
	}
	if dsID, ok := event["dataSourceID"].(string); ok {
		logEvent.DataSourceID = dsID
	}
	if jID, ok := event["jobID"].(string); ok {
		logEvent.JobID = jID
	}

	// Only log info level and above to db
	level, hasLevel := event["level"].(string)
	if !hasLevel || (level != zerolog.LevelInfoValue && level != zerolog.LevelWarnValue && level != zerolog.LevelErrorValue && level != zerolog.LevelFatalValue && level != zerolog.LevelPanicValue) {
		return
	}

	// Use the rest of the fields as details, after removing the standard ones.
	eventCopy := make(map[string]interface{})
	for k, v := range event {
		eventCopy[k] = v
	}
	delete(eventCopy, "time")
	delete(eventCopy, "level")
	delete(eventCopy, "message")
	delete(eventCopy, "dataSourceID")
	delete(eventCopy, "jobID")
	logEvent.Details = eventCopy

	_ = logToDatabase(w.db, logEvent)
}

// logToDatabase stores the log in the database with improved error handling
func logToDatabase(dbConn *db.DB, event LogEvent) error {
	// Skip logging if data source ID is empty to avoid foreign key constraint violations
	if event.DataSourceID == "" {
		return nil
	}

	// Create a context with a shorter timeout to avoid blocking
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Generate a unique ID for the log
	logID := uuid.New().String()

	// Convert details to JSON
	var detailsJSON json.RawMessage
	if event.Details != nil {
		detailsJSON, _ = json.Marshal(event.Details)
	} else {
		detailsJSON = json.RawMessage("{}")
	}

	var jobID pgtype.Text
	if event.JobID != "" {
		jobID = pgtype.Text{String: event.JobID, Valid: true}
	} else {
		jobID = pgtype.Text{Valid: false}
	}

	// Create message if provided
	message := pgtype.Text{String: event.Message, Valid: event.Message != ""}

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

	_, err := dbConn.Queries.CreateCrawlerLog(ctx, logParams)
	return err
}

// InitializeLogging sets up global zerolog configuration with database hooks
func InitializeLogging(db *db.DB) {
	dbWriter := NewDBLogWriter(db)
	consoleWriter := zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339}

	multiWriter := zerolog.MultiLevelWriter(consoleWriter, dbWriter)

	log.Logger = zerolog.New(multiWriter).With().Timestamp().Logger()
}
