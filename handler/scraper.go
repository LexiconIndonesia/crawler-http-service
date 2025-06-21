package handler

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/LexiconIndonesia/crawler-http-service/common/config"
	"github.com/LexiconIndonesia/crawler-http-service/common/constants"
	"github.com/LexiconIndonesia/crawler-http-service/common/db"
	"github.com/LexiconIndonesia/crawler-http-service/common/messaging"
	"github.com/LexiconIndonesia/crawler-http-service/common/services"
	"github.com/LexiconIndonesia/crawler-http-service/common/utils"
	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

type ScraperHandler struct {
	db     *db.DB
	broker *messaging.NatsBroker
	router *chi.Mux
	cfg    config.Config
}

func NewScraperHandler(db *db.DB, broker *messaging.NatsBroker, cfg config.Config) *ScraperHandler {
	router := chi.NewRouter()

	h := &ScraperHandler{
		db:     db,
		broker: broker,
		router: router,
		cfg:    cfg,
	}

	router.Post("/", h.handleRunScraper)
	return h
}

func (h *ScraperHandler) Router() *chi.Mux {
	return h.router
}

func (h *ScraperHandler) handleRunScraper(w http.ResponseWriter, r *http.Request) {
	var p ScraperRunParams

	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		utils.WriteError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	validate := validator.New()
	if err := validate.Struct(p); err != nil {
		utils.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	dsRepo := services.NewDataSourceRepository(h.db.Queries)
	ds, err := dsRepo.GetByName(r.Context(), p.DataSource)
	if err != nil {
		utils.WriteError(w, http.StatusNotFound, "Data source not found")
		return
	}

	req := messaging.ScrapeRequest{
		ID:   uuid.NewString(),
		Type: constants.ActionType(p.Action),
	}

	payload := messaging.ScrapePayload{}
	if p.UrlFrontierID != nil {
		payload.URLFrontierID = *p.UrlFrontierID
	}
	req.Payload = payload

	topic := fmt.Sprintf("%s.extraction", ds.Name)
	msg, err := json.Marshal(req)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, "Failed to marshal message")
		return
	}

	if err := h.broker.PublishSync(r.Context(), topic, msg); err != nil {
		log.Error().Err(err).Msg("Failed to publish message")
		utils.WriteError(w, http.StatusInternalServerError, "Failed to publish message")
		return
	}

	utils.WriteJSON(w, http.StatusOK, map[string]string{"message": "success", "job_id": req.ID})
}

type ScraperRunParams struct {
	DataSource    string  `json:"data_source" validate:"required"`
	Action        string  `json:"action" validate:"required"`
	UrlFrontierID *string `json:"url_frontier_id"`
}
