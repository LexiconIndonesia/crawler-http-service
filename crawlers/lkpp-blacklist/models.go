package lkpp

import (
	"time"
)

// BlacklistItem represents a company or entity in the LKPP blacklist
type BlacklistItem struct {
	ID                 string    `json:"id"`
	CompanyName        string    `json:"company_name"`
	BlacklistReason    string    `json:"blacklist_reason"`
	BlacklistPeriod    string    `json:"blacklist_period"` // e.g. "2 years"
	StartDate          time.Time `json:"start_date"`
	EndDate            time.Time `json:"end_date"`
	BlacklistingEntity string    `json:"blacklisting_entity"` // Government entity that blacklisted
	DirectorName       string    `json:"director_name"`
	Address            string    `json:"address"`
	NPWP               string    `json:"npwp"` // Tax identification number
	NIB                string    `json:"nib"`  // Business identification number
}

// SearchParams represents search parameters for the LKPP Blacklist
type SearchParams struct {
	Keyword   string `json:"keyword"`
	StartDate string `json:"start_date"` // Format: YYYY-MM-DD
	EndDate   string `json:"end_date"`   // Format: YYYY-MM-DD
	Page      int    `json:"page"`
	PageSize  int    `json:"page_size"`
}
