package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/LexiconIndonesia/crawler-http-service/common/db"
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

	utils.WriteJSON(w, http.StatusOK, dataSources)
}

func (h *DataSourceHandler) handleCreateDataSource(w http.ResponseWriter, r *http.Request) {
	var params repository.CreateDataSourceParams
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		utils.WriteError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}
	defer r.Body.Close()

	dsRepo := services.NewDataSourceRepository(h.db.Queries)

	params.ID = uuid.NewString()
	params.CreatedAt = time.Now()
	params.UpdatedAt = time.Now()

	dataSource, err := dsRepo.Create(r.Context(), params)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, "Failed to create data source")
		return
	}

	utils.WriteJSON(w, http.StatusCreated, dataSource)
}

func (h *DataSourceHandler) handleGetDataSource(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	dsRepo := services.NewDataSourceRepository(h.db.Queries)
	dataSource, err := dsRepo.GetByID(r.Context(), id)
	if err != nil {
		utils.WriteError(w, http.StatusNotFound, "Data source not found")
		return
	}

	utils.WriteJSON(w, http.StatusOK, dataSource)
}

func (h *DataSourceHandler) handleUpdateDataSource(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var params repository.UpdateDataSourceParams
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		utils.WriteError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}
	defer r.Body.Close()

	dsRepo := services.NewDataSourceRepository(h.db.Queries)

	params.ID = id
	params.UpdatedAt = time.Now()

	dataSource, err := dsRepo.Update(r.Context(), params)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, "Failed to update data source")
		return
	}

	utils.WriteJSON(w, http.StatusOK, dataSource)
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
