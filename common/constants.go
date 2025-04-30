package common

const (
	// AppName is the name of the application
	AppName = "lexicon-crawler"

	// GCS constants
	GCSBucketName = "lexicon-bo-bucket"
)

// CrawlerType represents the type of crawler
type CrawlerType string

const (
	// IndonesiaSupremeCourt represents the Indonesia Supreme Court crawler
	IndonesiaSupremeCourt CrawlerType = "indonesia-supreme-court-crawler"
	// OpenSanction represents the Open Sanction crawler
	OpenSanction CrawlerType = "open-sanction-crawler"
	// MalaysiaPesalahRasuah represents the Malaysia Pesalah Rasuah crawler
	MalaysiaPesalahRasuah CrawlerType = "malaysia-pesalah-rasuah-crawler"
	// LKPPBlacklist represents the LKPP Blacklist crawler
	LKPPBlacklist CrawlerType = "lkpp-blacklist-crawler"
	// SingaporeSupremeCourt represents the Singapore Supreme Court crawler
	SingaporeSupremeCourt CrawlerType = "singapore-supreme-court-crawler"
)
