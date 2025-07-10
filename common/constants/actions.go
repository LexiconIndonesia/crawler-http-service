package constants

// ActionType defines the type of action a message represents.
type ActionType string

const (
	// CrawlAllAction triggers a full crawl of a data source.
	CrawlAllAction ActionType = "crawl:all"
	// CrawlByKeywordAction triggers a crawl for a specific keyword.
	CrawlByKeywordAction ActionType = "crawl:by_keyword"
	// CrawlByURLAction triggers a crawl for a specific URL.
	CrawlByURLAction ActionType = "crawl:by_url"

	// ScrapeByIDAction triggers scraping for a specific URL frontier ID.
	ScrapeByIDAction ActionType = "scrape:by_id"
	// ScrapeAllAction triggers scraping for all URL frontiers.
	ScrapeAllAction ActionType = "scrape:all"
)
