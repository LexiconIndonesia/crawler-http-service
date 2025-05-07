package crawler

import (
	"sync"
)

var (
	crawlerRegistry     = make(map[string]CrawlerCreator)
	crawlerRegistryLock sync.RWMutex
)

// RegisterCrawler registers a crawler creator function for a specific data source type
func RegisterCrawler(dataSourceType string, creator CrawlerCreator) {
	crawlerRegistryLock.Lock()
	defer crawlerRegistryLock.Unlock()
	crawlerRegistry[dataSourceType] = creator
}

// GetCrawlerRegistry returns the crawler registry
func GetCrawlerRegistry() map[string]CrawlerCreator {
	crawlerRegistryLock.RLock()
	defer crawlerRegistryLock.RUnlock()

	// Create a copy to avoid race conditions
	registryCopy := make(map[string]CrawlerCreator, len(crawlerRegistry))
	for k, v := range crawlerRegistry {
		registryCopy[k] = v
	}

	return registryCopy
}
