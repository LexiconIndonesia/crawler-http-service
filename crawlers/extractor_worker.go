package crawler

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog/log"

	"github.com/adryanev/go-http-service-template/common/messaging"
	"github.com/adryanev/go-http-service-template/common/worker"
)

// ExtractionJob represents a single extraction task
type ExtractionJob struct {
	id            worker.JobID
	urlFrontierID string
	dataSourceID  string
	cancel        context.CancelFunc
	service       CrawlerService
}

// GetID returns the job's unique identifier
func (j *ExtractionJob) GetID() worker.JobID {
	return j.id
}

// Execute runs the extraction job
func (j *ExtractionJob) Execute(ctx context.Context) error {
	// Create a cancellable context for this job
	jobCtx, cancel := context.WithCancel(ctx)
	j.cancel = cancel

	defer func() {
		if j.cancel != nil {
			j.cancel()
		}
	}()

	log.Info().
		Str("job_id", string(j.id)).
		Str("url_frontier_id", j.urlFrontierID).
		Str("data_source_id", j.dataSourceID).
		Msg("Starting extraction job")

	// Get the URL frontier
	frontier, err := j.service.GetUrlFrontierById(jobCtx, j.urlFrontierID)
	if err != nil {
		log.Error().Err(err).
			Str("job_id", string(j.id)).
			Str("url_frontier_id", j.urlFrontierID).
			Msg("Failed to get URL frontier")
		return err
	}

	// Get the data source
	dataSource, err := j.service.GetDataSourceById(jobCtx, j.dataSourceID)
	if err != nil {
		log.Error().Err(err).
			Str("job_id", string(j.id)).
			Str("data_source_id", j.dataSourceID).
			Msg("Failed to get data source")
		return err
	}

	// Get the raw page content
	rawPageURL := ""
	if frontier.Metadata != nil {
		metadata := make(map[string]interface{})
		if err := json.Unmarshal(frontier.Metadata, &metadata); err == nil {
			if val, ok := metadata["raw_page_link"].(string); ok {
				rawPageURL = val
			}
		}
	}

	if rawPageURL == "" {
		log.Error().
			Str("job_id", string(j.id)).
			Str("url_frontier_id", j.urlFrontierID).
			Msg("No raw page link found in frontier metadata")
		return fmt.Errorf("no raw page link found")
	}

	// Fetch the content
	content, err := j.service.FetchContent(jobCtx, rawPageURL)
	if err != nil {
		log.Error().Err(err).
			Str("job_id", string(j.id)).
			Str("url", rawPageURL).
			Msg("Failed to fetch content")
		return err
	}

	// Generate hash from content
	hash := calculatePageHash(content)

	// Parse content as string instead of using goquery
	contentStr, err := io.ReadAll(content)
	if err != nil {
		log.Error().Err(err).
			Str("job_id", string(j.id)).
			Str("url", rawPageURL).
			Msg("Failed to read content")
		return err
	}
	contentString := string(contentStr)

	// Get extraction rules from data source config
	var config struct {
		ExtractionRules map[string]string `json:"extraction_rules"`
	}

	if err := json.Unmarshal(dataSource.Config, &config); err != nil {
		log.Error().Err(err).
			Str("job_id", string(j.id)).
			Str("data_source_id", j.dataSourceID).
			Msg("Failed to parse data source config")
		return err
	}

	// Extract data based on rules (simplified without goquery)
	extractedData := make(map[string]interface{})
	for field, selector := range config.ExtractionRules {
		// Simple text extraction (this is a basic implementation)
		// In a real app, you'd use proper HTML parsing
		if strings.Contains(contentString, selector) {
			extractedData[field] = selector
		}
	}

	// Check if we already have an extraction for this URL
	existingExtraction, err := j.service.GetExtractionByUrlFrontierID(jobCtx, j.urlFrontierID)
	if err == nil {
		// We have an existing extraction, check if the hash is different
		existingHashStr := existingExtraction.PageHash.String
		if existingExtraction.PageHash.Valid && existingHashStr != hash {
			// Hash is different, create a new version
			err = j.service.CreateExtractionVersion(jobCtx, existingExtraction.ID, hash, extractedData)
			if err != nil {
				log.Error().Err(err).
					Str("job_id", string(j.id)).
					Str("extraction_id", existingExtraction.ID).
					Msg("Failed to create extraction version")
				return err
			}

			log.Info().
				Str("job_id", string(j.id)).
				Str("url_frontier_id", j.urlFrontierID).
				Str("extraction_id", existingExtraction.ID).
				Msg("Created new extraction version")
		} else {
			log.Info().
				Str("job_id", string(j.id)).
				Str("url_frontier_id", j.urlFrontierID).
				Str("extraction_id", existingExtraction.ID).
				Msg("Content unchanged, no new version created")
		}
	} else {
		// No existing extraction, create a new one
		extractionID := uuid.New().String()
		err = j.service.CreateExtraction(jobCtx, extractionID, j.urlFrontierID, hash, extractedData)
		if err != nil {
			log.Error().Err(err).
				Str("job_id", string(j.id)).
				Str("url_frontier_id", j.urlFrontierID).
				Msg("Failed to create extraction")
			return err
		}

		log.Info().
			Str("job_id", string(j.id)).
			Str("url_frontier_id", j.urlFrontierID).
			Str("extraction_id", extractionID).
			Msg("Created new extraction")
	}

	// Update URL frontier status
	err = j.service.UpdateUrlFrontierStatus(jobCtx, j.urlFrontierID, 2) // status 2 = extracted
	if err != nil {
		log.Error().Err(err).
			Str("job_id", string(j.id)).
			Str("url_frontier_id", j.urlFrontierID).
			Msg("Failed to update URL frontier status")
		return err
	}

	log.Info().
		Str("job_id", string(j.id)).
		Str("url_frontier_id", j.urlFrontierID).
		Msg("Extraction job completed successfully")

	return nil
}

// Cancel stops the extraction job
func (j *ExtractionJob) Cancel() error {
	if j.cancel != nil {
		j.cancel()
		j.cancel = nil
		log.Info().
			Str("job_id", string(j.id)).
			Msg("Extraction job cancelled")
	}
	return nil
}

// ExtractorWorker listens for extract.run messages and processes them
type ExtractorWorker struct {
	natsClient *messaging.NatsClient
	service    CrawlerService
	sub        *nats.Subscription
	activeJobs map[worker.JobID]*ExtractionJob
	mu         sync.RWMutex
	ctx        context.Context
	cancel     context.CancelFunc
}

// NewExtractorWorker creates a new extractor worker
func NewExtractorWorker(natsClient *messaging.NatsClient, service CrawlerService) *ExtractorWorker {
	return &ExtractorWorker{
		natsClient: natsClient,
		service:    service,
		activeJobs: make(map[worker.JobID]*ExtractionJob),
	}
}

// Start initializes the worker and begins listening for messages
func (w *ExtractorWorker) Start(ctx context.Context) error {
	w.ctx, w.cancel = context.WithCancel(ctx)

	// Create message handler
	handler := func(msg *nats.Msg) error {
		var extractMsg messaging.ExtractRunMessage
		if err := json.Unmarshal(msg.Data, &extractMsg); err != nil {
			log.Error().Err(err).Msg("Failed to unmarshal extract.run message")
			return err
		}

		// Generate job ID if not provided
		jobID := extractMsg.JobID
		if jobID == "" {
			jobID = uuid.New().String()
		}

		// Create and register job
		job := &ExtractionJob{
			id:            worker.JobID(jobID),
			urlFrontierID: extractMsg.URLFrontierID,
			dataSourceID:  extractMsg.DataSourceID,
			service:       w.service,
		}

		// Add to active jobs
		w.mu.Lock()
		w.activeJobs[job.GetID()] = job
		w.mu.Unlock()

		// Execute job in a goroutine
		go func() {
			defer func() {
				// Remove from active jobs when done
				w.mu.Lock()
				delete(w.activeJobs, job.GetID())
				w.mu.Unlock()
			}()

			if err := job.Execute(w.ctx); err != nil {
				log.Error().Err(err).
					Str("job_id", string(job.GetID())).
					Msg("Extraction job failed")
			}
		}()

		return nil
	}

	// Subscribe to extract.run subject
	sub, err := messaging.SubscribeToSubject(w.natsClient, messaging.SubjectExtractRun, handler)
	if err != nil {
		return fmt.Errorf("failed to subscribe to extract.run: %w", err)
	}

	w.sub = sub
	log.Info().Msg("Extractor worker started")

	return nil
}

// Stop gracefully stops the worker
func (w *ExtractorWorker) Stop(ctx context.Context) error {
	if w.cancel != nil {
		w.cancel()
	}

	// Cancel all active jobs
	w.mu.Lock()
	for _, job := range w.activeJobs {
		job.Cancel()
	}
	w.activeJobs = make(map[worker.JobID]*ExtractionJob)
	w.mu.Unlock()

	// Unsubscribe from NATS
	if w.sub != nil {
		if err := w.sub.Unsubscribe(); err != nil {
			return fmt.Errorf("failed to unsubscribe from NATS: %w", err)
		}
	}

	log.Info().Msg("Extractor worker stopped")
	return nil
}

// CancelJob cancels a specific job by ID
func (w *ExtractorWorker) CancelJob(jobID worker.JobID) error {
	w.mu.RLock()
	job, exists := w.activeJobs[jobID]
	w.mu.RUnlock()

	if !exists {
		return fmt.Errorf("job not found: %s", jobID)
	}

	return job.Cancel()
}

// calculatePageHash generates a SHA-256 hash from the page content
func calculatePageHash(content interface{}) string {
	h := sha256.New()

	switch v := content.(type) {
	case []byte:
		h.Write(v)
	case string:
		h.Write([]byte(v))
	default:
		jsonData, _ := json.Marshal(v)
		h.Write(jsonData)
	}

	return hex.EncodeToString(h.Sum(nil))
}
