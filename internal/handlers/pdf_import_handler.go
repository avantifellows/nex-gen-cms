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

// matrixEmbedPrefix / matrixEmbedSuffix bracket server-generated matrix HTML inside
// question_text so buildProcessedText can escape plain text without mangling tags.
const matrixEmbedPrefix = `<div class="my-3 overflow-x-auto"><table class="matrix-extract`
const matrixEmbedSuffix = `</tbody></table></div>`

// odlFigureNumRegex matches [FIGURE_N] tokens from ODL layout extraction.
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
- If the PDF shows an equilibrium constant above a reversible reaction arrow, encode it in LaTeX as \( \overset{K_1}{\rightleftharpoons} \) using the same symbol as printed (e.g. \(K_1\), \(K_2\), \(K_c\)); do not use a bare \( \rightleftharpoons \) that omits the label.
- Braced superscripts must be ^{...} on the base (e.g. \(2^{3}\), \(x^{-n}\)). Do not insert a spurious backslash before the caret: \(2\^{3}\) is wrong and breaks MathJax; use \(2^{3}\) instead.

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

	return questions, rawResp, nil
}

// runODLPipeline runs OpenDataLoader once over the PDF and crops figures for hybrid import.
// onlyFigureQuestions limits crop assignment to direct-PDF has_figure questions (empty = all chunks).
func runODLPipeline(pdfBytes []byte, onlyFigureQuestions map[int]bool) (map[int][]byte, error) {
	tmpPDF, err := os.CreateTemp("", "odl-input-*.pdf")
	if err != nil {
		return nil, fmt.Errorf("creating temp PDF: %w", err)
	}
	defer os.Remove(tmpPDF.Name())
	if _, err := tmpPDF.Write(pdfBytes); err != nil {
		tmpPDF.Close()
		return nil, fmt.Errorf("writing temp PDF: %w", err)
	}
	tmpPDF.Close()

	pageImages, err := pdfToImages(pdfBytes, odlRasterDPI)
	if err != nil {
		return nil, fmt.Errorf("rasterising PDF: %w", err)
	}

	elements, err := extractWithODL(tmpPDF.Name())
	if err != nil {
		return nil, fmt.Errorf("OpenDataLoader extraction: %w", err)
	}
	elements = sortODLElementsReadingOrder(elements)

	figureCounter := 0
	figureCrops := map[int][]byte{}
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

	chunks := splitODLQuestionChunks(contentBuf.String())
	if len(chunks) == 0 {
		return nil, fmt.Errorf("could not detect question boundaries in ODL extracted text")
	}

	figureByQuestion := make(map[int][]byte)
	for _, ch := range chunks {
		if len(onlyFigureQuestions) > 0 && !onlyFigureQuestions[ch.Number] {
			continue
		}
		for _, figNum := range ch.FigureNums {
			if crop, ok := figureCrops[figNum]; ok && len(crop) > 0 {
				figureByQuestion[ch.Number] = crop
				break
			}
		}
	}

	return figureByQuestion, nil
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

	figQuestions := make(map[int]bool)
	for _, q := range directQuestions {
		if q.HasFigure {
			figQuestions[q.Number] = true
		}
	}
	figureByQuestion, err := runODLPipeline(pdfBytes, figQuestions)
	var odlRaw string
	if err != nil {
		odlRaw = "ODL layout failed: " + err.Error()
	} else {
		odlRaw = fmt.Sprintf("ODL layout: %d PNG crop(s) for has_figure questions", len(figureByQuestion))
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

type odlQuestionChunk struct {
	Number     int
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
		var figureNums []int
		if figMatches := odlFigureNumRegex.FindAllStringSubmatch(chunkText, -1); len(figMatches) > 0 {
			seenFig := map[int]bool{}
			for _, m := range figMatches {
				if len(m) < 2 {
					continue
				}
				fn, convErr := strconv.Atoi(m[1])
				if convErr != nil || seenFig[fn] {
					continue
				}
				seenFig[fn] = true
				figureNums = append(figureNums, fn)
			}
		}
		chunks = append(chunks, odlQuestionChunk{
			Number:     n,
			FigureNums: figureNums,
		})
	}
	return chunks
}

// postProcessQuestions builds ProcessedText with the [FIGURE] placeholder
// replaced inline by a figure box (image in ODL mode, text description in direct mode).
func postProcessQuestions(questions []ExtractedQuestion) {
	for i := range questions {
		q := &questions[i]
		q.Text = strings.ReplaceAll(q.Text, `\n`, "\n")
		for j := range q.Options {
			q.Options[j] = strings.ReplaceAll(q.Options[j], `\n`, "\n")
			q.Options[j] = normalizeOptionSoftWraps(q.Options[j])
		}
		q.Text = trimDuplicatedOptionsFromQuestionText(q.Text, q.Options)
		if intro, h1, h2, rows, ok := tryExtractMatrixMatchLayout(q.Text); ok {
			introNorm := normalizeQuestionSoftWraps(intro)
			tableHTML := buildMatrixMatchTableHTMLString(h1, h2, rows)
			q.Text = strings.TrimSpace(introNorm + "\n" + tableHTML)
			q.Type = "matrix_match"
		} else {
			q.Text = normalizeQuestionSoftWraps(q.Text)
		}
		q.ProcessedText = buildProcessedText(q)
	}
}

// normalizeListHeaderLine collapses spaces and unicode dashes for comparing
// "List-I", "List – I", "List  II", etc.
func normalizeListHeaderLine(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))
	s = strings.ReplaceAll(s, "\t", " ")
	s = strings.Join(strings.Fields(s), " ")
	s = strings.ReplaceAll(s, "–", "-")
	s = strings.ReplaceAll(s, "—", "-")
	return strings.ReplaceAll(s, " ", "")
}

func isListIIHeaderLine(line string) bool {
	n := normalizeListHeaderLine(line)
	switch n {
	case "list-ii", "listii", "list-2", "list2":
		return true
	default:
		return false
	}
}

func isListIHeaderLine(line string) bool {
	n := normalizeListHeaderLine(line)
	switch n {
	case "list-i", "listi", "list-1", "list1":
		return true
	default:
		return false
	}
}

func matrixMatchNonEmptyLines(s string) []string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	var out []string
	for _, line := range strings.Split(s, "\n") {
		t := strings.TrimSpace(line)
		if t != "" {
			out = append(out, t)
		}
	}
	return out
}

// matrixMatchNumberedOptionStart is true for lines like "(1) ..." that start
// the inline MCQ key block after a matrix table.
func matrixMatchNumberedOptionStart(line string) bool {
	t := strings.TrimSpace(line)
	if len(t) < 3 || t[0] != '(' {
		return false
	}
	close := strings.IndexByte(t, ')')
	if close <= 1 {
		return false
	}
	for _, ch := range t[1:close] {
		if ch < '0' || ch > '9' {
			return false
		}
	}
	return true
}

// tryExtractMatrixMatchLayout detects "List-I" / "List-II" headers followed by
// alternating row pairs (two lines per row: first column, second column). Row
// labels are not validated so letters, digits, roman numerals, etc. all work.
func tryExtractMatrixMatchLayout(text string) (intro, col1Hdr, col2Hdr string, rows [][2]string, ok bool) {
	lines := matrixMatchNonEmptyLines(text)
	if len(lines) < 4 {
		return "", "", "", nil, false
	}
	headerIdx := -1
	for i := 0; i+1 < len(lines); i++ {
		if isListIHeaderLine(lines[i]) && isListIIHeaderLine(lines[i+1]) {
			headerIdx = i
			break
		}
	}
	if headerIdx < 0 {
		return "", "", "", nil, false
	}
	col1Hdr = strings.TrimSpace(lines[headerIdx])
	col2Hdr = strings.TrimSpace(lines[headerIdx+1])
	if col1Hdr == "" || col2Hdr == "" {
		return "", "", "", nil, false
	}
	dataStart := headerIdx + 2
	pos := dataStart
	const maxMatrixRows = 10
	for pos+1 < len(lines) && len(rows) < maxMatrixRows {
		if matrixMatchNumberedOptionStart(lines[pos]) {
			break
		}
		rows = append(rows, [2]string{lines[pos], lines[pos+1]})
		pos += 2
	}
	if len(rows) < 2 {
		return "", "", "", nil, false
	}
	if headerIdx == 0 {
		intro = ""
	} else {
		intro = strings.Join(lines[:headerIdx], "\n")
	}
	return intro, col1Hdr, col2Hdr, rows, true
}

func buildMatrixMatchTableHTMLString(col1Hdr, col2Hdr string, rows [][2]string) string {
	cell := func(s string) string {
		return strings.ReplaceAll(html.EscapeString(s), "\n", "<br>")
	}
	var b strings.Builder
	b.WriteString(matrixEmbedPrefix)
	b.WriteString(` min-w-[12rem] w-full max-w-2xl border-collapse border border-gray-300 text-sm">`)
	b.WriteString("<thead><tr>")
	b.WriteString(`<th scope="col" class="border border-gray-300 bg-gray-100 px-3 py-2 text-left font-semibold text-gray-800">`)
	b.WriteString(html.EscapeString(col1Hdr))
	b.WriteString(`</th><th scope="col" class="border border-gray-300 bg-gray-100 px-3 py-2 text-left font-semibold text-gray-800">`)
	b.WriteString(html.EscapeString(col2Hdr))
	b.WriteString("</th></tr></thead><tbody>")
	for _, r := range rows {
		b.WriteString("<tr>")
		b.WriteString(`<td class="border border-gray-300 px-3 py-2 align-top text-gray-800 whitespace-pre-wrap">`)
		b.WriteString(cell(r[0]))
		b.WriteString(`</td><td class="border border-gray-300 px-3 py-2 align-top text-gray-800 whitespace-pre-wrap">`)
		b.WriteString(cell(r[1]))
		b.WriteString("</td></tr>")
	}
	b.WriteString(matrixEmbedSuffix)
	return b.String()
}

func findEmbeddedMatrixTableRange(s string) (start, end int, ok bool) {
	start = strings.Index(s, matrixEmbedPrefix)
	if start < 0 {
		return 0, 0, false
	}
	rel := strings.Index(s[start:], matrixEmbedSuffix)
	if rel < 0 {
		return 0, 0, false
	}
	end = start + rel + len(matrixEmbedSuffix)
	return start, end, true
}

// renderPlainQuestionHTML escapes plain question text and inlines [FIGURE].
// If suppressTrailingFigure is true, an implicit trailing figure box is not
// appended when the segment has no [FIGURE] marker (used for matrix stems so
// the figure can be placed after the embedded table).
func renderPlainQuestionHTML(plain string, q *ExtractedQuestion, figureBox string, escapeText func(string) string, suppressTrailingFigure bool) string {
	if q.HasFigure && strings.Contains(plain, "[FIGURE]") {
		parts := strings.Split(plain, "[FIGURE]")
		var sb strings.Builder
		for i, part := range parts {
			sb.WriteString(escapeText(part))
			if i < len(parts)-1 {
				sb.WriteString(figureBox)
			}
		}
		return sb.String()
	}
	var sb strings.Builder
	sb.WriteString(escapeText(plain))
	if q.HasFigure && figureBox != "" && !strings.Contains(plain, "[FIGURE]") && !suppressTrailingFigure {
		sb.WriteString(figureBox)
	}
	return sb.String()
}

var questionListLineStartRegex = regexp.MustCompile(`(?i)^\s*(?:\(?[a-d]\)|\(?\d{1,3}\)|\([ivx]+\)|[ivx]+\)|\d+\.)\s+`)
var whitespaceRunRegex = regexp.MustCompile(`\s+`)

// normalizeSoftWrapsSimple collapses PDF soft-wrap newlines into spaces.
// If preserveListLineStarts is true, it keeps a newline before list-like lines
// such as "(1) ...", "A) ...", "1. ...".
//
// This is intentionally minimal to address:
// - question_text: "....\nfollows the\n(1) ..." → ".... follows the\n(1) ..."
// - options: ".... nuclear and\ncell division ...." → ".... nuclear and cell division ...."
func normalizeSoftWrapsSimple(s string, preserveListLineStarts bool) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	lines := strings.Split(s, "\n")
	if len(lines) <= 1 {
		return strings.TrimSpace(s)
	}

	var outLines []string
	var cur strings.Builder

	flush := func() {
		if cur.Len() == 0 {
			return
		}
		outLines = append(outLines, strings.TrimSpace(whitespaceRunRegex.ReplaceAllString(cur.String(), " ")))
		cur.Reset()
	}

	for _, raw := range lines {
		line := strings.TrimSpace(raw)
		if line == "" {
			// Ignore blank lines for now; neither example needs paragraph preservation.
			continue
		}

		if preserveListLineStarts && questionListLineStartRegex.MatchString(line) {
			flush()
			outLines = append(outLines, strings.TrimSpace(whitespaceRunRegex.ReplaceAllString(line, " ")))
			continue
		}

		if cur.Len() > 0 {
			cur.WriteByte(' ')
		}
		cur.WriteString(line)
	}
	flush()

	sep := " "
	if preserveListLineStarts {
		sep = "\n"
	}
	return strings.TrimSpace(strings.Join(outLines, sep))
}

// normalizeOptionSoftWraps collapses PDF "soft-wrap" newlines inside a single
// option into spaces. Options are displayed one-per-line in UI, so preserving
// intra-option line breaks is usually undesirable.
func normalizeOptionSoftWraps(s string) string {
	return normalizeSoftWrapsSimple(s, false)
}

// normalizeQuestionSoftWraps collapses PDF "soft-wrap" line breaks inside a
// sentence into spaces, while preserving real paragraph breaks and list-like
// formatting (e.g. lines starting with "(i)", "(1)", "A)", "1.").
func normalizeQuestionSoftWraps(s string) string {
	return normalizeSoftWrapsSimple(s, true)
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
				`<div class="my-3">`+
					`<img src="data:image/png;base64,%s" alt="" `+
					`style="display:block;margin:0 auto;max-width:min(100%%,14em);max-height:min(36vh,11em);width:auto;height:auto;object-fit:contain" />`+
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

	body := q.Text
	if start, end, ok := findEmbeddedMatrixTableRange(body); ok {
		before := strings.TrimSpace(body[:start])
		tableHTML := body[start:end]
		after := strings.TrimSpace(body[end:])
		combinedPlain := before
		if after != "" {
			if combinedPlain != "" {
				combinedPlain += "\n"
			}
			combinedPlain += after
		}
		hasFigureMarker := q.HasFigure && strings.Contains(combinedPlain, "[FIGURE]")

		var sb strings.Builder
		sb.WriteString(renderPlainQuestionHTML(before, q, figureBox, escapeText, true))
		sb.WriteString(tableHTML)
		sb.WriteString(renderPlainQuestionHTML(after, q, figureBox, escapeText, true))
		if q.HasFigure && figureBox != "" && !hasFigureMarker {
			sb.WriteString(figureBox)
		}
		return htmltemplate.HTML(sb.String())
	}

	return htmltemplate.HTML(renderPlainQuestionHTML(body, q, figureBox, escapeText, false))
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
