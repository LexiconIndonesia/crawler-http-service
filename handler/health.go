package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/LexiconIndonesia/crawler-http-service/common/db"
	"github.com/LexiconIndonesia/crawler-http-service/common/logger"
	"github.com/LexiconIndonesia/crawler-http-service/common/utils"
	"github.com/go-chi/chi/v5"
)

type HealthHandler struct {
	db         *db.DB
	logService *logger.LogService
	router     *chi.Mux
}

func NewHealthHandler(db *db.DB) *HealthHandler {
	h := &HealthHandler{
		db:         db,
		logService: logger.NewLogService(db),
	}

	r := chi.NewRouter()
	r.Get("/", h.handleHealthCheck)
	r.Get("/database", h.handleDatabaseHealth)
	r.Get("/logs", h.handleLogsHealth)

	h.router = r
	return h
}

func (h *HealthHandler) Router() *chi.Mux {
	return h.router
}

func (h *HealthHandler) handleHealthCheck(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().UTC(),
		"service":   "crawler-http-service",
	}

	utils.WriteJSON(w, http.StatusOK, response)
}

func (h *HealthHandler) handleDatabaseHealth(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	// Check database health
	dbErr := h.logService.CheckDatabaseHealth(ctx)

	// Get database stats
	dbStats := h.logService.GetDatabaseStats()

	response := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().UTC(),
		"database": map[string]interface{}{
			"status": "healthy",
			"stats":  dbStats,
		},
	}

	if dbErr != nil {
		response["status"] = "unhealthy"
		response["database"].(map[string]interface{})["status"] = "unhealthy"
		response["database"].(map[string]interface{})["error"] = dbErr.Error()
		utils.WriteJSON(w, http.StatusServiceUnavailable, response)
		return
	}

	utils.WriteJSON(w, http.StatusOK, response)
}

func (h *HealthHandler) handleLogsHealth(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	// Check database health for logging
	dbErr := h.logService.CheckDatabaseHealth(ctx)

	// Get database stats
	dbStats := h.logService.GetDatabaseStats()

	response := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().UTC(),
		"logging": map[string]interface{}{
			"status":         "healthy",
			"database_stats": dbStats,
		},
	}

	if dbErr != nil {
		response["status"] = "unhealthy"
		response["logging"].(map[string]interface{})["status"] = "unhealthy"
		response["logging"].(map[string]interface{})["error"] = dbErr.Error()
		utils.WriteJSON(w, http.StatusServiceUnavailable, response)
		return
	}

	utils.WriteJSON(w, http.StatusOK, response)
}
