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

type CrawlerHandler struct {
	db     *db.DB
	broker *messaging.NatsBroker
	router *chi.Mux
	cfg    config.Config
}

func NewCrawlerHandler(db *db.DB, broker *messaging.NatsBroker, cfg config.Config) *CrawlerHandler {
	router := chi.NewRouter()

	h := &CrawlerHandler{
		db:     db,
		broker: broker,
		router: router,
		cfg:    cfg,
	}

	router.Post("/", h.handleRunCrawler)
	return h
}

func (h *CrawlerHandler) Router() *chi.Mux {
	return h.router
}

func (h *CrawlerHandler) handleRunCrawler(w http.ResponseWriter, r *http.Request) {
	var p CrawlerRunParams

	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		utils.WriteError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	validate := validator.New()
	if err := validate.Struct(p); err != nil {
		utils.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	// find data source
	dsRepo := services.NewDataSourceRepository(h.db.Queries)
	ds, err := dsRepo.GetByName(r.Context(), p.DataSource)
	if err != nil {
		utils.WriteError(w, http.StatusNotFound, "Data source not found")
		return
	}

	// create task
	req := messaging.CrawlRequest{
		ID:   uuid.NewString(),
		Type: constants.ActionType(p.Action),
	}

	payload := messaging.CrawlPayload{}
	if p.Url != nil {
		payload.URL = *p.Url
	}
	if p.Keyword != nil {
		payload.Keyword = *p.Keyword
	}
	req.Payload = payload

	// publish message
	topic := fmt.Sprintf("%s.frontier", ds.Name)
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

type CrawlerRunParams struct {
	DataSource string  `json:"data_source" validate:"required"`
	Action     string  `json:"action" validate:"required"`
	Url        *string `json:"url"`
	Keyword    *string `json:"keyword"`
}
