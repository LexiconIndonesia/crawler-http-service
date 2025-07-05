package services

import (
	"context"
	"strings"
	"time"

	"github.com/LexiconIndonesia/crawler-http-service/common/models"
	"github.com/LexiconIndonesia/crawler-http-service/repository"
	"github.com/jackc/pgx/v5/pgtype"
)

// UrlFrontierRepository is a PostgreSQL implementation of crawler.UrlFrontierRepository
type UrlFrontierRepository struct {
	db *repository.Queries
}

// NewUrlFrontierRepository creates a new PostgreSQL UrlFrontierRepository
func NewUrlFrontierRepository(db *repository.Queries) UrlFrontierService {
	return &UrlFrontierRepository{
		db: db,
	}
}

// Create creates a new URL frontier
func (r *UrlFrontierRepository) Create(ctx context.Context, frontier repository.UrlFrontier) (repository.UrlFrontier, error) {
	params := repository.UpsertUrlFrontierParams(frontier)
	err := r.db.UpsertUrlFrontier(ctx, params)
	if err != nil {
		return repository.UrlFrontier{}, err
	}

	// Fetch the created record
	return r.GetByID(ctx, frontier.ID)
}

// CreateBatch creates multiple URL frontiers in a batch
func (r *UrlFrontierRepository) CreateBatch(ctx context.Context, frontiers []repository.UrlFrontier) ([]repository.UrlFrontier, error) {
	params := make([]repository.UpsertUrlFrontiersParams, len(frontiers))
	for i, frontier := range frontiers {
		params[i] = repository.UpsertUrlFrontiersParams(frontier)
	}

	batchResults := r.db.UpsertUrlFrontiers(ctx, params)

	var errs []error
	batchResults.Exec(func(i int, err error) {
		if err != nil {
			errs = append(errs, err)
		}
	})

	if len(errs) > 0 {
		return nil, errs[0] // Return the first error
	}

	// Return the created frontiers
	// Note: This is a simplified approach; in a real implementation,
	// you might want to fetch all the created frontiers
	result := make([]repository.UrlFrontier, len(frontiers))
	for i, frontier := range frontiers {
		result[i] = frontier
	}

	return result, nil
}

// GetByID gets a URL frontier by ID
func (r *UrlFrontierRepository) GetByID(ctx context.Context, id string) (repository.UrlFrontier, error) {
	return r.db.GetUrlFrontierById(ctx, id)
}

// GetByURL gets a URL frontier by URL and data source ID
func (r *UrlFrontierRepository) GetByURL(ctx context.Context, url string, dataSourceID string) (repository.UrlFrontier, error) {
	// The function GetUrlFrontierByUrl only takes URL as parameter, but we need to filter by dataSourceID as well
	// We could modify the generated code to support this, but for now, let's fetch by URL and check dataSourceID
	frontier, err := r.db.GetUrlFrontierByUrl(ctx, url)
	if err != nil {
		return repository.UrlFrontier{}, err
	}

	// Check if dataSourceID matches
	if frontier.DataSourceID != dataSourceID {
		return repository.UrlFrontier{}, nil // Return empty frontier
	}

	return frontier, nil
}

// GetPendingByDataSource gets pending URL frontiers for a data source
func (r *UrlFrontierRepository) GetPendingByDataSource(ctx context.Context, dataSourceID string, limit int) ([]repository.UrlFrontier, error) {
	params := repository.GetUnscrappedUrlFrontiersParams{
		DataSourceID: dataSourceID,
		Status:       int16(models.UrlFrontierStatusPending),
		Limit:        int32(limit),
	}

	return r.db.GetUnscrappedUrlFrontiers(ctx, params)
}

// UpdateStatus updates the status of a URL frontier
func (r *UrlFrontierRepository) UpdateStatus(ctx context.Context, id string, status models.UrlFrontierStatus, errorMessage string) error {
	errMsg := pgtype.Text{}
	if errorMessage != "" {
		errMsg.String = errorMessage
		errMsg.Valid = true
	}

	params := repository.UpdateUrlFrontierStatusParams{
		ID:           id,
		Status:       int16(status),
		ErrorMessage: errMsg,
		UpdatedAt:    time.Now(),
	}

	err := r.db.UpdateUrlFrontierStatus(ctx, params)

	return err
}

// UpdateStatusBatch updates the status of a URL frontier by batch
func (r *UrlFrontierRepository) UpdateStatusBatch(ctx context.Context, id []string, status models.UrlFrontierStatus, errorMessage string) error {
	errMsg := pgtype.Text{}
	if errorMessage != "" {
		errMsg.String = errorMessage
		errMsg.Valid = true
	}

	params := repository.UpdateUrlFrontierStatusBatchParams{
		ID:           "{" + strings.Join(id, ",") + "}",
		Status:       int16(status),
		ErrorMessage: errMsg,
		UpdatedAt:    time.Now(),
	}

	err := r.db.UpdateUrlFrontierStatusBatch(ctx, params)

	if err != nil {
		return err
	}

	return err
}

// UpdateNextCrawl updates the next crawl time of a URL frontier
func (r *UrlFrontierRepository) UpdateNextCrawl(ctx context.Context, id string, nextCrawlAt time.Time) error {
	// Get the current frontier
	frontier, err := r.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// Update nextCrawlAt
	nextCrawl := pgtype.Timestamptz{}
	nextCrawl.Time = nextCrawlAt
	nextCrawl.Valid = true
	frontier.NextCrawlAt = nextCrawl
	frontier.UpdatedAt = time.Now()

	// Update the frontier
	params := repository.UpsertUrlFrontierParams(frontier)
	return r.db.UpsertUrlFrontier(ctx, params)
}

// IncrementAttempts increments the attempt count of a URL frontier
func (r *UrlFrontierRepository) IncrementAttempts(ctx context.Context, id string) error {
	// Get the current frontier
	frontier, err := r.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// Increment attempts
	frontier.Attempts++
	frontier.UpdatedAt = time.Now()

	// Update the frontier
	params := repository.UpsertUrlFrontierParams(frontier)
	return r.db.UpsertUrlFrontier(ctx, params)
}
