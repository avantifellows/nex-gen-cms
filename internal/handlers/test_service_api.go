package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/avantifellows/nex-gen-cms/internal/models"
)

// AssembledTest is the service-API contract consumed by quiz-backend's CMS->quiz mapper:
// the Test (structure + marks cascade at test/subject/section/problem levels inside
// type_params) plus a flat list of fully-resolved Problems (text, options, answer,
// paragraph — images already base64-inline in the HTML from db-service). The mapper joins
// each problem to its ResProblem reference by ID. See task lms-cms-tests for the locked
// contract.
type AssembledTest struct {
	Test     *models.Test      `json:"test"`
	Problems []*models.Problem `json:"problems"`
}

// GetTestsJSON is the JSON sibling of GetTests (which renders HTMX rows). It lists active
// tests for a curriculum/grade/subtype so a session-creation surface (af_lms,
// quiz-creator) can present a picker instead of pasting a CMS URL. Query params mirror the
// HTMX route: curriculum-dropdown, grade-dropdown, testtype-dropdown.
func (h *TestsHandler) GetTestsJSON(responseWriter http.ResponseWriter, request *http.Request) {
	urlVals := request.URL.Query()

	curriculumId, gradeId, _ := getCurriculumGradeSubjectIds(urlVals)
	if curriculumId == 0 || gradeId == 0 {
		http.Error(responseWriter, "curriculum-dropdown and grade-dropdown are required", http.StatusBadRequest)
		return
	}

	tests, err := h.listTests(curriculumId, gradeId, urlVals.Get(TESTTYPE_DROPDOWN_NAME),
		urlVals.Get("sortColumn"), urlVals.Get("sortOrder"))
	if err != nil {
		http.Error(responseWriter, fmt.Sprintf("Error fetching tests: %v", err), http.StatusInternalServerError)
		return
	}

	writeJSON(responseWriter, tests)
}

// GetAssembledTestJSON returns a single test with all its problems inlined — the input
// contract for quiz-backend ingest. It reuses the same resolution the PDF/detail views use
// (getTest + getTestProblems), so the assembled shape stays in lockstep with what the CMS
// renders. Query params: id (test id), curriculum_id, grade_id.
func (h *TestsHandler) GetAssembledTestJSON(responseWriter http.ResponseWriter, request *http.Request) {
	if request.URL.Query().Get("id") == "" {
		http.Error(responseWriter, "id is required", http.StatusBadRequest)
		return
	}

	testPtr, code, err := h.getTest(responseWriter, request)
	if err != nil {
		http.Error(responseWriter, err.Error(), code)
		return
	}

	// getTestProblems writes its own http.Error and returns nil on failure.
	problems := h.getTestProblems(responseWriter, request)
	if problems == nil {
		return
	}

	writeJSON(responseWriter, AssembledTest{Test: testPtr, Problems: *problems})
}

// writeJSON marshals v as an application/json response.
func writeJSON(responseWriter http.ResponseWriter, v any) {
	responseWriter.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(responseWriter).Encode(v); err != nil {
		http.Error(responseWriter, fmt.Sprintf("Error encoding JSON: %v", err), http.StatusInternalServerError)
	}
}
