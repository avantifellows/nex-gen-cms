package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/avantifellows/nex-gen-cms/config"
	"github.com/avantifellows/nex-gen-cms/internal/constants"
	"github.com/avantifellows/nex-gen-cms/internal/handlers/handlerutils"
	"github.com/avantifellows/nex-gen-cms/internal/models"
	"github.com/avantifellows/nex-gen-cms/internal/pdfimport"
)

const (
	pdfImportDefaultDifficulty = "medium"
	pdfImportLangCode          = "en"
)

// PdfImportHandler handles HTTP endpoints for PDF question import.
type PdfImportHandler struct {
	problemsHandler *ProblemsHandler
}

func NewPdfImportHandler(problemsHandler *ProblemsHandler) *PdfImportHandler {
	return &PdfImportHandler{problemsHandler: problemsHandler}
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

	problems, curriculumGrades, err := extractProblemsFromRequest(r, onProgress)
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
		if err := h.problemsHandler.postBatchProblems(reqBody); err != nil {
			writeSSE("error", map[string]string{"message": fmt.Sprintf("failed to save questions: %v", err)})
			return
		}
		onProgress(100, "Questions saved")
	}

	writeSSE("complete", map[string]any{"question_count": len(problems)})
}

func extractProblemsFromRequest(r *http.Request, onProgress pdfimport.ProgressFunc) ([]models.Problem, []models.CurriculumGrade, error) {
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		return nil, nil, fmt.Errorf("failed to parse form: %w", err)
	}

	curriculumGrades, err := handlerutils.ParseCurriculumGradesFromForm(r.Form)
	if err != nil {
		return nil, nil, err
	}

	file, _, err := r.FormFile("pdf_file")
	if err != nil {
		return nil, nil, fmt.Errorf("no PDF file provided: %w", err)
	}
	defer file.Close()

	pdfBytes, err := io.ReadAll(file)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read PDF: %w", err)
	}

	if onProgress != nil {
		onProgress(5, "PDF received, starting extraction…")
	}

	apiKey := config.GetEnv("OPENROUTER_API_KEY", "")
	problems, _, err := pdfimport.ExtractProblemsWithProgress(pdfBytes, apiKey, onProgress)
	if err != nil {
		return nil, nil, err
	}
	return problems, curriculumGrades, nil
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
