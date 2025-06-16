package messaging

import "github.com/LexiconIndonesia/crawler-http-service/common/constants"

type CrawlRequest struct {
	Type    constants.ActionType `json:"type"`
	Payload string               `json:"payload,omitempty"`
}

type ScrapeRequest struct {
	Type          constants.ActionType `json:"type"`
	UrlFrontierID string               `json:"url_frontier_id"`
}
