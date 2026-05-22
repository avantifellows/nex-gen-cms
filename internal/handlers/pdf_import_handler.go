package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/avantifellows/nex-gen-cms/config"
	"github.com/avantifellows/nex-gen-cms/internal/pdfimport"
)

// PdfImportHandler handles HTTP endpoints for PDF question import.
type PdfImportHandler struct{}

func NewPdfImportHandler() *PdfImportHandler {
	return &PdfImportHandler{}
}

// ExtractQuestionsFromPDF parses an uploaded PDF and logs extracted questions (add-test modal flow).
func (h *PdfImportHandler) ExtractQuestionsFromPDF(w http.ResponseWriter, r *http.Request) {
	questions, rawResp, err := extractQuestionsFromRequest(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	logExtractedQuestions(questions)
	if rawResp != "" {
		log.Printf("pdf import: raw LLM response length=%d bytes", len(rawResp))
	}

	w.WriteHeader(http.StatusNoContent)
}

func extractQuestionsFromRequest(r *http.Request) ([]pdfimport.ExtractedQuestion, string, error) {
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

	apiKey := config.GetEnv("OPENROUTER_API_KEY", "")
	return pdfimport.ExtractQuestions(pdfBytes, apiKey)
}

func logExtractedQuestions(questions []pdfimport.ExtractedQuestion) {
	log.Printf("pdf import: extracted %d question(s)", len(questions))
	for _, q := range questions {
		payload, err := json.Marshal(q)
		if err != nil {
			log.Printf("pdf import: question %d: marshal error: %v", q.Number, err)
			continue
		}
		log.Printf("pdf import: question %d: %s", q.Number, string(payload))
	}
}
