package models

import (
	"time"

	"github.com/LexiconIndonesia/crawler-http-service/repository"
	"github.com/jackc/pgx/v5/pgtype"
)

type CrawlerLogResponse struct {
	ID           string      `json:"id"`
	DataSourceID string      `json:"data_source_id"`
	JobID        pgtype.Text `json:"job_id"`
	EventType    string      `json:"event_type"`
	Message      pgtype.Text `json:"message"`
	Details      interface{} `json:"details"`
	CreatedAt    time.Time   `json:"created_at"`
}

type WorkDetailResponse struct {
	Job  repository.Job       `json:"job"`
	Logs []CrawlerLogResponse `json:"logs"`
}
