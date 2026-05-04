package handlers

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html"
	htmltemplate "html/template"
	"io"
	"net"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/avantifellows/nex-gen-cms/config"
	"github.com/avantifellows/nex-gen-cms/internal/dto"
	"github.com/avantifellows/nex-gen-cms/internal/views"
)

const pdfImportTemplate = "pdf_import.html"
const openRouterURL = "https://openrouter.ai/api/v1/chat/completions"

var inlineOptionMarkerRegex = regexp.MustCompile(`\(\s*(?:[A-Da-d]|[0-9]{1,2}|i|ii|iii|iv|v|vi|vii|viii|ix|x)\s*\)`)

// odlFigureNumRegex matches [FIGURE_N] tokens emitted in ODL-mode LLM responses.
var odlFigureNumRegex = regexp.MustCompile(`\[FIGURE_(\d+)\]`)
var questionStartRegex = regexp.MustCompile(`(?m)(?:^|\n)\s*(?:Q(?:uestion)?\s*)?([1-9]\d{0,2})\s*[\.\-:]\s+`)
var questionStartLooseRegex = regexp.MustCompile(`(?m)(?:^|\n)\s*(?:Q(?:uestion)?\s*)?([1-9]\d{1,2})\s+`)
var questionStartInlineRegex = regexp.MustCompile(`(?:^|[\n\r\t ])(?:Q(?:uestion)?\s*)?([1-9]\d{1,2})\s*[\.\-:]\s+`)

// ExtractedQuestion holds a single question parsed from the PDF.
type ExtractedQuestion struct {
	Number            int               `json:"question_number"`
	Text              string            `json:"question_text"`
	Options           []string          `json:"options"`
	Type              string            `json:"question_type"`
	HasFigure         bool              `json:"has_figure"`
	FigureDescription string            `json:"figure_description"`
	FigureImagePNG    []byte            `json:"-"` // cropped figure PNG; populated in ODL mode
	ProcessedText     htmltemplate.HTML `json:"-"` // HTML-safe text with [FIGURE] replaced inline
}

type pdfImportData struct {
	dto.HomeData
	Questions      []ExtractedQuestion
	Error          string
	RawResponse    string
	ProcessedJSON  string
	Processed      bool
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

	questions, rawResp, callErr := callOpenRouterHybrid(pdfBytes, apiKey)
	if callErr != nil {
		renderPdfImportPage(w, pdfImportData{Error: callErr.Error(), RawResponse: rawResp, Processed: true})
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
- Every LaTeX expression must be syntactically valid and balanced (matching braces/brackets/delimiters). Do NOT output placeholders like "Math Input", "Math Error", or broken TeX.

FIGURES / GRAPHS / IMAGES:
- If a question references or depends on a figure, graph, diagram, table, or image (whether embedded in the question body or provided as a separate numbered figure), set "has_figure" to true.
- Insert the token [FIGURE] at the exact position in "question_text" where the figure appears (e.g. "Consider the graph below.\n[FIGURE]\nWhat is the slope?").
- In "figure_description", write a thorough plain-English description of everything visible in the figure: axis labels and ranges, curve shapes, key coordinates, shaded regions, arrows, table values, etc. Use LaTeX for any mathematical quantities within the description. If the question has no figure, set "has_figure" to false and "figure_description" to "".

For each question return a JSON object with exactly these fields:
- question_number: integer (the question number as printed)
- question_text: full question text with LaTeX math (include any sub-parts/paragraphs; insert [FIGURE] placeholder where applicable)
- options: array of strings with LaTeX math — one entry per option (A, B, C, D text). Empty array [] for numerical questions that have no options.
- question_type: "mcq" if options are present, "numerical" if no options
- has_figure: boolean — true if the question contains or references a figure/graph/image/diagram/table
- figure_description: string — detailed description of the figure (empty string "" if has_figure is false)

Return ONLY a valid JSON array — no markdown fences, no explanation, no extra text.

Example output:
[
  {"question_number":1,"question_text":"If \(f(x) = x^2 + 1\), find \(f(3)\).","options":["\(9\)","\(10\)","\(12\)","None of these"],"question_type":"mcq","has_figure":false,"figure_description":""},
  {"question_number":2,"question_text":"Find the value of \[\int_0^1 x \, dx\]","options":[],"question_type":"numerical","has_figure":false,"figure_description":""},
  {"question_number":3,"question_text":"The velocity-time graph of a particle is shown below.\n[FIGURE]\nThe acceleration of the particle at \(t = 2\) s is:","options":["zero","\(2 \, \text{m/s}^2\)","\(4 \, \text{m/s}^2\)","\(8 \, \text{m/s}^2\)"],"question_type":"mcq","has_figure":true,"figure_description":"A velocity-time (v-t) graph with the horizontal axis labelled 't (s)' ranging from 0 to 6 and the vertical axis labelled 'v (m/s)' ranging from 0 to 12. A straight line rises from the origin \((0,0)\) to the point \((3, 6)\), then remains horizontal from \((3, 6)\) to \((6, 6)\)."}
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

	postProcessQuestions(questions)
	return questions, rawResp, nil
}

// callOpenRouterODL is the two-stage ODL pipeline:
//
//  1. Rasterise the PDF to per-page PNGs (for figure cropping).
//  2. Run OpenDataLoader locally to get text elements + picture bounding boxes.
//  3. Build a compact prompt: extracted text with [FIGURE_N] markers + cropped
//     figure images as image_url blocks.
//  4. Send to OpenRouter and parse the response.
//  5. Assign the correct cropped PNG to each parsed question.
func callOpenRouterODL(pdfBytes []byte, apiKey string) ([]ExtractedQuestion, string, error) {
	// --- Step 1: write PDF to disk (ODL needs a file path) ---
	tmpPDF, err := os.CreateTemp("", "odl-input-*.pdf")
	if err != nil {
		return nil, "", fmt.Errorf("creating temp PDF: %w", err)
	}
	defer os.Remove(tmpPDF.Name())
	if _, err := tmpPDF.Write(pdfBytes); err != nil {
		tmpPDF.Close()
		return nil, "", fmt.Errorf("writing temp PDF: %w", err)
	}
	tmpPDF.Close()

	// --- Step 2: rasterise PDF pages ---
	pageImages, err := pdfToImages(pdfBytes, odlRasterDPI)
	if err != nil {
		return nil, "", fmt.Errorf("rasterising PDF: %w", err)
	}

	// --- Step 3: run ODL extraction ---
	elements, err := extractWithODL(tmpPDF.Name())
	if err != nil {
		return nil, "", fmt.Errorf("OpenDataLoader extraction: %w", err)
	}
	elements = sortODLElementsReadingOrder(elements)

	// --- Step 4: build content string + crop figure images ---
	figureCounter := 0
	figureCrops := map[int][]byte{} // figureNum → cropped PNG bytes

	var contentBuf strings.Builder
	currentPage := 0

	for _, el := range elements {
		if el.PageNumber != currentPage {
			currentPage = el.PageNumber
			fmt.Fprintf(&contentBuf, "\n\n--- PAGE %d ---\n", currentPage)
		}

		switch strings.ToLower(el.Type) {
		case "paragraph", "heading", "caption", "list", "list item", "table":
			if el.Content != "" {
				contentBuf.WriteString(el.Content)
				contentBuf.WriteByte('\n')
			}
		case "formula":
			if el.Content != "" {
				contentBuf.WriteString(el.Content)
				contentBuf.WriteByte('\n')
			}
		case "picture", "image", "figure":
			figureCounter++
			n := figureCounter
			fmt.Fprintf(&contentBuf, "[FIGURE_%d]\n", n)

			pageIdx := el.PageNumber - 1
			if pageIdx >= 0 && pageIdx < len(pageImages) {
				rect, rectErr := odlBboxToPixelRect(el.BoundingBox, pageImages[pageIdx])
				if rectErr == nil && rect.Dx() > 0 && rect.Dy() > 0 {
					if cropBytes, cropErr := cropPageToPNG(pageImages[pageIdx], rect); cropErr == nil {
						figureCrops[n] = cropBytes
					}
				}
			}
		}
	}

	extractedText := contentBuf.String()

	// --- Step 5: split into deterministic per-question chunks ---
	chunks := splitODLQuestionChunks(extractedText)
	if len(chunks) == 0 {
		return nil, "", fmt.Errorf("could not detect question boundaries in ODL extracted text")
	}
	questions := make([]ExtractedQuestion, 0, len(chunks))
	rawParts := make([]string, 0, len(chunks))

	for _, ch := range chunks {
		content := []map[string]any{
			{
				"type": "text",
				"text": buildODLSingleQuestionPrompt(ch.Number, ch.Text),
			},
		}
		for _, figNum := range ch.FigureNums {
			cropBytes, ok := figureCrops[figNum]
			if !ok {
				continue
			}
			b64 := base64.StdEncoding.EncodeToString(cropBytes)
			content = append(content,
				map[string]any{
					"type": "text",
					"text": fmt.Sprintf("Figure %d (referenced as [FIGURE_%d] in the question text):", figNum, figNum),
				},
				map[string]any{
					"type":      "image_url",
					"image_url": map[string]any{"url": "data:image/png;base64," + b64},
				},
			)
		}

		parsed, rawResp, err := callOpenRouterWithMultimodalContent(content, apiKey)
		if err != nil {
			return nil, rawResp, fmt.Errorf("question %d extraction failed: %w", ch.Number, err)
		}
		rawParts = append(rawParts, fmt.Sprintf("Q%d:\n%s", ch.Number, rawResp))
		if len(parsed) == 0 {
			continue
		}
		q := parsed[0]
		// Keep deterministic boundary from source chunk.
		q.Number = ch.Number
		questions = append(questions, q)
	}
	rawResp := strings.Join(rawParts, "\n\n-----\n\n")

	// --- Step 6: assign figure crops to questions, normalise [FIGURE_N] → [FIGURE] ---
	for i := range questions {
		q := &questions[i]
		if matches := odlFigureNumRegex.FindStringSubmatch(q.Text); len(matches) > 1 {
			if n, convErr := strconv.Atoi(matches[1]); convErr == nil {
				q.FigureImagePNG = figureCrops[n]
			}
		}
		// Replace all [FIGURE_N] with [FIGURE] so buildProcessedText works uniformly.
		q.Text = odlFigureNumRegex.ReplaceAllString(q.Text, "[FIGURE]")
	}

	postProcessQuestions(questions)
	return questions, rawResp, nil
}

// callOpenRouterHybrid keeps direct-PDF text quality for question/options, then
// overlays ODL-derived figure crops by question number only when the direct
// model already set has_figure, so ODL noise (fraction bars, symbol fragments)
// is not shown on questions the model did not flag as having a figure.
func callOpenRouterHybrid(pdfBytes []byte, apiKey string) ([]ExtractedQuestion, string, error) {
	directQuestions, directRaw, err := callOpenRouter(pdfBytes, apiKey)
	if err != nil {
		return nil, directRaw, err
	}

	odlQuestions, odlRaw, err := callOpenRouterODL(pdfBytes, apiKey)
	if err != nil {
		// Keep direct output usable even if ODL figure extraction fails.
		return directQuestions, fmt.Sprintf("DIRECT RESPONSE:\n%s\n\n-----\n\nODL RESPONSE (failed):\n%s", directRaw, odlRaw), nil
	}

	figureByQuestion := make(map[int][]byte, len(odlQuestions))
	for _, q := range odlQuestions {
		if len(q.FigureImagePNG) == 0 {
			continue
		}
		figureByQuestion[q.Number] = q.FigureImagePNG
	}

	for i := range directQuestions {
		q := &directQuestions[i]
		if !q.HasFigure {
			continue
		}
		if fig, ok := figureByQuestion[q.Number]; ok && len(fig) > 0 {
			q.FigureImagePNG = fig
		}
	}

	// Rebuild ProcessedText so attached image crops render in UI.
	postProcessQuestions(directQuestions)
	return directQuestions, fmt.Sprintf("DIRECT RESPONSE:\n%s\n\n-----\n\nODL RESPONSE:\n%s", directRaw, odlRaw), nil
}

func buildODLSingleQuestionPrompt(questionNumber int, chunkText string) string {
	return fmt.Sprintf(`You are a JEE exam paper parser. Parse exactly ONE question from the extracted source below.

Rules:
- The question number MUST be %d.
- Do not merge with previous/next question.
- Keep all maths in LaTeX using \( ... \) or \[ ... \].
- Keep options only inside "options" array (not in question_text).
- Preserve any [FIGURE_N] marker inside question_text where that figure appears.

Return ONLY valid JSON (single object or one-item array) with these fields:
- question_number (integer)
- question_text (string)
- options (array of strings)
- question_type ("mcq" or "numerical")
- has_figure (boolean)
- figure_description (string)

SOURCE CHUNK:
%s`, questionNumber, chunkText)
}

func callOpenRouterWithMultimodalContent(content []map[string]any, apiKey string) ([]ExtractedQuestion, string, error) {
	model := config.GetEnv("OPENROUTER_MODEL", "google/gemini-2.0-flash-001")
	reqBody := map[string]any{
		"model": model,
		"messages": []map[string]any{
			{"role": "user", "content": content},
		},
		"temperature": 0.1,
	}
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, "", fmt.Errorf("marshal error: %v", err)
	}
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
	if strings.HasPrefix(text, "```") {
		text = strings.TrimPrefix(text, "```json")
		text = strings.TrimPrefix(text, "```")
		if idx := strings.LastIndex(text, "```"); idx != -1 {
			text = text[:idx]
		}
		text = strings.TrimSpace(text)
	}
	text = fixInvalidJSONEscapes(text)
	questions, err := parseQuestionsJSON(text)
	if err != nil {
		return nil, rawResp, fmt.Errorf("parsing questions JSON: %v\n\nText returned by model:\n%s", err, text)
	}
	return questions, rawResp, nil
}

func parseQuestionsJSON(text string) ([]ExtractedQuestion, error) {
	var arr []ExtractedQuestion
	if err := json.Unmarshal([]byte(text), &arr); err == nil {
		return arr, nil
	}
	var one ExtractedQuestion
	if err := json.Unmarshal([]byte(text), &one); err == nil {
		return []ExtractedQuestion{one}, nil
	}
	return nil, fmt.Errorf("response is neither question array nor single question object")
}

type odlQuestionChunk struct {
	Number    int
	Text      string
	FigureNums []int
}

func splitODLQuestionChunks(extractedText string) []odlQuestionChunk {
	matches := questionStartRegex.FindAllStringSubmatchIndex(extractedText, -1)
	if len(matches) == 0 {
		matches = questionStartLooseRegex.FindAllStringSubmatchIndex(extractedText, -1)
	}
	if len(matches) == 0 {
		matches = questionStartInlineRegex.FindAllStringSubmatchIndex(extractedText, -1)
	}
	if len(matches) == 0 {
		return nil
	}

	// If the source uses two/three-digit question numbers (e.g. 34..46), avoid
	// treating option markers like 1..4 as question starts.
	minAcceptedQuestionNumber := 1
	maxDetectedQuestionNumber := 0
	for _, m := range matches {
		if len(m) < 4 {
			continue
		}
		if n, err := strconv.Atoi(extractedText[m[2]:m[3]]); err == nil && n > maxDetectedQuestionNumber {
			maxDetectedQuestionNumber = n
		}
	}
	if maxDetectedQuestionNumber >= 20 {
		minAcceptedQuestionNumber = 10
	}

	chunks := make([]odlQuestionChunk, 0, len(matches))
	seen := map[int]bool{}
	lastNum := -1
	for i, m := range matches {
		if len(m) < 4 {
			continue
		}
		numStart, numEnd := m[2], m[3]
		start := m[0]
		end := len(extractedText)
		if i+1 < len(matches) {
			end = matches[i+1][0]
		}
		n, err := strconv.Atoi(extractedText[numStart:numEnd])
		if err != nil {
			continue
		}
		// Heuristics to ignore spurious number matches from body text/math.
		if n < minAcceptedQuestionNumber || n > 500 {
			continue
		}
		if seen[n] {
			continue
		}
		if lastNum > 0 && n+3 < lastNum {
			continue
		}
		chunkText := strings.TrimSpace(extractedText[start:end])
		if chunkText == "" {
			continue
		}
		seen[n] = true
		lastNum = n
		chunks = append(chunks, odlQuestionChunk{
			Number:    n,
			Text:      chunkText,
			FigureNums: extractFigureNums(chunkText),
		})
	}
	return chunks
}

func extractFigureNums(s string) []int {
	matches := odlFigureNumRegex.FindAllStringSubmatch(s, -1)
	if len(matches) == 0 {
		return nil
	}
	seen := map[int]bool{}
	out := make([]int, 0, len(matches))
	for _, m := range matches {
		if len(m) < 2 {
			continue
		}
		n, err := strconv.Atoi(m[1])
		if err != nil || seen[n] {
			continue
		}
		seen[n] = true
		out = append(out, n)
	}
	return out
}

// postProcessQuestions builds ProcessedText with the [FIGURE] placeholder with the [FIGURE] placeholder
// replaced inline by a figure box (image in ODL mode, text description in direct mode).
func postProcessQuestions(questions []ExtractedQuestion) {
	for i := range questions {
		q := &questions[i]
		q.Text = strings.ReplaceAll(q.Text, `\n`, "\n")
		for j := range q.Options {
			q.Options[j] = strings.ReplaceAll(q.Options[j], `\n`, "\n")
		}
		q.Text = trimDuplicatedOptionsFromQuestionText(q.Text, q.Options)
		q.ProcessedText = buildProcessedText(q)
	}
}

// buildProcessedText returns HTML-safe question text with any [FIGURE] token
// replaced by either a real cropped figure image (ODL mode) or an LLM text
// description box (direct PDF mode).
func buildProcessedText(q *ExtractedQuestion) htmltemplate.HTML {
	figureBox := ""
	if q.HasFigure {
		if q.FigureImagePNG != nil {
			// ODL mode: embed the actual cropped figure image.
			b64 := base64.StdEncoding.EncodeToString(q.FigureImagePNG)
			figureBox = fmt.Sprintf(
				`<div class="my-3 rounded-md border border-gray-200 overflow-hidden">`+
					`<div class="bg-gray-50 p-2"><img src="data:image/png;base64,%s" class="max-w-full h-auto" /></div>`+
					`</div>`,
				b64,
			)
		} else if q.FigureDescription != "" {
			// Direct PDF mode: show LLM text description.
			figureBox = fmt.Sprintf(
				`<div class="my-3 rounded-md border border-amber-300 bg-amber-50 px-3 py-2 text-xs text-amber-900">`+
					`<p class="mb-1 font-semibold text-amber-700">&#128202; Figure / Graph description</p>`+
					`<p class="leading-relaxed whitespace-pre-wrap">%s</p></div>`,
				html.EscapeString(q.FigureDescription),
			)
		}
	}

	// escapeText converts raw question text to browser-safe HTML:
	// HTML-escapes special chars and turns literal newlines into <br> so that
	// multi-line question text (options on separate lines etc.) renders correctly.
	escapeText := func(s string) string {
		return strings.ReplaceAll(html.EscapeString(s), "\n", "<br>")
	}

	// [FIGURE] is the normalised placeholder (ODL mode normalises [FIGURE_N] to
	// [FIGURE] before reaching here; direct PDF mode emits [FIGURE] directly).
	var sb strings.Builder
	if q.HasFigure && strings.Contains(q.Text, "[FIGURE]") {
		parts := strings.Split(q.Text, "[FIGURE]")
		for i, part := range parts {
			sb.WriteString(escapeText(part))
			if i < len(parts)-1 {
				sb.WriteString(figureBox)
			}
		}
	} else {
		sb.WriteString(escapeText(q.Text))
		// No inline [FIGURE] marker: append the figure box at the end.
		if q.HasFigure && figureBox != "" {
			sb.WriteString(figureBox)
		}
	}
	return htmltemplate.HTML(sb.String())
}

// trimDuplicatedOptionsFromQuestionText removes a trailing block of option lines
// from question_text only when those lines actually match entries from options.
// This keeps genuine question content intact while handling model duplication.
func trimDuplicatedOptionsFromQuestionText(text string, options []string) string {
	if strings.TrimSpace(text) == "" || len(options) < 2 {
		return text
	}

	// Strategy 1: line-by-line trailing option block.
	if cleaned, ok := trimTrailingOptionLines(text, options); ok {
		return cleaned
	}
	// Strategy 2: inline packed options, e.g. "(1) ... (2) ... (3) ... (4) ...".
	if cleaned, ok := trimInlinePackedOptions(text, options); ok {
		return cleaned
	}
	return text
}

func trimTrailingOptionLines(text string, options []string) (string, bool) {
	lines := strings.Split(text, "\n")
	remaining := len(options)
	optionCounts := map[string]int{}
	for _, opt := range options {
		optionCounts[normalizeOptionText(stripOptionPrefix(opt))]++
	}

	end := len(lines)
	removed := 0
	for end > 0 && remaining > 0 {
		line := strings.TrimSpace(lines[end-1])
		if line == "" {
			end--
			continue
		}
		lineOpt, ok := parseOptionLine(line)
		if !ok {
			break
		}
		norm := normalizeOptionText(stripOptionPrefix(lineOpt))
		if optionCounts[norm] <= 0 {
			break
		}
		optionCounts[norm]--
		remaining--
		removed++
		end--
	}

	if removed >= 2 {
		return strings.TrimSpace(strings.Join(lines[:end], "\n")), true
	}
	return text, false
}

func trimInlinePackedOptions(text string, options []string) (string, bool) {
	matches := inlineOptionMarkerRegex.FindAllStringIndex(text, -1)
	if len(matches) < 2 {
		return text, false
	}

	start := matches[0][0]
	if start <= 0 {
		return text, false
	}

	segments := make([]string, 0, len(matches))
	for i := 0; i < len(matches); i++ {
		bodyStart := matches[i][1]
		bodyEnd := len(text)
		if i+1 < len(matches) {
			bodyEnd = matches[i+1][0]
		}
		body := strings.TrimSpace(text[bodyStart:bodyEnd])
		if body == "" {
			return text, false
		}
		segments = append(segments, body)
	}

	if !optionsMatch(segments, options) {
		return text, false
	}
	return strings.TrimSpace(text[:start]), true
}

func optionsMatch(candidates []string, options []string) bool {
	if len(candidates) != len(options) {
		return false
	}
	counts := map[string]int{}
	for _, opt := range options {
		counts[normalizeOptionText(stripOptionPrefix(opt))]++
	}
	for _, cand := range candidates {
		key := normalizeOptionText(stripOptionPrefix(cand))
		if counts[key] <= 0 {
			return false
		}
		counts[key]--
	}
	return true
}

// parseOptionLine extracts option content from common MCQ prefixes.
// Supported examples: "(A) x", "A) x", "B. x", "C: x", "(1) x", "2) x", "3. x".
func parseOptionLine(line string) (string, bool) {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return "", false
	}

	// Pattern: "(marker) text", where marker can be A-D / 1-99 / i-vii.
	if len(trimmed) >= 4 && trimmed[0] == '(' {
		if closeIdx := strings.IndexByte(trimmed, ')'); closeIdx > 1 {
			marker := strings.TrimSpace(trimmed[1:closeIdx])
			if isOptionMarker(marker) {
				return strings.TrimSpace(trimmed[closeIdx+1:]), true
			}
		}
	}

	// Pattern: "marker) text" / "marker. text" / "marker: text"
	for i := 1; i < len(trimmed); i++ {
		if trimmed[i] == ')' || trimmed[i] == '.' || trimmed[i] == ':' {
			marker := strings.TrimSpace(trimmed[:i])
			if isOptionMarker(marker) {
				return strings.TrimSpace(trimmed[i+1:]), true
			}
			break
		}
	}
	return "", false
}

func isOptionLetterString(s string) bool {
	return s == "a" || s == "b" || s == "c" || s == "d"
}

func isRomanOptionString(s string) bool {
	switch s {
	case "i", "ii", "iii", "iv", "v", "vi", "vii", "viii", "ix", "x":
		return true
	default:
		return false
	}
}

func isOptionMarker(s string) bool {
	lower := strings.ToLower(strings.TrimSpace(s))
	if lower == "" {
		return false
	}
	if isOptionLetterString(lower) || isRomanOptionString(lower) {
		return true
	}
	if len(lower) > 2 && strings.HasPrefix(lower, "option") {
		// "Option 1", "option-a", etc. are treated as option markers.
		return true
	}
	for i := 0; i < len(lower); i++ {
		if lower[i] < '0' || lower[i] > '9' {
			return false
		}
	}
	return true
}

// stripOptionPrefix removes one common option marker prefix if present.
// Example: "(1) x" -> "x", "A) x" -> "x", "iii. x" -> "x".
func stripOptionPrefix(s string) string {
	if body, ok := parseOptionLine(s); ok {
		return body
	}
	return strings.TrimSpace(s)
}

func normalizeOptionText(s string) string {
	return strings.Join(strings.Fields(strings.ToLower(strings.TrimSpace(s))), " ")
}

// fixInvalidJSONEscapes doubles any backslash that is likely part of a LaTeX
// command but appears as a raw JSON escape (e.g. \frac, \text, \beta).
// This converts sequences like \( \) \[ \] \frac into valid JSON string
// content while still allowing standard JSON escapes like \n.
func fixInvalidJSONEscapes(s string) string {
	var b strings.Builder
	b.Grow(len(s) + 64)
	i := 0
	for i < len(s) {
		if s[i] == '\\' && i+1 < len(s) {
			next := s[i+1]
			switch next {
			case '"', '\\', '/':
				// Always valid JSON escapes.
				b.WriteByte(s[i])
				b.WriteByte(next)
				i += 2
			case 'u':
				// Keep \u only if followed by 4 hex digits, otherwise treat as LaTeX.
				if i+5 < len(s) && isHex(s[i+2]) && isHex(s[i+3]) && isHex(s[i+4]) && isHex(s[i+5]) {
					b.WriteByte(s[i])
					b.WriteByte(next)
					b.WriteByte(s[i+2])
					b.WriteByte(s[i+3])
					b.WriteByte(s[i+4])
					b.WriteByte(s[i+5])
					i += 6
					continue
				}
				b.WriteByte('\\')
				b.WriteByte('\\')
				i++
			case 'b', 'f', 'n', 'r', 't':
				// If the escape letter is followed by another alphabetic letter,
				// it's almost certainly a LaTeX command (\frac, \text, \beta, ...),
				// not an intended JSON control escape.
				if i+2 < len(s) && isASCIIAlpha(s[i+2]) {
					b.WriteByte('\\')
					b.WriteByte('\\')
					i++
				} else {
					b.WriteByte(s[i])
					b.WriteByte(next)
					i += 2
				}
			default:
				// Invalid JSON escape (e.g. \( \) \[ \]) — emit \\ and let the
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

func isHex(c byte) bool {
	return (c >= '0' && c <= '9') ||
		(c >= 'a' && c <= 'f') ||
		(c >= 'A' && c <= 'F')
}

func isASCIIAlpha(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}
