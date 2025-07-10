package ssc

import (
	"encoding/json"

	"github.com/LexiconIndonesia/crawler-http-service/common/models"
)

var EmptyMetadata Metadata

type Metadata struct {
	Title               string                      `json:"title"`
	Defendant           string                      `json:"defendant"`
	Numbers             []string                    `json:"numbers"`
	CitationNumber      string                      `json:"citation_number"`
	Classifications     []string                    `json:"classifications"`
	Year                string                      `json:"year"`
	JudicalInstitution  string                      `json:"judicial_institution"`
	Judges              string                      `json:"judges"`
	Counsel             string                      `json:"counsel"`
	Verdict             string                      `json:"verdict"`
	VerdictMarkdown     string                      `json:"verdict_markdown"`
	DecisionDate        string                      `json:"decision_date"`
	PdfUrl              string                      `json:"pdf_url"`
	ExtractionArtifacts []models.ExtractionArtifact `json:"extraction_artifacts"`
}

type UrlFrontierMetadata struct {
	CitationNumber string   `json:"citation_number"`
	DecisionDate   string   `json:"decision_date"`
	Title          string   `json:"title"`
	Categories     []string `json:"categories"`
	CaseNumbers    []string `json:"case_numbers"`
}

func (m *UrlFrontierMetadata) ToJson() ([]byte, error) {
	return json.Marshal(m)
}

func UrlFrontierMetadataFromJson(j []byte) (*UrlFrontierMetadata, error) {
	var m UrlFrontierMetadata
	err := json.Unmarshal(j, &m)
	return &m, err
}
