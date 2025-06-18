package handler

import (
	"math"
	"net/http"
	"strconv"

	"github.com/LexiconIndonesia/crawler-http-service/common/config"
	"github.com/LexiconIndonesia/crawler-http-service/common/db"
	"github.com/LexiconIndonesia/crawler-http-service/common/models"
	"github.com/LexiconIndonesia/crawler-http-service/common/utils"
	"github.com/LexiconIndonesia/crawler-http-service/common/work"
	"github.com/LexiconIndonesia/crawler-http-service/repository"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type WorkManagerHandler struct {
	db          *db.DB
	router      *chi.Mux
	cfg         config.Config
	workManager *work.WorkManager
}

func NewWorkManagerHandler(db *db.DB, cfg config.Config) *WorkManagerHandler {
	router := chi.NewRouter()

	h := &WorkManagerHandler{
		db:          db,
		router:      router,
		cfg:         cfg,
		workManager: work.NewWorkManager(db),
	}

	router.Get("/", h.handleListWorks)
	router.Get("/{jobID}", h.handleGetWork)
	router.Post("/{jobID}/cancel", h.handleCancelWork)

	return h
}

func (h *WorkManagerHandler) Router() *chi.Mux {
	return h.router
}

func (h *WorkManagerHandler) handleListWorks(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit < 1 {
		limit = 10
	}

	offset := (page - 1) * limit

	status := r.URL.Query().Get("status")
	search := r.URL.Query().Get("q")

	listParams := repository.ListJobsParams{
		Limit:  int32(limit),
		Offset: int32(offset),
	}
	if status != "" {
		listParams.Status = pgtype.Text{String: status, Valid: true}
	}
	if search != "" {
		listParams.Search = pgtype.Text{String: search, Valid: true}
	}

	jobs, err := h.db.Queries.ListJobs(r.Context(), listParams)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, "Failed to get jobs")
		return
	}

	countParams := repository.CountJobsParams{}
	if status != "" {
		countParams.Status = pgtype.Text{String: status, Valid: true}
	}
	if search != "" {
		countParams.Search = pgtype.Text{String: search, Valid: true}
	}

	total, err := h.db.Queries.CountJobs(r.Context(), countParams)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, "Failed to count jobs")
		return
	}

	meta := models.MetaResponse{
		CurrentPage: int64(page),
		LastPage:    int64(math.Ceil(float64(total) / float64(limit))),
		PerPage:     int64(limit),
		Total:       total,
	}
	response := models.BasePaginationResponse{
		Data: jobs,
		Meta: meta,
	}

	utils.WriteJSON(w, http.StatusOK, response)
}

func (h *WorkManagerHandler) handleGetWork(w http.ResponseWriter, r *http.Request) {
	jobID := chi.URLParam(r, "jobID")

	job, err := h.db.Queries.GetJobByID(r.Context(), jobID)
	if err != nil {
		utils.WriteError(w, http.StatusNotFound, "Job not found")
		return
	}

	logs, err := h.db.Queries.GetCrawlerLogsByJobId(r.Context(), pgtype.Text{String: jobID, Valid: true})
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, "Failed to get job logs")
		return
	}

	type WorkDetail struct {
		Job  repository.Job          `json:"job"`
		Logs []repository.CrawlerLog `json:"logs"`
	}

	detail := WorkDetail{
		Job:  job,
		Logs: logs,
	}

	utils.WriteJSON(w, http.StatusOK, models.BaseResponse{Data: detail})
}

func (h *WorkManagerHandler) handleCancelWork(w http.ResponseWriter, r *http.Request) {
	jobID := chi.URLParam(r, "jobID")

	err := h.workManager.Cancel(r.Context(), jobID)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	utils.WriteJSON(w, http.StatusOK, map[string]string{"message": "success"})
}
