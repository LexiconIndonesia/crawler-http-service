package models

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

type DataSource struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Country     string                 `json:"country"`
	SourceType  string                 `json:"source_type"`
	BaseUrl     pgtype.Text            `json:"base_url"`
	Description pgtype.Text            `json:"description"`
	Config      map[string]interface{} `json:"config"`
	IsActive    bool                   `json:"is_active"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	DeletedAt   pgtype.Timestamptz     `json:"deleted_at"`
}
