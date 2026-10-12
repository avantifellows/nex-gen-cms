package dto

import "github.com/avantifellows/nex-gen-cms/internal/models"

type TestsContainingProblemsRequest struct {
	ProblemIDs []int `json:"problem_ids"`
}

type TestsContainingProblemsResponse struct {
	ProblemTests []ProblemTestsAssociation `json:"problem_tests"`
}

type ProblemTestsAssociation struct {
	ProblemID   int              `json:"problem_id"`
	ProblemCode string           `json:"problem_code"`
	Tests       []models.LinkedTest `json:"tests"`
}
