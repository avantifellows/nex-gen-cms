package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/avantifellows/nex-gen-cms/config"
	"github.com/avantifellows/nex-gen-cms/internal/constants"
	"github.com/avantifellows/nex-gen-cms/internal/handlers/handlerutils"
	"github.com/avantifellows/nex-gen-cms/internal/models"
	"github.com/avantifellows/nex-gen-cms/internal/pdfimport"
	"github.com/avantifellows/nex-gen-cms/internal/services"
	"github.com/avantifellows/nex-gen-cms/utils"
)

const (
	pdfImportDefaultDifficulty = "medium"
	pdfImportLangCode          = "en"
)

// PdfImportHandler handles HTTP endpoints for PDF question import.
type PdfImportHandler struct {
	problemsHandler *ProblemsHandler
	testsService    *services.Service[models.Test]
}

func NewPdfImportHandler(problemsHandler *ProblemsHandler, testsService *services.Service[models.Test]) *PdfImportHandler {
	return &PdfImportHandler{problemsHandler: problemsHandler, testsService: testsService}
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
	if h.testsService == nil {
		return nil, fmt.Errorf("tests service is not configured")
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

	return h.testsService.AddObject(testObj, testsKey, resourcesEndPoint)
}
