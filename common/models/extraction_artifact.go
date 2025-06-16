package models

import "encoding/json"

// ExtractionArtifact represents an artifact extracted from a page
type ExtractionArtifact struct {
	FileName    string
	Size        int64
	ContentType string
	URL         string
}

// to json
func (e *ExtractionArtifact) ToJson() ([]byte, error) {
	return json.Marshal(e)
}

// from json
func ExtractionArtifactFromJson(j []byte) (*ExtractionArtifact, error) {
	var e ExtractionArtifact
	err := json.Unmarshal(j, &e)
	if err != nil {
		return nil, err
	}
	return &e, nil
}
