package crawler

import (
	"errors"
)

// Common error constants
var (
	// ErrNotImplemented is already defined elsewhere in the package

	// ErrInvalidConfig is returned when an invalid configuration is provided
	ErrInvalidConfig = errors.New("invalid configuration")

	// ErrUnsupportedDataSource is returned when an unsupported data source is specified
	ErrUnsupportedDataSource = errors.New("unsupported data source")

	// ErrCrawlerFailed is returned when a crawler operation fails
	ErrCrawlerFailed = errors.New("crawler operation failed")
)
