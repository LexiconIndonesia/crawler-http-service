package services

import (
	"context"
	"time"

	"github.com/LexiconIndonesia/crawler-http-service/common/models"
	"github.com/LexiconIndonesia/crawler-http-service/repository"
)

// UrlFrontierService defines the interface for URL frontier database operations
type UrlFrontierService interface {
	// Create creates a new URL frontier
	Create(ctx context.Context, frontier repository.UrlFrontier) (repository.UrlFrontier, error)

	// CreateBatch creates multiple URL frontiers in a batch
	CreateBatch(ctx context.Context, frontiers []repository.UrlFrontier) ([]repository.UrlFrontier, error)

	// GetByID gets a URL frontier by ID
	GetByID(ctx context.Context, id string) (repository.UrlFrontier, error)

	// GetByURL gets a URL frontier by URL and data source ID
	GetByURL(ctx context.Context, url string, dataSourceID string) (repository.UrlFrontier, error)

	// GetPendingByDataSource gets pending URL frontiers for a data source
	GetPendingByDataSource(ctx context.Context, dataSourceID string, limit int) ([]repository.UrlFrontier, error)

	// UpdateStatus updates the status of a URL frontier
	UpdateStatus(ctx context.Context, id string, status models.UrlFrontierStatus, errorMessage string) error

	// UpdateStatusBatch updates the status of a URL frontier by batch
	UpdateStatusBatch(ctx context.Context, id []string, status models.UrlFrontierStatus, errorMessage string) error

	// UpdateNextCrawl updates the next crawl time of a URL frontier
	UpdateNextCrawl(ctx context.Context, id string, nextCrawlAt time.Time) error

	// IncrementAttempts increments the attempt count of a URL frontier
	IncrementAttempts(ctx context.Context, id string) error
}

// ExtractionService defines the interface for extraction database operations
type ExtractionService interface {
	// Create creates a new extraction
	Create(ctx context.Context, extraction repository.Extraction) (repository.Extraction, error)

	// CreateBatch creates multiple extractions in a batch
	CreateBatch(ctx context.Context, extractions []repository.Extraction) ([]repository.Extraction, error)

	// GetByID gets an extraction by ID
	GetByID(ctx context.Context, id string) (repository.Extraction, error)

	// GetByUrlFrontierID gets extractions by URL frontier ID
	GetByUrlFrontierID(ctx context.Context, urlFrontierID string) ([]repository.Extraction, error)

	// UpdateLinks updates the artifact and raw page links of an extraction
	UpdateLinks(ctx context.Context, id string, artifactLink string, rawPageLink string) error

	// CreateVersion creates a new version of an extraction
	CreateVersion(ctx context.Context, extractionID string, version int, content string, metadata map[string]interface{}, pageHash string) error
}

// DataSourceService defines the interface for data source database operations
type DataSourceService interface {
	// GetByID gets a data source by ID
	GetByID(ctx context.Context, id string) (repository.DataSource, error)

	// GetByType gets data sources by type
	GetByType(ctx context.Context, sourceType string) ([]repository.DataSource, error)

	// GetActive gets active data sources
	GetActive(ctx context.Context) ([]repository.DataSource, error)

	// Get Datasource by name
	GetByName(ctx context.Context, name string) (repository.DataSource, error)

	// Create creates a new data source
	Create(ctx context.Context, arg repository.CreateDataSourceParams) (repository.DataSource, error)

	// Update updates a data source
	Update(ctx context.Context, arg repository.UpdateDataSourceParams) (repository.DataSource, error)

	// Delete deletes a data source
	Delete(ctx context.Context, id string) error

	// GetAll gets all data sources
	GetAll(ctx context.Context) ([]repository.DataSource, error)
}
