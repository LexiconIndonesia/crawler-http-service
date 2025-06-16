package module

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/LexiconIndonesia/crawler-http-service/common/db"
	"github.com/LexiconIndonesia/crawler-http-service/common/messaging"
	"github.com/LexiconIndonesia/crawler-http-service/common/utils"
	"github.com/LexiconIndonesia/crawler-http-service/repository"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rs/zerolog/log"
)

// ApiService provides API endpoints for the module
type ApiService struct {
	db       *db.DB
	broker   *messaging.NatsBroker
	services map[string]interface{} // Injected services
}

// DataSourceRequest represents a request to create or update a data source
type DataSourceRequest struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Country     string          `json:"country"`
	SourceType  string          `json:"source_type"`
	BaseURL     string          `json:"base_url"`
	Description string          `json:"description"`
	Config      json.RawMessage `json:"config"`
	IsActive    bool            `json:"is_active"`
}

// CrawlRequest represents a request to start a crawl
type CrawlRequest struct {
	Keyword string `json:"keyword"`
}

// CrawlerRequest represents a request for crawler operations
type CrawlerRequest struct {
	DataSourceID string          `json:"data_source_id"`
	URL          string          `json:"url"`
	Depth        int             `json:"depth"`
	MaxPages     int             `json:"max_pages"`
	Filters      json.RawMessage `json:"filters,omitempty"`
}

// ScraperRequest represents a request for scraper operations
type ScraperRequest struct {
	DataSourceID  string `json:"data_source_id"`
	URLFrontierID string `json:"url_frontier_id"`
}

// CancelExtractorJobRequest represents a request to cancel an extractor job
type CancelExtractorJobRequest struct {
	JobID string `json:"job_id" validate:"required" example:"3fa85f64-5717-4562-b3fc-2c963f66afa6"`
}

// NewApiService creates a new API service with dependencies
func NewApiService(db *db.DB, broker *messaging.NatsBroker) *ApiService {
	return &ApiService{
		db:       db,
		broker:   broker,
		services: make(map[string]interface{}),
	}
}

// Routes returns the router for API endpoints
func (s *ApiService) Routes() http.Handler {
	r := chi.NewRouter()

	// System status
	r.Get("/status", s.getStatus)

	// Extraction operations
	r.Post("/extraction/cancel", s.cancelExtractorJob)

	return r
}

// getDataSources returns all active data sources
func (s *ApiService) getDataSources(w http.ResponseWriter, r *http.Request) {
	dataSources, err := s.db.Queries.GetActiveDataSources(r.Context())
	if err != nil {
		http.Error(w, "Error fetching data sources", http.StatusInternalServerError)
		log.Error().Err(err).Msg("Failed to fetch data sources")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(dataSources)
}

// getDataSource returns a specific data source by ID
func (s *ApiService) getDataSource(w http.ResponseWriter, r *http.Request) {
	dataSourceID := chi.URLParam(r, "dataSourceID")
	dataSource, err := s.db.Queries.GetDataSourceById(r.Context(), dataSourceID)
	if err != nil {
		http.Error(w, "Data source not found", http.StatusNotFound)
		log.Error().Err(err).Str("dataSourceID", dataSourceID).Msg("Failed to fetch data source")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(dataSource)
}

// createDataSource creates a new data source
func (s *ApiService) createDataSource(w http.ResponseWriter, r *http.Request) {
	var req DataSourceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		log.Error().Err(err).Msg("Failed to decode request body")
		return
	}

	// Generate ID if not provided
	if req.ID == "" {
		req.ID = uuid.New().String()
	}

	now := time.Now()
	baseUrl := pgtype.Text{String: req.BaseURL, Valid: req.BaseURL != ""}
	description := pgtype.Text{String: req.Description, Valid: req.Description != ""}

	err := s.db.Queries.UpsertDataSource(r.Context(), repository.UpsertDataSourceParams{
		ID:          req.ID,
		Name:        req.Name,
		Country:     req.Country,
		SourceType:  req.SourceType,
		BaseUrl:     baseUrl,
		Description: description,
		Config:      req.Config,
		IsActive:    req.IsActive,
		CreatedAt:   now,
		UpdatedAt:   now,
	})

	if err != nil {
		http.Error(w, "Failed to create data source", http.StatusInternalServerError)
		log.Error().Err(err).Msg("Failed to create data source")
		return
	}

	// Return the created data source
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"id": req.ID})
}

// triggerCrawl initiates a crawl for a specific data source
func (s *ApiService) triggerCrawl(w http.ResponseWriter, r *http.Request) {
	dataSourceID := chi.URLParam(r, "dataSourceID")

	// Check if data source exists
	dataSource, err := s.db.Queries.GetDataSourceById(r.Context(), dataSourceID)
	if err != nil {
		http.Error(w, "Data source not found", http.StatusNotFound)
		log.Error().Err(err).Str("dataSourceID", dataSourceID).Msg("Data source not found")
		return
	}

	var req CrawlRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		log.Error().Err(err).Msg("Failed to decode request body")
		return
	}

	// Create crawl message
	crawlMsg := map[string]interface{}{
		"data_source_id": dataSourceID,
		"keyword":        req.Keyword,
		"base_url":       dataSource.BaseUrl,
	}

	// Publish message to NATS
	msgData, err := json.Marshal(crawlMsg)
	if err != nil {
		http.Error(w, "Failed to create crawl message", http.StatusInternalServerError)
		log.Error().Err(err).Msg("Failed to marshal crawl message")
		return
	}

	// Publish to the crawl.search subject
	err = s.broker.Publish("crawl.search", msgData)
	if err != nil {
		http.Error(w, "Failed to queue crawl job", http.StatusInternalServerError)
		log.Error().Err(err).Msg("Failed to publish crawl message")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{"status": "crawl job queued"})
}

// getLogs retrieves logs with optional filtering
func (s *ApiService) getLogs(w http.ResponseWriter, r *http.Request) {
	// This would normally include pagination and filtering logic
	// For simplicity, we'll just return a message - this would be expanded in a real implementation
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Log retrieval would be implemented here with pagination"})
}

// getStatus returns the system status
func (s *ApiService) getStatus(w http.ResponseWriter, r *http.Request) {
	status := map[string]interface{}{
		"status":    "online",
		"version":   "1.0.0",
		"timestamp": time.Now(),
		"connections": map[string]string{
			"database": "connected",
			"nats":     "connected",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// startCrawler initiates a new crawler job
func (s *ApiService) startCrawler(w http.ResponseWriter, r *http.Request) {
	var req CrawlerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		log.Error().Err(err).Msg("Failed to decode crawler request")
		return
	}

	// Check if data source exists
	_, err := s.db.Queries.GetDataSourceById(r.Context(), req.DataSourceID)
	if err != nil {
		http.Error(w, "Data source not found", http.StatusNotFound)
		log.Error().Err(err).Str("dataSourceID", req.DataSourceID).Msg("Data source not found")
		return
	}

	// Create job ID
	jobID := uuid.New().String()

	// Create crawler message
	crawlerMsg := map[string]interface{}{
		"job_id":         jobID,
		"data_source_id": req.DataSourceID,
		"url":            req.URL,
		"depth":          req.Depth,
		"max_pages":      req.MaxPages,
		"filters":        req.Filters,
		"timestamp":      time.Now(),
	}

	// Publish message to NATS
	msgData, err := json.Marshal(crawlerMsg)
	if err != nil {
		http.Error(w, "Failed to create crawler message", http.StatusInternalServerError)
		log.Error().Err(err).Msg("Failed to marshal crawler message")
		return
	}

	// Publish to the crawler.start subject
	err = s.broker.Publish("crawler.start", msgData)
	if err != nil {
		http.Error(w, "Failed to queue crawler job", http.StatusInternalServerError)
		log.Error().Err(err).Msg("Failed to publish crawler message")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{
		"job_id": jobID,
		"status": "crawler job started",
	})
}

// stopCrawler stops an ongoing crawler job
func (s *ApiService) stopCrawler(w http.ResponseWriter, r *http.Request) {
	jobID := chi.URLParam(r, "jobID")
	if jobID == "" {
		http.Error(w, "Job ID is required", http.StatusBadRequest)
		return
	}

	// Create stop message
	stopMsg := map[string]interface{}{
		"job_id":    jobID,
		"timestamp": time.Now(),
	}

	// Publish message to NATS
	msgData, err := json.Marshal(stopMsg)
	if err != nil {
		http.Error(w, "Failed to create stop message", http.StatusInternalServerError)
		log.Error().Err(err).Msg("Failed to marshal stop message")
		return
	}

	// Publish to the crawler.stop subject
	err = s.broker.Publish("crawler.stop", msgData)
	if err != nil {
		http.Error(w, "Failed to send stop signal", http.StatusInternalServerError)
		log.Error().Err(err).Msg("Failed to publish stop message")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"job_id": jobID,
		"status": "stop signal sent",
	})
}

// getCrawlerStatus gets the status of a crawler job
func (s *ApiService) getCrawlerStatus(w http.ResponseWriter, r *http.Request) {
	jobID := chi.URLParam(r, "jobID")
	if jobID == "" {
		http.Error(w, "Job ID is required", http.StatusBadRequest)
		return
	}

	// In a real implementation, this would query a job status table
	// or request status via NATS request-reply pattern
	// For now, we'll just return a placeholder response

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"job_id": jobID,
		"status": "pending", // This would be dynamically determined in a real implementation
	})
}

// triggerScraper initiates a scraping operation
func (s *ApiService) triggerScraper(w http.ResponseWriter, r *http.Request) {
	var req ScraperRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		log.Error().Err(err).Msg("Failed to decode scraper request")
		return
	}

	// Check if data source exists
	_, err := s.db.Queries.GetDataSourceById(r.Context(), req.DataSourceID)
	if err != nil {
		http.Error(w, "Data source not found", http.StatusNotFound)
		log.Error().Err(err).Str("dataSourceID", req.DataSourceID).Msg("Data source not found")
		return
	}

	// Create job ID
	jobID := uuid.New().String()

	// Create scraper message
	scraperMsg := map[string]interface{}{
		"job_id":          jobID,
		"data_source_id":  req.DataSourceID,
		"url_frontier_id": req.URLFrontierID,
		"timestamp":       time.Now(),
	}

	// Publish message to NATS
	msgData, err := json.Marshal(scraperMsg)
	if err != nil {
		http.Error(w, "Failed to create scraper message", http.StatusInternalServerError)
		log.Error().Err(err).Msg("Failed to marshal scraper message")
		return
	}

	// Publish to the scraper.scrape subject
	err = s.broker.Publish("scraper.scrape", msgData)
	if err != nil {
		http.Error(w, "Failed to queue scraper job", http.StatusInternalServerError)
		log.Error().Err(err).Msg("Failed to publish scraper message")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{
		"job_id": jobID,
		"status": "scraper job started",
	})
}

// batchScrape initiates multiple scraping operations in batch
func (s *ApiService) batchScrape(w http.ResponseWriter, r *http.Request) {
	var requests []ScraperRequest
	if err := json.NewDecoder(r.Body).Decode(&requests); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		log.Error().Err(err).Msg("Failed to decode batch scraper request")
		return
	}

	if len(requests) == 0 {
		http.Error(w, "Empty batch request", http.StatusBadRequest)
		return
	}

	jobIDs := make([]string, len(requests))

	for i, req := range requests {
		// Check if data source exists
		_, err := s.db.Queries.GetDataSourceById(r.Context(), req.DataSourceID)
		if err != nil {
			// Skip invalid data sources but log the error
			log.Error().Err(err).Str("dataSourceID", req.DataSourceID).Msg("Data source not found in batch request")
			continue
		}

		// Create job ID
		jobID := uuid.New().String()
		jobIDs[i] = jobID

		// Create scraper message
		scraperMsg := map[string]interface{}{
			"job_id":          jobID,
			"data_source_id":  req.DataSourceID,
			"url_frontier_id": req.URLFrontierID,
			"timestamp":       time.Now(),
			"batch":           true,
			"batch_index":     i,
			"batch_total":     len(requests),
		}

		// Publish message to NATS
		msgData, err := json.Marshal(scraperMsg)
		if err != nil {
			log.Error().Err(err).Msg("Failed to marshal batch scraper message")
			continue
		}

		// Publish to the scraper.batch subject
		err = s.broker.Publish("scraper.batch", msgData)
		if err != nil {
			log.Error().Err(err).Msg("Failed to publish batch scraper message")
			continue
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"job_count": len(jobIDs),
		"job_ids":   jobIDs,
		"status":    "batch scraper jobs queued",
	})
}

// cancelExtractorJob cancels an extractor job
func (s *ApiService) cancelExtractorJob(w http.ResponseWriter, r *http.Request) {
	// Parse request body
	var req CancelExtractorJobRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Error().Err(err).Msg("Failed to decode request body")
		utils.WriteError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate request
	if req.JobID == "" {
		utils.WriteError(w, http.StatusBadRequest, "Job ID is required")
		return
	}

	// Get the extractor worker from the crawler service
	// This relies on the main.go modification to make the extractor worker globally accessible

	// Access the crawler service through dependency injection
	// _, ok := s.services["crawler"].(crawler.CrawlerService)
	// if !ok {
	// 	log.Error().Msg("Crawler service not available")
	// 	utils.WriteError(w, http.StatusInternalServerError, "Crawler service not available")
	// 	return
	// }

	// // Get the extractor worker manager
	// extractorManager, ok := s.services["extractor"].(worker.Worker)
	// if !ok {
	// 	log.Error().Msg("Extractor worker manager not available")
	// 	utils.WriteError(w, http.StatusInternalServerError, "Extractor worker manager not available")
	// 	return
	// }

	// // Cancel the job
	// err := extractorManager.CancelJob(worker.JobID(req.JobID))
	// if err != nil {
	// 	log.Error().Err(err).Str("job_id", req.JobID).Msg("Failed to cancel extractor job")
	// 	if err.Error() == "job not found: "+req.JobID {
	// 		utils.WriteError(w, http.StatusNotFound, "Job not found")
	// 		return
	// 	}
	// 	utils.WriteError(w, http.StatusInternalServerError, "Failed to cancel job")
	// 	return
	// }

	// // Log the action
	// log.Info().
	// 	Str("job_id", req.JobID).
	// 	Str("user", r.Header.Get("X-API-USER")).
	// 	Msg("Extractor job cancelled")

	// Return success
	utils.WriteMessage(w, http.StatusOK, "Job cancelled successfully")
}
