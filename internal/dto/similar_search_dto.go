package dto

type SimilarSearchRequest struct {
	Languages []SimilarSearchLanguage `json:"languages"`
}

type SimilarSearchLanguage struct {
	LangCode string `json:"lang_code"`
	Text     string `json:"text"`
}

type SimilarSearchResponse struct {
	Problems []SimilarProblemMatch `json:"problems"`
}

type SimilarProblemMatch struct {
	ID         int     `json:"id"`
	Code       string  `json:"code"`
	LangCode   string  `json:"lang_code"`
	MatchScore float64 `json:"match_score"`
}

// DuplicateProblemMatch is the template-ready form of a SimilarProblemMatch shown in
// duplicate_problems_modal.html — percent is pre-rounded so the template does no arithmetic.
// ID is carried through for a future "click to open the existing problem" link.
type DuplicateProblemMatch struct {
	ID           int
	Code         string
	LangCode     string
	MatchPercent int
}
