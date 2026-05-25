package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/avantifellows/nex-gen-cms/config"
	"github.com/avantifellows/nex-gen-cms/internal/models"
	"github.com/avantifellows/nex-gen-cms/internal/pdfimport"
)

// PdfImportHandler handles HTTP endpoints for PDF question import.
type PdfImportHandler struct{}

func NewPdfImportHandler() *PdfImportHandler {
	return &PdfImportHandler{}
}

// ExtractQuestionsFromPDF parses an uploaded PDF, streams progress as SSE, then logs results.
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

	problems, _, err := extractProblemsFromRequest(r, onProgress)
	if err != nil {
		writeSSE("error", map[string]string{"message": err.Error()})
		return
	}

	logExtractedProblems(problems)

	writeSSE("complete", map[string]any{"question_count": len(problems)})
}

func extractProblemsFromRequest(r *http.Request, onProgress pdfimport.ProgressFunc) ([]models.Problem, string, error) {
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		return nil, "", fmt.Errorf("failed to parse form: %w", err)
	}

	file, _, err := r.FormFile("pdf_file")
	if err != nil {
		return nil, "", fmt.Errorf("no PDF file provided: %w", err)
	}
	defer file.Close()

	pdfBytes, err := io.ReadAll(file)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read PDF: %w", err)
	}

	if onProgress != nil {
		onProgress(5, "PDF received, starting extraction…")
	}

	apiKey := config.GetEnv("OPENROUTER_API_KEY", "")
	return pdfimport.ExtractProblemsWithProgress(pdfBytes, apiKey, onProgress)
}

func logExtractedProblems(problems []models.Problem) {
	log.Printf("pdf import: extracted %d problem(s)", len(problems))
	for i, p := range problems {
		payload, err := json.Marshal(p)
		if err != nil {
			log.Printf("pdf import: problem %d: marshal error: %v", i+1, err)
			continue
		}
		log.Printf("pdf import: problem %d (subtype=%s): %s", i+1, p.Subtype, string(payload))
	}
}
