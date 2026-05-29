package handlers

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/avantifellows/nex-gen-cms/config"
	"github.com/avantifellows/nex-gen-cms/internal/constants"
	"github.com/avantifellows/nex-gen-cms/internal/dto"
	"github.com/avantifellows/nex-gen-cms/internal/handlers/handlerutils"
	"github.com/avantifellows/nex-gen-cms/internal/models"
	"github.com/avantifellows/nex-gen-cms/internal/pdfimport"
	"github.com/avantifellows/nex-gen-cms/internal/views"
	"github.com/avantifellows/nex-gen-cms/utils"
)

const (
	pdfImportDefaultDifficulty = "medium"
	pdfImportLangCode          = "en"
	importTestReviewTemplate   = "import_test_review.html"
)

// PdfImportHandler handles HTTP endpoints for PDF question import and post-import review.
type PdfImportHandler struct {
	problemsHandler *ProblemsHandler
	testsHandler    *TestsHandler
}

func NewPdfImportHandler(problemsHandler *ProblemsHandler, testsHandler *TestsHandler) *PdfImportHandler {
	return &PdfImportHandler{
		problemsHandler: problemsHandler,
		testsHandler:    testsHandler,
	}
}

// pdfImportCreatedFixture stores identifiers of draft objects created during PDF import.
// This lets later runs skip calling create-problems and create-test APIs.
type pdfImportCreatedFixture struct {
	TestID     int   `json:"test_id"`
	ProblemIDs []int `json:"problem_ids"`
}

// ExtractQuestionsFromPDF parses an uploaded PDF, streams progress as SSE, batch-saves problems, then completes.
func (h *PdfImportHandler) ExtractQuestionsFromPDF(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	writeSSE := func(event string, payload any) {
		data, err := json.Marshal(payload)
		if err != nil {
			return
		}
		fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, data)
		flusher.Flush()
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)

	onProgress := func(percent int, stage string) {
		writeSSE("progress", map[string]any{"percent": percent, "stage": stage})
	}

	// If a created fixture is configured, reuse the already-created draft objects in DB
	// (skips create-problems + create-test; extraction may still be skipped via PDF_IMPORT_FIXTURE).
	if fixturePath := config.GetEnv("PDF_IMPORT_CREATED_FIXTURE", ""); fixturePath != "" {
		fixture, err := loadPDFImportCreatedFixture(fixturePath)
		if err != nil {
			writeSSE("error", map[string]string{"message": err.Error()})
			return
		}
		if onProgress != nil {
			onProgress(100, "Reusing draft test/problems from fixture")
		}
		writeSSE("complete", map[string]any{
			"question_count": len(fixture.ProblemIDs),
			"test_id":        fixture.TestID,
		})
		return
	}

	problems, curriculumGrades, testSubtype, examID, err := extractProblemsFromRequest(r, onProgress)
	if err != nil {
		writeSSE("error", map[string]string{"message": err.Error()})
		return
	}

	if len(problems) > 0 {
		onProgress(92, "Saving questions to server…")
		reqBody, err := marshalPDFImportBatchBody(problems, curriculumGrades)
		if err != nil {
			writeSSE("error", map[string]string{"message": fmt.Sprintf("failed to prepare batch request: %v", err)})
			return
		}
		createdProblems, err := h.problemsHandler.postBatchProblems(reqBody)
		if err != nil {
			writeSSE("error", map[string]string{"message": fmt.Sprintf("failed to save questions: %v", err)})
			return
		}

		onProgress(96, "Creating draft test…")
		testPtr, err := h.createDraftTestFromProblems(createdProblems, curriculumGrades, testSubtype, examID)
		if err != nil {
			writeSSE("error", map[string]string{"message": fmt.Sprintf("failed to create draft test: %v", err)})
			return
		}

		// Optional: persist created draft IDs to a fixture for future re-runs without DB growth.
		if createdFixturePath := config.GetEnv("PDF_IMPORT_CREATED_DUMP_PATH", ""); createdFixturePath != "" {
			ids := make([]int, 0, len(createdProblems))
			for _, p := range createdProblems {
				ids = append(ids, p.ID)
			}
			fixture := pdfImportCreatedFixture{TestID: testPtr.ID, ProblemIDs: ids}
			if err := writePDFImportCreatedFixture(createdFixturePath, fixture); err != nil {
				// Non-fatal: user can still proceed with review.
				onProgress(99, fmt.Sprintf("Warning: failed to write created fixture: %v", err))
			}
		}

		onProgress(100, "Questions saved and draft test created")
		writeSSE("complete", map[string]any{
			"question_count": len(createdProblems),
			"test_id":        testPtr.ID,
		})
		return
	}

	writeSSE("complete", map[string]any{"question_count": len(problems)})
}

func extractProblemsFromRequest(r *http.Request, onProgress pdfimport.ProgressFunc) ([]models.Problem, []models.CurriculumGrade, string, int8, error) {
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		return nil, nil, "", 0, fmt.Errorf("failed to parse form: %w", err)
	}

	curriculumGrades, err := handlerutils.ParseCurriculumGradesFromForm(r.Form)
	if err != nil {
		return nil, nil, "", 0, err
	}

	testSubtype := r.FormValue("modal-testType")
	if testSubtype == "" {
		return nil, nil, "", 0, fmt.Errorf("test type is required")
	}

	examID, err := utils.StringToIntType[int8](r.FormValue("modal-examType"))
	if err != nil {
		return nil, nil, "", 0, fmt.Errorf("invalid exam type")
	}

	file, _, err := r.FormFile("pdf_file")
	if err != nil {
		return nil, nil, "", 0, fmt.Errorf("no PDF file provided: %w", err)
	}
	defer file.Close()

	pdfBytes, err := io.ReadAll(file)
	if err != nil {
		return nil, nil, "", 0, fmt.Errorf("failed to read PDF: %w", err)
	}

	if fixturePath := config.GetEnv("PDF_IMPORT_FIXTURE", ""); fixturePath != "" {
		problems, err := loadPDFImportFixture(fixturePath)
		if err != nil {
			return nil, nil, "", 0, err
		}
		if onProgress != nil {
			onProgress(90, "Loaded questions from fixture (skipped AI extraction)")
		}
		return problems, curriculumGrades, testSubtype, examID, nil
	}

	if onProgress != nil {
		onProgress(5, "PDF received, starting extraction…")
	}

	apiKey := config.GetEnv("OPENROUTER_API_KEY", "")
	problems, _, err := pdfimport.ExtractProblemsWithProgress(pdfBytes, apiKey, onProgress)
	if err != nil {
		return nil, nil, "", 0, err
	}

	if dumpPath := config.GetEnv("PDF_IMPORT_DUMP_PATH", ""); dumpPath != "" {
		if err := writePDFImportFixture(dumpPath, problems); err != nil {
			return nil, nil, "", 0, fmt.Errorf("failed to write PDF_IMPORT_DUMP_PATH %s: %w", dumpPath, err)
		}
	}

	return problems, curriculumGrades, testSubtype, examID, nil
}

// loadPDFImportFixture reads []models.Problem JSON saved from a prior extraction run.
func loadPDFImportFixture(path string) ([]models.Problem, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read PDF_IMPORT_FIXTURE %s: %w", path, err)
	}
	var problems []models.Problem
	if err := json.Unmarshal(data, &problems); err != nil {
		return nil, fmt.Errorf("parse PDF_IMPORT_FIXTURE %s: %w", path, err)
	}
	if len(problems) == 0 {
		return nil, fmt.Errorf("PDF_IMPORT_FIXTURE %s contains no problems", path)
	}
	return problems, nil
}

func writePDFImportFixture(path string, problems []models.Problem) error {
	data, err := json.MarshalIndent(problems, "", "  ")
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	return os.WriteFile(path, data, 0o644)
}

func loadPDFImportCreatedFixture(path string) (*pdfImportCreatedFixture, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read PDF_IMPORT_CREATED_FIXTURE %s: %w", path, err)
	}
	var fixture pdfImportCreatedFixture
	if err := json.Unmarshal(data, &fixture); err != nil {
		return nil, fmt.Errorf("parse PDF_IMPORT_CREATED_FIXTURE %s: %w", path, err)
	}
	if fixture.TestID == 0 {
		return nil, fmt.Errorf("PDF_IMPORT_CREATED_FIXTURE %s missing test_id", path)
	}
	if len(fixture.ProblemIDs) == 0 {
		return nil, fmt.Errorf("PDF_IMPORT_CREATED_FIXTURE %s contains no problem_ids", path)
	}
	return &fixture, nil
}

func writePDFImportCreatedFixture(path string, fixture pdfImportCreatedFixture) error {
	data, err := json.MarshalIndent(fixture, "", "  ")
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	return os.WriteFile(path, data, 0o644)
}

// marshalPDFImportBatchBody builds the POST resources/problems/batch JSON for PDF import.
func marshalPDFImportBatchBody(problems []models.Problem, curriculumGrades []models.CurriculumGrade) ([]byte, error) {
	items := make([]map[string]any, len(problems))
	for i, p := range problems {
		difficulty := pdfImportDefaultDifficulty
		items[i] = map[string]any{
			"lang_code":         pdfImportLangCode,
			"type":              p.Type,
			"subtype":           p.Subtype,
			"type_params":       models.ProbTypeParams{TestIds: []int{}},
			"curriculum_grades": curriculumGrades,
			"difficulty_level":  difficulty,
			"cms_status_id":     constants.StatusDraft,
			"meta_data":         p.MetaData,
		}
	}
	return json.Marshal(map[string]any{"problems": items})
}

func (h *PdfImportHandler) createDraftTestFromProblems(created []models.Problem, curriculumGrades []models.CurriculumGrade, testSubtype string, examID int8) (*models.Test, error) {
	if h.testsHandler == nil {
		return nil, fmt.Errorf("tests handler is not configured")
	}
	if len(created) == 0 {
		return nil, fmt.Errorf("no created problems to add to test")
	}

	sectionsBySubtype := make(map[string][]models.ResProblem)
	for _, p := range created {
		subtype := p.Subtype
		if subtype == "" {
			subtype = "mcq_single_answer"
		}
		sectionsBySubtype[subtype] = append(sectionsBySubtype[subtype], models.ResProblem{ID: p.ID})
	}

	sections := make([]models.ResSection, 0, len(sectionsBySubtype))
	for subtype, resProblems := range sectionsBySubtype {
		sections = append(sections, models.ResSection{
			Type: subtype,
			Compulsory: models.ResCompulsory{
				Problems: resProblems,
			},
		})
	}

	now := time.Now()
	testObj := models.Test{
		Name: []models.ResName{
			{LangCode: pdfImportLangCode, Resource: fmt.Sprintf("PDF Import Draft (%s)", now.Format("2006-01-02 15:04"))},
		},
		Code:             fmt.Sprintf("PDFIMP-%d", now.Unix()),
		Type:             "test",
		Subtype:          testSubtype,
		ExamIDs:          []int8{examID},
		CurriculumGrades: curriculumGrades,
		TypeParams: models.ResTypeParams{
			Subjects: []models.ResSubject{
				{
					SubjectID: 0,
					Sections:  sections,
				},
			},
		},
		StatusID: constants.StatusDraft,
	}

	return h.testsHandler.testsService.AddObject(testObj, testsKey, resourcesEndPoint)
}

// ImportTestReview renders the per-question review UI for a draft test created from PDF import.
func (h *PdfImportHandler) ImportTestReview(responseWriter http.ResponseWriter, request *http.Request) {
	testIDStr := request.URL.Query().Get("test_id")
	if testIDStr == "" {
		http.Error(responseWriter, "test_id is required", http.StatusBadRequest)
		return
	}

	// Fetch the test once. (curriculum_id is only required for the test-problems API below.)
	testPtr, code, err := h.testsHandler.getTest(responseWriter, requestWithQuery(request, map[string]string{"id": testIDStr}))
	if err != nil {
		http.Error(responseWriter, err.Error(), code)
		return
	}

	if testPtr.StatusID != constants.StatusDraft {
		http.Error(responseWriter, "only draft tests can be reviewed for import", http.StatusBadRequest)
		return
	}

	if len(testPtr.CurriculumGrades) == 0 {
		http.Error(responseWriter, "test has no curriculum/grade; cannot load imported questions", http.StatusBadRequest)
		return
	}
	cg := testPtr.CurriculumGrades[0]
	curriculumID := utils.IntToString(cg.CurriculumID)
	gradeID := utils.IntToString(cg.GradeID)

	problemsReq := requestWithQuery(request, map[string]string{
		"id":                      testIDStr,
		QUERY_PARAM_CURRICULUM_ID: curriculumID,
		"grade_id":                gradeID,
	})
	problems := h.testsHandler.getTestProblems(responseWriter, problemsReq)
	if problems == nil {
		return
	}

	data := dto.ImportTestReviewData{
		TestPtr:  testPtr,
		Problems: *problems,
	}
	if len(testPtr.CurriculumGrades) > 0 {
		data.CurriculumID = testPtr.CurriculumGrades[0].CurriculumID
		data.GradeID = testPtr.CurriculumGrades[0].GradeID
	}

	setHtmxReplaceURL(responseWriter, request)

	views.ExecuteTemplates(responseWriter, data, template.FuncMap{
		"toJson":    utils.ToJson,
		"getName":   getTestName,
		"dict":      utils.Dict,
		"add":       utils.Add,
		"joinInt16": utils.JoinInt16,
	}, baseTemplate, importTestReviewTemplate, problemTypeOptionsTemplate,
		editorTemplate, problemAnswerNumericalTemplate, inputTagsTemplate)
}

// ImportTestReviewContinue validates reviewed questions and opens test composition (edit draft test).
// func (h *PdfImportHandler) ImportTestReviewContinue(responseWriter http.ResponseWriter, request *http.Request) {
// 	testIDStr := request.URL.Query().Get("test_id")
// 	if testIDStr == "" {
// 		http.Error(responseWriter, "test_id is required", http.StatusBadRequest)
// 		return
// 	}

// 	// Fetch the test once. (curriculum_id is only required for the test-problems API below.)
// 	testPtr, code, err := h.testsHandler.getTest(responseWriter, requestWithQuery(request, map[string]string{"id": testIDStr}))
// 	if err != nil {
// 		http.Error(responseWriter, err.Error(), code)
// 		return
// 	}

// 	if len(testPtr.CurriculumGrades) == 0 {
// 		http.Error(responseWriter, "test has no curriculum/grade; cannot validate imported questions", http.StatusBadRequest)
// 		return
// 	}
// 	cg := testPtr.CurriculumGrades[0]
// 	curriculumID := utils.IntToString(cg.CurriculumID)
// 	gradeID := utils.IntToString(cg.GradeID)

// 	problemsReq := requestWithQuery(request, map[string]string{
// 		"id":                      testIDStr,
// 		QUERY_PARAM_CURRICULUM_ID: curriculumID,
// 		"grade_id":                gradeID,
// 	})
// 	problems := h.testsHandler.getTestProblems(responseWriter, problemsReq)
// 	if problems == nil {
// 		return
// 	}

// 	ordered := orderProblemsForTest(testPtr, *problems)
// 	var missing []string
// 	for i, p := range ordered {
// 		if p.TopicID == 0 || p.ChapterID == 0 {
// 			missing = append(missing, fmt.Sprintf("Q%d", i+1))
// 		}
// 	}
// 	if len(missing) > 0 {
// 		http.Error(responseWriter,
// 			fmt.Sprintf("Assign chapter and topic for: %s", strings.Join(missing, ", ")),
// 			http.StatusBadRequest)
// 		return
// 	}

// 	redirectURL := fmt.Sprintf("/tests/edit-test?id=%s&%s=%s&grade_id=%s",
// 		testIDStr, QUERY_PARAM_CURRICULUM_ID, curriculumID, gradeID)
// 	responseWriter.Header().Set("HX-Redirect", redirectURL)
// }

// func orderProblemsForTest(testPtr *models.Test, problems []*models.Problem) []*models.Problem {
// 	byID := make(map[int]*models.Problem, len(problems))
// 	for _, p := range problems {
// 		if p == nil || p.StatusID == constants.StatusArchived {
// 			continue
// 		}
// 		byID[p.ID] = p
// 	}

// 	var ordered []*models.Problem
// 	seen := make(map[int]bool)
// 	appendID := func(id int) {
// 		if id == 0 || seen[id] {
// 			return
// 		}
// 		if p, ok := byID[id]; ok {
// 			seen[id] = true
// 			ordered = append(ordered, p)
// 		}
// 	}

// 	for _, subj := range testPtr.TypeParams.Subjects {
// 		for _, sec := range subj.Sections {
// 			for _, rp := range sec.Compulsory.Problems {
// 				appendID(rp.ID)
// 			}
// 			if sec.Optional != nil {
// 				for _, rp := range sec.Optional.Problems {
// 					appendID(rp.ID)
// 				}
// 			}
// 		}
// 	}

// 	for _, p := range problems {
// 		if p == nil || p.StatusID == constants.StatusArchived || seen[p.ID] {
// 			continue
// 		}
// 		ordered = append(ordered, p)
// 	}
// 	return ordered
// }

func setHtmxReplaceURL(w http.ResponseWriter, r *http.Request) {
	replaceURL := r.URL.Path
	if r.URL.RawQuery != "" {
		replaceURL += "?" + r.URL.RawQuery
	}
	w.Header().Set("HX-Replace-Url", replaceURL)
}

func requestWithQuery(r *http.Request, params map[string]string) *http.Request {
	q := r.URL.Query()
	for k, v := range params {
		if v != "" {
			q.Set(k, v)
		}
	}
	u := *r.URL
	u.RawQuery = q.Encode()
	clone := *r
	clone.URL = &u
	return &clone
}
