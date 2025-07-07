package models

import "encoding/json"

// ExtractionArtifact represents an artifact extracted from a page
type ExtractionArtifact struct {
	FileName    string `json:"file_name"`
	Size        int64  `json:"size"`
	ContentType string `json:"content_type"`
	URL         string `json:"url"`
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
