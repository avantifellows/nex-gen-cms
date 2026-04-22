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
const geminiBaseURL = "https://generativelanguage.googleapis.com/v1beta/models/"

// ExtractedQuestion holds a single question parsed from the PDF.
type ExtractedQuestion struct {
	Number  int      `json:"question_number"`
	Text    string   `json:"question_text"`
	Options []string `json:"options"`
	Type    string   `json:"question_type"`
}

type pdfImportData struct {
	dto.HomeData
	Questions   []ExtractedQuestion
	Error       string
	RawResponse string
	Processed   bool
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

	apiKey := config.GetEnv("GEMINI_API_KEY", "")
	if apiKey == "" {
		renderPdfImportPage(w, pdfImportData{Error: "GEMINI_API_KEY is not set in .env", Processed: true})
		return
	}

	questions, rawResp, err := callGemini(pdfBytes, apiKey)
	if err != nil {
		renderPdfImportPage(w, pdfImportData{Error: err.Error(), RawResponse: rawResp, Processed: true})
		return
	}

	renderPdfImportPage(w, pdfImportData{Questions: questions, RawResponse: rawResp, Processed: true})
}

func renderPdfImportPage(w http.ResponseWriter, data pdfImportData) {
	views.ExecuteTemplates(w, data, nil, baseTemplate, pdfImportTemplate)
}

func callGemini(pdfBytes []byte, apiKey string) ([]ExtractedQuestion, string, error) {
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

	reqBody := map[string]any{
		"contents": []map[string]any{
			{
				"parts": []map[string]any{
					{
						"inline_data": map[string]any{
							"mime_type": "application/pdf",
							"data":      pdfBase64,
						},
					},
					{
						"text": prompt,
					},
				},
			},
		},
		"generationConfig": map[string]any{
			"temperature": 0.1,
		},
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, "", fmt.Errorf("marshal error: %v", err)
	}

	// Force IPv4 to avoid i/o timeout on IPv6-only routes.
	// 5-minute timeout: gemini-2.5-flash can be slow on large PDFs.
	client := &http.Client{
		Timeout: 300 * time.Second,
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				return (&net.Dialer{Timeout: 30 * time.Second}).DialContext(ctx, "tcp4", addr)
			},
		},
	}

	model := config.GetEnv("GEMINI_MODEL", "gemini-2.0-flash")
	url := geminiBaseURL + model + ":generateContent?key=" + apiKey
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return nil, "", fmt.Errorf("creating request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("Gemini API call failed: %v", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("reading Gemini response: %v", err)
	}
	rawResp := string(respBytes)

	if resp.StatusCode != http.StatusOK {
		return nil, rawResp, fmt.Errorf("Gemini returned status %d", resp.StatusCode)
	}

	var geminiResp struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}

	if err := json.Unmarshal(respBytes, &geminiResp); err != nil {
		return nil, rawResp, fmt.Errorf("parsing Gemini response: %v", err)
	}
	if geminiResp.Error != nil {
		return nil, rawResp, fmt.Errorf("Gemini API error: %s", geminiResp.Error.Message)
	}
	if len(geminiResp.Candidates) == 0 || len(geminiResp.Candidates[0].Content.Parts) == 0 {
		return nil, rawResp, fmt.Errorf("Gemini returned no content")
	}

	text := strings.TrimSpace(geminiResp.Candidates[0].Content.Parts[0].Text)

	// Strip markdown code fences if Gemini wrapped the JSON anyway.
	if strings.HasPrefix(text, "```") {
		text = strings.TrimPrefix(text, "```json")
		text = strings.TrimPrefix(text, "```")
		if idx := strings.LastIndex(text, "```"); idx != -1 {
			text = text[:idx]
		}
		text = strings.TrimSpace(text)
	}

	// Gemini inconsistently escapes LaTeX delimiters: some questions emit bare
	// \( \) \[ \] (invalid JSON) while others correctly emit \\( \\) \\[ \\].
	// Normalise by doubling any backslash not followed by a valid JSON escape char.
	text = fixInvalidJSONEscapes(text)

	var questions []ExtractedQuestion
	if err := json.Unmarshal([]byte(text), &questions); err != nil {
		return nil, rawResp, fmt.Errorf("parsing questions JSON: %v\n\nText returned by Gemini:\n%s", err, text)
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
