package models

// ExtractionArtifact represents an artifact extracted from a page
type ExtractionArtifact struct {
	FileName    string
	FileContent []byte
	ContentType string
	URL         string
}
