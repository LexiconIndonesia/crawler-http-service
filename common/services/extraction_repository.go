package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/LexiconIndonesia/crawler-http-service/repository"
	"github.com/jackc/pgx/v5/pgtype"
)

// ExtractionRepository is a PostgreSQL implementation of crawler.ExtractionRepository
type ExtractionRepository struct {
	db *repository.Queries
}

// NewExtractionRepository creates a new PostgreSQL ExtractionRepository
func NewExtractionRepository(db *repository.Queries) ExtractionService {
	return &ExtractionRepository{
		db: db,
	}
}

// Create creates a new extraction
func (r *ExtractionRepository) Create(ctx context.Context, extraction repository.Extraction) (repository.Extraction, error) {
	params := repository.UpsertExtractionParams(extraction)
	batchParams := []repository.UpsertExtractionParams{params}
	batchResults := r.db.UpsertExtraction(ctx, batchParams)

	var err error
	batchResults.Exec(func(i int, e error) {
		if e != nil {
			err = e
		}
	})

	if err != nil {
		return repository.Extraction{}, err
	}

	// Fetch the created record
	return r.GetByID(ctx, extraction.ID)
}

// CreateBatch creates multiple extractions in a batch
func (r *ExtractionRepository) CreateBatch(ctx context.Context, extractions []repository.Extraction) ([]repository.Extraction, error) {
	if len(extractions) == 0 {
		return nil, fmt.Errorf("no extractions provided")
	}

	params := make([]repository.UpsertExtractionParams, len(extractions))
	for i, extraction := range extractions {
		if extraction.ID == "" {
			return nil, fmt.Errorf("extraction ID cannot be empty at index %d", i)
		}
		if extraction.UrlFrontierID == "" {
			return nil, fmt.Errorf("URL frontier ID cannot be empty at index %d", i)
		}
		params[i] = repository.UpsertExtractionParams(extraction)
	}

	batchResults := r.db.UpsertExtraction(ctx, params)

	var errs []error
	var failedIndices []int
	batchResults.Exec(func(i int, err error) {
		if err != nil {
			errs = append(errs, fmt.Errorf("failed to create extraction at index %d: %w", i, err))
			failedIndices = append(failedIndices, i)
		}
	})

	if len(errs) > 0 {
		// If all operations failed, return the first error
		if len(failedIndices) == len(extractions) {
			return nil, errs[0]
		}
		// If some operations succeeded, return partial results with errors
		result := make([]repository.Extraction, 0, len(extractions)-len(failedIndices))
		for i, extraction := range extractions {
			if !contains(failedIndices, i) {
				result = append(result, extraction)
			}
		}
		return result, fmt.Errorf("partial batch creation failed: %v", errs)
	}

	return extractions, nil
}

// GetByID gets an extraction by ID
func (r *ExtractionRepository) GetByID(ctx context.Context, id string) (repository.Extraction, error) {
	if id == "" {
		return repository.Extraction{}, fmt.Errorf("extraction ID cannot be empty")
	}
	return r.db.GetExtractionById(ctx, id)
}

// GetByUrlFrontierID gets extractions by URL frontier ID
func (r *ExtractionRepository) GetByUrlFrontierID(ctx context.Context, urlFrontierID string) ([]repository.Extraction, error) {
	if urlFrontierID == "" {
		return nil, fmt.Errorf("URL frontier ID cannot be empty")
	}
	return r.db.GetExtractionsByUrlFrontierID(ctx, urlFrontierID)
}

// UpdateLinks updates the artifact and raw page links of an extraction
func (r *ExtractionRepository) UpdateLinks(ctx context.Context, id string, artifactLink string, rawPageLink string) error {
	if id == "" {
		return fmt.Errorf("extraction ID cannot be empty")
	}

	// Get the current extraction
	extraction, err := r.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get extraction: %w", err)
	}

	// Update links
	artifactLinkPg := pgtype.Text{}
	if artifactLink != "" {
		artifactLinkPg.String = artifactLink
		artifactLinkPg.Valid = true
	}

	rawPageLinkPg := pgtype.Text{}
	if rawPageLink != "" {
		rawPageLinkPg.String = rawPageLink
		rawPageLinkPg.Valid = true
	}

	extraction.ArtifactLink = artifactLinkPg
	extraction.RawPageLink = rawPageLinkPg
	extraction.UpdatedAt = time.Now()

	params := repository.UpsertExtractionParams(extraction)
	batchParams := []repository.UpsertExtractionParams{params}
	batchResults := r.db.UpsertExtraction(ctx, batchParams)

	var updateErr error
	batchResults.Exec(func(i int, e error) {
		if e != nil {
			updateErr = fmt.Errorf("failed to update extraction links: %w", e)
		}
	})

	return updateErr
}

// CreateVersion creates a new version of an extraction
func (r *ExtractionRepository) CreateVersion(ctx context.Context, extractionID string, version int, content string, metadata map[string]interface{}, pageHash string) error {
	if extractionID == "" {
		return fmt.Errorf("extraction ID cannot be empty")
	}
	if version < 1 {
		return fmt.Errorf("version must be greater than 0")
	}

	// Get the current extraction
	extraction, err := r.GetByID(ctx, extractionID)
	if err != nil {
		return fmt.Errorf("failed to get extraction: %w", err)
	}

	// Convert metadata to JSON
	metadataBytes, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	// Create version fields
	siteContentPg := pgtype.Text{}
	if content != "" {
		siteContentPg.String = content
		siteContentPg.Valid = true
	}

	pageHashPg := pgtype.Text{}
	if pageHash != "" {
		pageHashPg.String = pageHash
		pageHashPg.Valid = true
	}

	// Create a new version record using sqlc-generated method
	versionID := fmt.Sprintf("%s-v%d", extractionID, version)
	versionParams := repository.CreateExtractionVersionParams{
		ID:           versionID,
		ExtractionID: extractionID,
		SiteContent:  siteContentPg,
		ArtifactLink: extraction.ArtifactLink,
		RawPageLink:  extraction.RawPageLink,
		Metadata:     metadataBytes,
		PageHash:     pageHashPg,
		Version:      int32(version),
		CreatedAt:    time.Now(),
	}
	if err := r.db.CreateExtractionVersion(ctx, versionParams); err != nil {
		return fmt.Errorf("failed to create version record: %w", err)
	}

	extraction.SiteContent = siteContentPg
	extraction.Metadata = metadataBytes
	extraction.PageHash = pageHashPg
	extraction.Version = int32(version)
	extraction.UpdatedAt = time.Now()

	params := repository.UpsertExtractionParams(extraction)
	batchParams := []repository.UpsertExtractionParams{params}
	batchResults := r.db.UpsertExtraction(ctx, batchParams)

	var updateErr error
	batchResults.Exec(func(i int, e error) {
		if e != nil {
			updateErr = fmt.Errorf("failed to update extraction: %w", e)
		}
	})

	return updateErr
}

// Helper function to check if a slice contains a value
func contains(slice []int, value int) bool {
	for _, v := range slice {
		if v == value {
			return true
		}
	}
	return false
}
