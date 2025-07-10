package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/LexiconIndonesia/crawler-http-service/common/db"
	"github.com/LexiconIndonesia/crawler-http-service/common/models"
	"github.com/LexiconIndonesia/crawler-http-service/common/services"
	"github.com/LexiconIndonesia/crawler-http-service/common/utils"
	"github.com/LexiconIndonesia/crawler-http-service/repository"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type DataSourceHandler struct {
	db     *db.DB
	router *chi.Mux
}

func NewDataSourceHandler(db *db.DB) *DataSourceHandler {
	h := &DataSourceHandler{
		db: db,
	}

	r := chi.NewRouter()
	r.Get("/", h.handleListDataSources)
	r.Post("/", h.handleCreateDataSource)
	r.Get("/{id}", h.handleGetDataSource)
	r.Put("/{id}", h.handleUpdateDataSource)
	r.Delete("/{id}", h.handleDeleteDataSource)

	h.router = r
	return h
}

func (h *DataSourceHandler) Router() *chi.Mux {
	return h.router
}

func (h *DataSourceHandler) handleListDataSources(w http.ResponseWriter, r *http.Request) {
	dsRepo := services.NewDataSourceRepository(h.db.Queries)
	dataSources, err := dsRepo.GetAll(r.Context())
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, "Failed to get data sources")
		return
	}

	var results []models.DataSource
	for _, ds := range dataSources {
		var config map[string]interface{}
		if ds.Config != nil {
			if err := json.Unmarshal(ds.Config, &config); err != nil {
				utils.WriteError(w, http.StatusInternalServerError, "Failed to unmarshal data source config")
				return
			}
		}

		results = append(results, models.DataSource{
			ID:          ds.ID,
			Name:        ds.Name,
			Country:     ds.Country,
			SourceType:  ds.SourceType,
			BaseUrl:     ds.BaseUrl,
			Description: ds.Description,
			Config:      config,
			IsActive:    ds.IsActive,
			CreatedAt:   ds.CreatedAt,
			UpdatedAt:   ds.UpdatedAt,
			DeletedAt:   ds.DeletedAt,
		})
	}

	utils.WriteJSON(w, http.StatusOK, results)
}

func (h *DataSourceHandler) handleCreateDataSource(w http.ResponseWriter, r *http.Request) {
	var p models.DataSource
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		utils.WriteError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}
	defer r.Body.Close()

	config, err := json.Marshal(p.Config)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, "Failed to marshal data source config")
		return
	}

	params := repository.CreateDataSourceParams{
		ID:          uuid.NewString(),
		Name:        p.Name,
		Country:     p.Country,
		SourceType:  p.SourceType,
		BaseUrl:     p.BaseUrl,
		Description: p.Description,
		Config:      config,
		IsActive:    p.IsActive,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	dsRepo := services.NewDataSourceRepository(h.db.Queries)
	dataSource, err := dsRepo.Create(r.Context(), params)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, "Failed to create data source")
		return
	}

	var resConfig map[string]interface{}
	if dataSource.Config != nil {
		if err := json.Unmarshal(dataSource.Config, &resConfig); err != nil {
			utils.WriteError(w, http.StatusInternalServerError, "Failed to unmarshal data source config")
			return
		}
	}

	result := models.DataSource{
		ID:          dataSource.ID,
		Name:        dataSource.Name,
		Country:     dataSource.Country,
		SourceType:  dataSource.SourceType,
		BaseUrl:     dataSource.BaseUrl,
		Description: dataSource.Description,
		Config:      resConfig,
		IsActive:    dataSource.IsActive,
		CreatedAt:   dataSource.CreatedAt,
		UpdatedAt:   dataSource.UpdatedAt,
		DeletedAt:   dataSource.DeletedAt,
	}

	utils.WriteJSON(w, http.StatusCreated, result)
}

func (h *DataSourceHandler) handleGetDataSource(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	dsRepo := services.NewDataSourceRepository(h.db.Queries)
	dataSource, err := dsRepo.GetByID(r.Context(), id)
	if err != nil {
		utils.WriteError(w, http.StatusNotFound, "Data source not found")
		return
	}

	var config map[string]interface{}
	if dataSource.Config != nil {
		if err := json.Unmarshal(dataSource.Config, &config); err != nil {
			utils.WriteError(w, http.StatusInternalServerError, "Failed to unmarshal data source config")
			return
		}
	}

	result := models.DataSource{
		ID:          dataSource.ID,
		Name:        dataSource.Name,
		Country:     dataSource.Country,
		SourceType:  dataSource.SourceType,
		BaseUrl:     dataSource.BaseUrl,
		Description: dataSource.Description,
		Config:      config,
		IsActive:    dataSource.IsActive,
		CreatedAt:   dataSource.CreatedAt,
		UpdatedAt:   dataSource.UpdatedAt,
		DeletedAt:   dataSource.DeletedAt,
	}

	utils.WriteJSON(w, http.StatusOK, result)
}

func (h *DataSourceHandler) handleUpdateDataSource(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var p models.DataSource
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		utils.WriteError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}
	defer r.Body.Close()

	config, err := json.Marshal(p.Config)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, "Failed to marshal data source config")
		return
	}

	params := repository.UpdateDataSourceParams{
		ID:          id,
		Name:        p.Name,
		Country:     p.Country,
		SourceType:  p.SourceType,
		BaseUrl:     p.BaseUrl,
		Description: p.Description,
		Config:      config,
		IsActive:    p.IsActive,
		UpdatedAt:   time.Now(),
	}

	dsRepo := services.NewDataSourceRepository(h.db.Queries)

	dataSource, err := dsRepo.Update(r.Context(), params)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, "Failed to update data source")
		return
	}

	var resConfig map[string]interface{}
	if dataSource.Config != nil {
		if err := json.Unmarshal(dataSource.Config, &resConfig); err != nil {
			utils.WriteError(w, http.StatusInternalServerError, "Failed to unmarshal data source config")
			return
		}
	}

	result := models.DataSource{
		ID:          dataSource.ID,
		Name:        dataSource.Name,
		Country:     dataSource.Country,
		SourceType:  dataSource.SourceType,
		BaseUrl:     dataSource.BaseUrl,
		Description: dataSource.Description,
		Config:      resConfig,
		IsActive:    dataSource.IsActive,
		CreatedAt:   dataSource.CreatedAt,
		UpdatedAt:   dataSource.UpdatedAt,
		DeletedAt:   dataSource.DeletedAt,
	}

	utils.WriteJSON(w, http.StatusOK, result)
}

func (h *DataSourceHandler) handleDeleteDataSource(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	dsRepo := services.NewDataSourceRepository(h.db.Queries)
	if err := dsRepo.Delete(r.Context(), id); err != nil {
		utils.WriteError(w, http.StatusInternalServerError, "Failed to delete data source")
		return
	}

	utils.WriteJSON(w, http.StatusOK, map[string]string{"message": "success"})
}
