package messaging

// ExtractRunMessage represents the message for extract.run event
type ExtractRunMessage struct {
	URLFrontierID string `json:"url_frontier_id"`
	DataSourceID  string `json:"data_source_id"`
	JobID         string `json:"job_id,omitempty"`
}

// Constants for NATS subjects
const (
	SubjectCrawlSearch = "crawl.search"
	SubjectExtractRun  = "extract.run"
)
