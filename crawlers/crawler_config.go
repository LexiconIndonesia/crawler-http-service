package crawler

import (
	"fmt"

	"github.com/adryanev/go-http-service-template/common"
)

// CrawlerConfig holds configuration for a crawler
type CrawlerConfig struct {
	Name          string
	DataSourceID  string
	GCSBucket     string
	GCSFolder     string
	GCSHTMLFolder string
}

// NewCrawlerConfig creates a new crawler configuration
func NewCrawlerConfig(name common.CrawlerType, dataSourceID string) *CrawlerConfig {
	config := &CrawlerConfig{
		Name:         string(name),
		DataSourceID: dataSourceID,
		GCSBucket:    common.GCSBucketName,
	}

	// Set default folders based on crawler name
	config.GCSFolder = fmt.Sprintf("%s/%s", config.Name, "pdf")
	config.GCSHTMLFolder = fmt.Sprintf("%s/%s", config.Name, "html")

	return config
}

// GetGCSFolder returns the GCS folder path
func (c *CrawlerConfig) GetGCSFolder() string {
	return c.GCSFolder
}

// GetGCSHTMLFolder returns the GCS HTML folder path
func (c *CrawlerConfig) GetGCSHTMLFolder() string {
	return c.GCSHTMLFolder
}

// GetDataSourceID returns the data source ID
func (c *CrawlerConfig) GetDataSourceID() string {
	return c.DataSourceID
}

type URLFrontierStatus int16

const (
	URL_FRONTIER_STATUS_NEW     URLFrontierStatus = iota
	URL_FRONTIER_STATUS_CRAWLED URLFrontierStatus = iota
	URL_FRONTIER_STATUS_ERROR   URLFrontierStatus = iota
)
