package services

import (
	"context"

	"github.com/LexiconIndonesia/crawler-http-service/repository"
)

// DataSourceRepository is a PostgreSQL implementation of crawler.DataSourceRepository
type DataSourceRepository struct {
	db *repository.Queries
}

// NewDataSourceRepository creates a new PostgreSQL DataSourceRepository
func NewDataSourceRepository(db *repository.Queries) DataSourceService {
	return &DataSourceRepository{
		db: db,
	}
}

// GetByID gets a data source by ID
func (r *DataSourceRepository) GetByID(ctx context.Context, id string) (repository.DataSource, error) {
	dataSource, err := r.db.GetDataSourceById(ctx, id)
	if err != nil {
		return repository.DataSource{}, err
	}

	return dataSource, nil
}

// GetByType gets data sources by type
func (r *DataSourceRepository) GetByType(ctx context.Context, sourceType string) ([]repository.DataSource, error) {
	// Assuming there's a generated function to get data sources by type
	// This function would need to be defined in the sqlc queries
	// For now, we'll return an empty slice
	// In a real implementation, you'd implement this to query the database
	return []repository.DataSource{}, nil
}

// GetActive gets active data sources
func (r *DataSourceRepository) GetActive(ctx context.Context) ([]repository.DataSource, error) {
	dataSources, err := r.db.GetActiveDataSources(ctx)
	if err != nil {
		return nil, err
	}

	return dataSources, nil
}

func (r *DataSourceRepository) GetByName(ctx context.Context, name string) (repository.DataSource, error) {
	dataSource, err := r.db.GetDataSourceByName(ctx, name)
	if err != nil {
		return repository.DataSource{}, err

	}

	return dataSource, nil
}

// Create creates a new data source
func (r *DataSourceRepository) Create(ctx context.Context, arg repository.CreateDataSourceParams) (repository.DataSource, error) {
	dataSource, err := r.db.CreateDataSource(ctx, arg)
	if err != nil {
		return repository.DataSource{}, err
	}

	return dataSource, nil
}

// Update updates a data source
func (r *DataSourceRepository) Update(ctx context.Context, arg repository.UpdateDataSourceParams) (repository.DataSource, error) {
	dataSource, err := r.db.UpdateDataSource(ctx, arg)
	if err != nil {
		return repository.DataSource{}, err
	}

	return dataSource, nil
}

// Delete deletes a data source
func (r *DataSourceRepository) Delete(ctx context.Context, id string) error {
	return r.db.DeleteDataSource(ctx, id)
}

// GetAll gets all data sources
func (r *DataSourceRepository) GetAll(ctx context.Context) ([]repository.DataSource, error) {
	dataSources, err := r.db.GetAllDataSources(ctx)
	if err != nil {
		return nil, err
	}

	return dataSources, nil
}
