package dto

type TestsContainingProblemsRequest struct {
	ProblemIDs []int `json:"problem_ids"`
}

type TestsContainingProblemsResponse struct {
	ProblemTests []ProblemTestsAssociation `json:"problem_tests"`
}

type ProblemTestsAssociation struct {
	ProblemID   int              `json:"problem_id"`
	ProblemCode string           `json:"problem_code"`
	Tests       []ProblemTestRef `json:"tests"`
}

type ProblemTestRef struct {
	TestID   int        `json:"test_id"`
	TestCode string     `json:"test_code"`
	Name     []TestName `json:"name"`
}

type TestName struct {
	Resource string `json:"resource"`
	LangCode string `json:"lang_code"`
}
