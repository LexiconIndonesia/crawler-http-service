package crawler

import (
	"context"
	"encoding/json"
	"io"
	"strings"
	"time"

	"github.com/adryanev/go-http-service-template/common/db"
	"github.com/adryanev/go-http-service-template/common/messaging"
	"github.com/adryanev/go-http-service-template/repository"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rs/zerolog/log"
	"github.com/samber/lo"
)

// CrawlerService provides operations for crawlers
type CrawlerService interface {
	// URL Frontier operations
	GetUnscrappedUrlFrontiers(ctx context.Context, limit int32) ([]repository.UrlFrontier, error)
	UpsertUrlFrontier(ctx context.Context, frontier repository.UrlFrontier) error
	UpsertUrlFrontiers(ctx context.Context, frontiers []repository.UrlFrontier) error
	UpdateFrontierStatuses(ctx context.Context, statuses []lo.Tuple2[string, int16]) error
	GetUrlFrontierById(ctx context.Context, id string) (repository.UrlFrontier, error)
	GetUrlFrontierByUrl(ctx context.Context, url string) (repository.UrlFrontier, error)
	UpdateUrlFrontierStatus(ctx context.Context, id string, status int16) error

	// Data Source operations
	GetDataSourceById(ctx context.Context, id string) (repository.DataSource, error)
	GetActiveDataSources(ctx context.Context) ([]repository.DataSource, error)
	UpsertDataSource(ctx context.Context, dataSource repository.DataSource) error

	// Extraction operations
	CreateExtraction(ctx context.Context, id string, urlFrontierID string, pageHash string, data map[string]interface{}) error
	GetExtractionByUrlFrontierID(ctx context.Context, urlFrontierID string) (repository.Extraction, error)
	CreateExtractionVersion(ctx context.Context, extractionID string, pageHash string, data map[string]interface{}) error

	// Content operations
	FetchContent(ctx context.Context, url string) (io.Reader, error)

	// Logging operations
	CreateCrawlerLog(ctx context.Context, logParams repository.CreateCrawlerLogParams) error

	// Messaging operations
	GetNatsClient() *messaging.NatsClient
}

// CrawlerServiceImpl is the default implementation of CrawlerService
type CrawlerServiceImpl struct {
	db         *db.DB
	natsClient *messaging.NatsClient
}

// NewCrawlerService creates a new crawler service
func NewCrawlerService(db *db.DB) CrawlerService {
	return &CrawlerServiceImpl{db: db}
}

// SetNatsClient sets the NATS client
func (s *CrawlerServiceImpl) SetNatsClient(client *messaging.NatsClient) {
	s.natsClient = client
}

// GetNatsClient returns the NATS client
func (s *CrawlerServiceImpl) GetNatsClient() *messaging.NatsClient {
	return s.natsClient
}

// UpsertUrlFrontiers creates or updates multiple URL frontiers
func (s *CrawlerServiceImpl) UpsertUrlFrontiers(ctx context.Context, urlFrontiers []repository.UrlFrontier) error {
	tx, err := s.db.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	queries := s.db.Queries.WithTx(tx)

	for _, frontier := range urlFrontiers {
		now := time.Now()
		nextCrawlTime := now.Add(time.Hour * 24)

		err := queries.UpsertUrlFrontier(ctx, repository.UpsertUrlFrontierParams{
			ID:           frontier.ID,
			DataSourceID: frontier.DataSourceID,
			Domain:       frontier.Domain,
			Url:          frontier.Url,
			Keyword:      frontier.Keyword,
			Priority:     frontier.Priority,
			Status:       frontier.Status,
			Attempts:     frontier.Attempts,
			LastCrawledAt: pgtype.Timestamptz{
				Time:  now,
				Valid: true,
			},
			NextCrawlAt: pgtype.Timestamptz{
				Time:  nextCrawlTime,
				Valid: true,
			},
			ErrorMessage: frontier.ErrorMessage,
			Metadata:     frontier.Metadata,
			CreatedAt:    now,
			UpdatedAt:    now,
		})
		if err != nil {
			log.Error().Err(err).Str("url", frontier.Url).Msg("Failed to upsert URL frontier")
			return err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}

	return nil
}

// UpsertUrlFrontier creates or updates a single URL frontier
func (s *CrawlerServiceImpl) UpsertUrlFrontier(ctx context.Context, frontier repository.UrlFrontier) error {
	frontiers := []repository.UrlFrontier{frontier}
	return s.UpsertUrlFrontiers(ctx, frontiers)
}

// UpdateFrontierStatuses updates the status of multiple frontiers
func (s *CrawlerServiceImpl) UpdateFrontierStatuses(ctx context.Context, statuses []lo.Tuple2[string, int16]) error {
	tx, err := s.db.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	queries := s.db.Queries.WithTx(tx)

	params := make([]repository.UpdateUrlFrontierStatusParams, 0, len(statuses))
	for _, status := range statuses {
		params = append(params, repository.UpdateUrlFrontierStatusParams{
			ID:           status.A,
			Status:       status.B,
			ErrorMessage: pgtype.Text{},
			UpdatedAt:    time.Now(),
		})
	}

	batch := queries.UpdateUrlFrontierStatus(ctx, params)
	if err := batch.Close(); err != nil {
		log.Error().Err(err).Msg("Failed to update frontier statuses")
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}

	return nil
}

// UpdateUrlFrontierStatus updates the status of a single URL frontier
func (s *CrawlerServiceImpl) UpdateUrlFrontierStatus(ctx context.Context, id string, status int16) error {
	statuses := []lo.Tuple2[string, int16]{{id, status}}
	return s.UpdateFrontierStatuses(ctx, statuses)
}

// GetUrlFrontierByUrl retrieves a URL frontier by URL
func (s *CrawlerServiceImpl) GetUrlFrontierByUrl(ctx context.Context, url string) (repository.UrlFrontier, error) {
	tx, err := s.db.Pool.Begin(ctx)
	if err != nil {
		return repository.UrlFrontier{}, err
	}
	defer tx.Rollback(ctx)

	queries := s.db.Queries.WithTx(tx)

	frontier, err := queries.GetUrlFrontierByUrl(ctx, url)
	if err != nil {
		log.Debug().Err(err).Str("url", url).Msg("Failed to get URL frontier by URL")
		return repository.UrlFrontier{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return repository.UrlFrontier{}, err
	}

	return frontier, nil
}

// GetUrlFrontierById retrieves a URL frontier by ID
func (s *CrawlerServiceImpl) GetUrlFrontierById(ctx context.Context, id string) (repository.UrlFrontier, error) {
	tx, err := s.db.Pool.Begin(ctx)
	if err != nil {
		return repository.UrlFrontier{}, err
	}
	defer tx.Rollback(ctx)

	queries := s.db.Queries.WithTx(tx)

	frontier, err := queries.GetUrlFrontierById(ctx, id)
	if err != nil {
		log.Debug().Err(err).Str("id", id).Msg("Failed to get URL frontier by ID")
		return repository.UrlFrontier{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return repository.UrlFrontier{}, err
	}

	return frontier, nil
}

// GetUnscrappedUrlFrontiers retrieves URL frontiers that haven't been scraped yet
func (s *CrawlerServiceImpl) GetUnscrappedUrlFrontiers(ctx context.Context, limit int32) ([]repository.UrlFrontier, error) {
	tx, err := s.db.Pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	queries := s.db.Queries.WithTx(tx)

	frontiers, err := queries.GetUnscrappedUrlFrontiers(ctx, repository.GetUnscrappedUrlFrontiersParams{
		DataSourceID: "1",
		Status:       0,
		Limit:        limit,
	})
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return frontiers, nil
}

// GetDataSourceById retrieves a data source by ID
func (s *CrawlerServiceImpl) GetDataSourceById(ctx context.Context, id string) (repository.DataSource, error) {
	tx, err := s.db.Pool.Begin(ctx)
	if err != nil {
		return repository.DataSource{}, err
	}
	defer tx.Rollback(ctx)

	queries := s.db.Queries.WithTx(tx)

	dataSource, err := queries.GetDataSourceById(ctx, id)
	if err != nil {
		log.Debug().Err(err).Str("id", id).Msg("Failed to get data source by ID")
		return repository.DataSource{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return repository.DataSource{}, err
	}

	return dataSource, nil
}

// GetActiveDataSources retrieves all active data sources
func (s *CrawlerServiceImpl) GetActiveDataSources(ctx context.Context) ([]repository.DataSource, error) {
	tx, err := s.db.Pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	queries := s.db.Queries.WithTx(tx)

	dataSources, err := queries.GetActiveDataSources(ctx)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to get active data sources")
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return dataSources, nil
}

// UpsertDataSource creates or updates a data source
func (s *CrawlerServiceImpl) UpsertDataSource(ctx context.Context, dataSource repository.DataSource) error {
	tx, err := s.db.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	queries := s.db.Queries.WithTx(tx)

	err = queries.UpsertDataSource(ctx, repository.UpsertDataSourceParams{
		ID:          dataSource.ID,
		Name:        dataSource.Name,
		Country:     dataSource.Country,
		SourceType:  dataSource.SourceType,
		BaseUrl:     dataSource.BaseUrl,
		Description: dataSource.Description,
		Config:      dataSource.Config,
		IsActive:    dataSource.IsActive,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	})
	if err != nil {
		log.Error().Err(err).Str("id", dataSource.ID).Msg("Failed to upsert data source")
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}

	return nil
}

// CreateExtraction creates a new extraction record
func (s *CrawlerServiceImpl) CreateExtraction(ctx context.Context, id string, urlFrontierID string, pageHash string, data map[string]interface{}) error {
	tx, err := s.db.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	queries := s.db.Queries.WithTx(tx)

	// Serialize data to JSON
	dataJson, err := json.Marshal(data)
	if err != nil {
		log.Error().Err(err).Msg("Failed to marshal extraction data")
		return err
	}

	now := time.Now()
	// Create extraction record
	params := []repository.UpsertExtractionParams{
		{
			ID:             id,
			UrlFrontierID:  urlFrontierID,
			SiteContent:    pgtype.Text{String: string(dataJson), Valid: true},
			ArtifactLink:   pgtype.Text{},
			RawPageLink:    pgtype.Text{},
			ExtractionDate: now,
			ContentType:    pgtype.Text{},
			Metadata:       json.RawMessage("{}"),
			Language:       "",
			PageHash:       pgtype.Text{String: pageHash, Valid: true},
			Version:        1,
			CreatedAt:      now,
			UpdatedAt:      now,
		},
	}

	batch := queries.UpsertExtraction(ctx, params)
	if err := batch.Close(); err != nil {
		log.Error().Err(err).Msg("Failed to create extraction")
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}

	return nil
}

// GetExtractionByUrlFrontierID retrieves an extraction by URL frontier ID
func (s *CrawlerServiceImpl) GetExtractionByUrlFrontierID(ctx context.Context, urlFrontierID string) (repository.Extraction, error) {
	// This would require a custom query since it's not generated by sqlc
	// For this example, we'll return an empty extraction
	return repository.Extraction{}, nil
}

// CreateExtractionVersion creates a new version of an extraction
func (s *CrawlerServiceImpl) CreateExtractionVersion(ctx context.Context, extractionID string, pageHash string, data map[string]interface{}) error {
	// This would require a custom implementation to create version records
	// For this example, we'll just log the attempt
	log.Info().
		Str("extraction_id", extractionID).
		Str("page_hash", pageHash).
		Msg("Creating extraction version")
	return nil
}

// FetchContent fetches content from a URL
func (s *CrawlerServiceImpl) FetchContent(ctx context.Context, url string) (io.Reader, error) {
	// For simplicity, just return a string reader with mocked content
	// In a real app, this would make an HTTP request
	siteContent := []byte("<html><body><h1>Mocked Content</h1></body></html>")

	// Return the content as a reader
	return strings.NewReader(string(siteContent)), nil
}

// CreateCrawlerLog creates a crawler log entry
func (s *CrawlerServiceImpl) CreateCrawlerLog(ctx context.Context, logParams repository.CreateCrawlerLogParams) error {
	tx, err := s.db.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	queries := s.db.Queries.WithTx(tx)

	err = queries.CreateCrawlerLog(ctx, logParams)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create crawler log")
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}

	return nil
}
