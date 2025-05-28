package singapore_supreme_court

var EmptyMetadata Metadata

type Metadata struct {
	Title              string   `json:"title"`
	Defendant          string   `json:"defendant"`
	Numbers            []string `json:"numbers"`
	CitationNumber     string   `json:"citation_number"`
	Classifications    []string `json:"classifications"`
	Year               string   `json:"year"`
	JudicalInstitution string   `json:"judicial_institution"`
	Judges             string   `json:"judges"`
	Counsel            string   `json:"counsel"`
	Verdict            string   `json:"verdict"`
	VerdictMarkdown    string   `json:"verdict_markdown"`
	DecisionDate       string   `json:"decision_date"`
	PdfUrl             string   `json:"pdf_url"`
}
