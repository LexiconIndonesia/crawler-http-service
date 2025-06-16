package isc

type UrlFrontierMetadata struct {
	Title            string   `json:"title"`
	DecisionDate     string   `json:"decision_date"`
	RegistrationDate string   `json:"registration_date"`
	UploadDate       string   `json:"upload_date"`
	Breadcrumbs      []string `json:"breadcrumbs"`
}

var EmptyUrlFrontierMetadata UrlFrontierMetadata

var EmptyMetadata ExtractionMetadata

type ExtractionMetadata struct {
	Id                       string `json:"id"`
	Title                    string `json:"title"`
	Defendant                string `json:"defendant"`
	Number                   string `json:"number"`
	ProcessLevel             string `json:"process_level"`
	Classification           string `json:"classification"`
	Keywords                 string `json:"keywords"`
	Year                     string `json:"year"`
	RegistrationDate         string `json:"registration_date"`
	JudicalInstitution       string `json:"judicial_institution"`
	TypeOfJudicalInstitution string `json:"type_of_judicial_institution"`
	ChiefJustice             string `json:"chief_justice"`
	MemberJudge              string `json:"member_judge"`
	Clerk                    string `json:"clerk"`
	Verdict                  string `json:"verdict"`
	OtherVerdict             string `json:"other_verdict"`
	VerdictNote              string `json:"verdict_note"`
	CourtDate                string `json:"court_date"`
	AnnouncementDate         string `json:"announcement_date"`
	Rule                     string `json:"rule"`
	Abstract                 string `json:"abstract"`
	PdfUrl                   string `json:"pdf_url"`
}
