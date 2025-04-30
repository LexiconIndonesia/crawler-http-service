package singapore_supreme_court

import (
	"time"
)

// JudgmentItem represents a judgment from the Singapore Supreme Court
type JudgmentItem struct {
	ID             string    `json:"id"`
	Title          string    `json:"title"`
	URL            string    `json:"url"`
	JudgmentDate   time.Time `json:"judgment_date"`
	CourtName      string    `json:"court_name"`
	CaseNumber     string    `json:"case_number"`
	JudgmentPDFURL string    `json:"judgment_pdf_url"`
	Category       string    `json:"category"`
	Judges         []string  `json:"judges"`
	Parties        []string  `json:"parties"`
}

// SearchParams represents search parameters for the Singapore Supreme Court
type SearchParams struct {
	Keyword    string `json:"keyword"`
	StartDate  string `json:"start_date"` // Format: YYYY-MM-DD
	EndDate    string `json:"end_date"`   // Format: YYYY-MM-DD
	Category   string `json:"category"`
	SortBy     string `json:"sort_by"`
	SortOrder  string `json:"sort_order"` // asc or desc
	PageNumber int    `json:"page_number"`
	PageSize   int    `json:"page_size"`
}
