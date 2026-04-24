package handlers

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/avantifellows/nex-gen-cms/config"
	"github.com/avantifellows/nex-gen-cms/internal/dto"
	"github.com/avantifellows/nex-gen-cms/internal/views"
)

const pdfImportTemplate = "pdf_import.html"
const openRouterURL = "https://openrouter.ai/api/v1/chat/completions"

// ExtractedQuestion holds a single question parsed from the PDF.
type ExtractedQuestion struct {
	Number  int      `json:"question_number"`
	Text    string   `json:"question_text"`
	Options []string `json:"options"`
	Type    string   `json:"question_type"`
}

type pdfImportData struct {
	dto.HomeData
	Questions     []ExtractedQuestion
	Error         string
	RawResponse   string
	ProcessedJSON string
	Processed     bool
}

// PdfImportHandler handles PDF question extraction (POC).
type PdfImportHandler struct{}

func NewPdfImportHandler() *PdfImportHandler {
	return &PdfImportHandler{}
}

func (h *PdfImportHandler) ShowForm(w http.ResponseWriter, r *http.Request) {
	views.ExecuteTemplates(w, pdfImportData{}, nil, baseTemplate, pdfImportTemplate)
}

func (h *PdfImportHandler) ProcessPDF(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		renderPdfImportPage(w, pdfImportData{Error: "Failed to parse form: " + err.Error(), Processed: true})
		return
	}

	file, _, err := r.FormFile("pdf_file")
	if err != nil {
		renderPdfImportPage(w, pdfImportData{Error: "No PDF file provided: " + err.Error(), Processed: true})
		return
	}
	defer file.Close()

	pdfBytes, err := io.ReadAll(file)
	if err != nil {
		renderPdfImportPage(w, pdfImportData{Error: "Failed to read PDF: " + err.Error(), Processed: true})
		return
	}

	apiKey := config.GetEnv("OPENROUTER_API_KEY", "")
	if apiKey == "" {
		renderPdfImportPage(w, pdfImportData{Error: "OPENROUTER_API_KEY is not set in .env", Processed: true})
		return
	}

	questions, rawResp, err := callOpenRouter(pdfBytes, apiKey)
	if err != nil {
		renderPdfImportPage(w, pdfImportData{Error: err.Error(), RawResponse: rawResp, Processed: true})
		return
	}

	processedJSON, _ := json.MarshalIndent(questions, "", "  ")
	renderPdfImportPage(w, pdfImportData{
		Questions:     questions,
		RawResponse:   rawResp,
		ProcessedJSON: string(processedJSON),
		Processed:     true,
	})
}

func renderPdfImportPage(w http.ResponseWriter, data pdfImportData) {
	views.ExecuteTemplates(w, data, nil, baseTemplate, pdfImportTemplate)
}

func callOpenRouter(pdfBytes []byte, apiKey string) ([]ExtractedQuestion, string, error) {
	pdfBase64 := base64.StdEncoding.EncodeToString(pdfBytes)

	prompt := `You are a JEE exam paper parser. Extract every question from this PDF.

IMPORTANT — formatting rules:
- All mathematical expressions, equations, formulas, symbols, fractions, integrals, summations, limits, vectors, matrices, subscripts, superscripts etc. MUST be written in LaTeX.
- Wrap inline math in \( ... \) and display/block math in \[ ... \].
- Plain text parts of the question stay as plain text.
- Do NOT use $...$ or $$...$$ delimiters — use \(...\) and \[...\] only.

For each question return a JSON object with exactly these fields:
- question_number: integer (the question number as printed)
- question_text: full question text with LaTeX math (include any sub-parts/paragraphs)
- options: array of strings with LaTeX math — one entry per option (A, B, C, D text). Empty array [] for numerical questions that have no options.
- question_type: "mcq" if options are present, "numerical" if no options

Return ONLY a valid JSON array — no markdown fences, no explanation, no extra text.

Example output:
[
  {"question_number":1,"question_text":"If \(f(x) = x^2 + 1\), find \(f(3)\).","options":["\(9\)","\(10\)","\(12\)","None of these"],"question_type":"mcq"},
  {"question_number":2,"question_text":"Find the value of \[\int_0^1 x \, dx\]","options":[],"question_type":"numerical"}
]`

	model := config.GetEnv("OPENROUTER_MODEL", "google/gemini-2.0-flash-001")

	reqBody := map[string]any{
		"model": model,
		"messages": []map[string]any{
			{
				"role": "user",
				"content": []map[string]any{
					{
						"type": "file",
						"file": map[string]any{
							"filename":  "document.pdf",
							"file_data": "data:application/pdf;base64," + pdfBase64,
						},
					},
					{
						"type": "text",
						"text": prompt,
					},
				},
			},
		},
		"temperature": 0.1,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, "", fmt.Errorf("marshal error: %v", err)
	}

	// Force IPv4 to avoid i/o timeout on IPv6-only routes.
	// 5-minute timeout: large PDFs with multimodal models can be slow.
	client := &http.Client{
		Timeout: 300 * time.Second,
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				return (&net.Dialer{Timeout: 30 * time.Second}).DialContext(ctx, "tcp4", addr)
			},
		},
	}

	req, err := http.NewRequest(http.MethodPost, openRouterURL, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return nil, "", fmt.Errorf("creating request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := client.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("OpenRouter API call failed: %v", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("reading OpenRouter response: %v", err)
	}
	rawResp := string(respBytes)

	if resp.StatusCode != http.StatusOK {
		return nil, rawResp, fmt.Errorf("OpenRouter returned status %d", resp.StatusCode)
	}

	var openRouterResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}

	if err := json.Unmarshal(respBytes, &openRouterResp); err != nil {
		return nil, rawResp, fmt.Errorf("parsing OpenRouter response: %v", err)
	}
	if openRouterResp.Error != nil {
		return nil, rawResp, fmt.Errorf("OpenRouter API error: %s", openRouterResp.Error.Message)
	}
	if len(openRouterResp.Choices) == 0 {
		return nil, rawResp, fmt.Errorf("OpenRouter returned no choices")
	}

	text := strings.TrimSpace(openRouterResp.Choices[0].Message.Content)

	// Strip markdown code fences if the model wrapped the JSON anyway.
	if strings.HasPrefix(text, "```") {
		text = strings.TrimPrefix(text, "```json")
		text = strings.TrimPrefix(text, "```")
		if idx := strings.LastIndex(text, "```"); idx != -1 {
			text = text[:idx]
		}
		text = strings.TrimSpace(text)
	}

	// Models inconsistently escape LaTeX delimiters: some questions emit bare
	// \( \) \[ \] (invalid JSON) while others correctly emit \\( \\) \\[ \\].
	// Normalise by doubling any backslash not followed by a valid JSON escape char.
	text = fixInvalidJSONEscapes(text)

	var questions []ExtractedQuestion
	if err := json.Unmarshal([]byte(text), &questions); err != nil {
		return nil, rawResp, fmt.Errorf("parsing questions JSON: %v\n\nText returned by model:\n%s", err, text)
	}

	return questions, rawResp, nil
}

// fixInvalidJSONEscapes doubles any backslash not followed by a valid JSON
// escape character (" \ / b f n r t u). This converts LaTeX sequences like
// \( \) \[ \] \frac into \\( \\) \\[ \\] \\frac — valid JSON string content.
func fixInvalidJSONEscapes(s string) string {
	validEscape := map[byte]bool{
		'"': true, '\\': true, '/': true,
		'b': true, 'f': true, 'n': true,
		'r': true, 't': true, 'u': true,
	}
	var b strings.Builder
	b.Grow(len(s))
	i := 0
	for i < len(s) {
		if s[i] == '\\' && i+1 < len(s) {
			if validEscape[s[i+1]] {
				// Already valid — copy both bytes and skip past them.
				b.WriteByte(s[i])
				b.WriteByte(s[i+1])
				i += 2
			} else {
				// Invalid escape (e.g. \( \) \[ \]) — emit \\ and let the
				// next character be written in the following iteration.
				b.WriteByte('\\')
				b.WriteByte('\\')
				i++
			}
		} else {
			b.WriteByte(s[i])
			i++
		}
	}
	return b.String()
}
