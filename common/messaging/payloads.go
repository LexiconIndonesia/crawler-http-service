package messaging

import "github.com/LexiconIndonesia/crawler-http-service/common/constants"

type CrawlRequest struct {
	ID      string               `json:"id"`
	Type    constants.ActionType `json:"type"`
	Payload CrawlPayload         `json:"payload"`
}

type ScrapeRequest struct {
	ID      string               `json:"id"`
	Type    constants.ActionType `json:"type"`
	Payload ScrapePayload        `json:"payload"`
}

type CrawlPayload struct {
	Keyword string `json:"keyword,omitempty"`
	URL     string `json:"url,omitempty"`
}

type ScrapePayload struct {
	URLFrontierID string `json:"url_frontier_id,omitempty"`
}
