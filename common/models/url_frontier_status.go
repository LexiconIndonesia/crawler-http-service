package models

// UrlFrontierStatus represents the status of a URL frontier
type UrlFrontierStatus int

const (
	// UrlFrontierStatusPending indicates the URL is pending crawling
	UrlFrontierStatusPending UrlFrontierStatus = 0
	// UrlFrontierStatusCrawled indicates the URL has been crawled
	UrlFrontierStatusCrawled UrlFrontierStatus = 1
	// UrlFrontierStatusChanged indicates the URL content has changed since last crawl
	UrlFrontierStatusChanged UrlFrontierStatus = 2
	// UrlFrontierStatusFailed indicates crawling the URL failed
	UrlFrontierStatusFailed UrlFrontierStatus = 3
	// UrlFrontierStatusScheduled indicates the URL is scheduled for crawling
	UrlFrontierStatusScheduled UrlFrontierStatus = 4
	// UrlFrontierStatusProcessing indicates the URL is being processed
	UrlFrontierStatusProcessing UrlFrontierStatus = 5
)
