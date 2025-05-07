package crawler

import (
	"context"
	"time"

	"github.com/LexiconIndonesia/crawler-http-service/repository"
)

// UrlFrontierStatus represents the status of a URL frontier
type UrlFrontierStatus int

const (
	// UrlFrontierStatusPending indicates the URL is pending crawling
	UrlFrontierStatusPending UrlFrontierStatus = 0
	// UrlFrontierStatusCrawled indicates the URL has been crawled
	UrlFrontierStatusCrawled UrlFrontierStatus = 1
	// UrlFrontierStatusChanged indicates the URL content has changed since last crawl
	UrlFrontierStatusChanged UrlFrontierStatus = 2
	// UrlFrontierStatusFailed indicates crawling the URL failed
	UrlFrontierStatusFailed UrlFrontierStatus = 3
	// UrlFrontierStatusScheduled indicates the URL is scheduled for crawling
	UrlFrontierStatusScheduled UrlFrontierStatus = 4
	// UrlFrontierStatusProcessing indicates the URL is being processed
	UrlFrontierStatusProcessing UrlFrontierStatus = 5
)

// UrlFrontierRepository defines the interface for URL frontier database operations
type UrlFrontierRepository interface {
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
	UpdateStatus(ctx context.Context, id string, status UrlFrontierStatus, errorMessage string) error

	// UpdateNextCrawl updates the next crawl time of a URL frontier
	UpdateNextCrawl(ctx context.Context, id string, nextCrawlAt time.Time) error

	// IncrementAttempts increments the attempt count of a URL frontier
	IncrementAttempts(ctx context.Context, id string) error
}

// ExtractionRepository defines the interface for extraction database operations
type ExtractionRepository interface {
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

// DataSourceRepository defines the interface for data source database operations
type DataSourceRepository interface {
	// GetByID gets a data source by ID
	GetByID(ctx context.Context, id string) (DataSource, error)

	// GetByType gets data sources by type
	GetByType(ctx context.Context, sourceType string) ([]DataSource, error)

	// GetActive gets active data sources
	GetActive(ctx context.Context) ([]DataSource, error)
}

// DataSource represents a data source
type DataSource struct {
	ID          string
	Name        string
	Country     string
	SourceType  string
	BaseURL     string
	Description string
	ConfigType  string
	Config      map[string]interface{}
	IsActive    bool
}
