package dto

type ProblemTestAssociation struct {
	ProblemID   int              `json:"problem_id"`
	ProblemCode string           `json:"problem_code"`
	Tests       []ProblemTestRef `json:"tests"`
}

type ProblemTestRef struct {
	TestID   int    `json:"test_id"`
	TestCode string `json:"test_code"`
}
