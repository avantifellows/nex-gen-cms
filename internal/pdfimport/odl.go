package pdfimport

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	_ "image/png" // register PNG decoder
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/avantifellows/nex-gen-cms/utils"
)

// ODLElement represents one element emitted by OpenDataLoader's JSON output.
// Field names follow ODL's exact snake_case/space convention.
type ODLElement struct {
	Type        string     `json:"type"`         // "paragraph","heading","picture","formula","table","list","caption"
	PageNumber  int        `json:"page number"`  // 1-indexed
	BoundingBox [4]float64 `json:"bounding box"` // [left, bottom, right, top] in PDF points (Y=0 at bottom)
	Content     string     `json:"content"`      // text content; empty for pictures
}

// odlRasterDPI is the DPI used when calling pdfToImages for ODL-mode crops.
// Must stay in sync with bbox→pixel conversion in odlBboxToPixelRect.
const odlRasterDPI = 150

// odlCropPaddingFraction adds relative padding on each side of figure crops.
const odlCropPaddingFraction = 0.02

// extractWithODL runs the Python wrapper script and returns the parsed ODL element list.
// pdfPath must be an absolute path to an existing file on disk.
func extractWithODL(pdfPath string) ([]ODLElement, error) {
	scriptPath, err := findODLScript()
	if err != nil {
		return nil, err
	}

	var stdout, stderr bytes.Buffer
	cmd := exec.Command("python", scriptPath, pdfPath)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	cmd.Env = append(os.Environ(), "PYTHONIOENCODING=utf-8", "PYTHONUTF8=1")

	if runErr := cmd.Run(); runErr != nil {
		// The script writes a JSON error object to stdout on failure.
		if stdout.Len() > 0 {
			var errObj struct {
				Error string `json:"error"`
			}
			if jsonErr := json.Unmarshal(stdout.Bytes(), &errObj); jsonErr == nil && errObj.Error != "" {
				return nil, fmt.Errorf("odl_extract.py: %s", errObj.Error)
			}
		}
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = runErr.Error()
		}
		return nil, fmt.Errorf("odl_extract.py exited with error: %s", msg)
	}

	// Happy path: stdout should be a JSON payload. Some ODL/Java versions print
	// INFO logs to stdout before JSON, so we strip leading noise.
	if stdout.Len() == 0 {
		return nil, fmt.Errorf("odl_extract.py produced no output")
	}
	jsonPayload, err := extractFirstJSONPayload(stdout.Bytes())
	if err != nil {
		snippet := stdout.String()
		if len(snippet) > 300 {
			snippet = snippet[:300] + "…"
		}
		return nil, fmt.Errorf("parsing ODL JSON output: %v\nraw snippet: %s", err, snippet)
	}

	// Guard against the script writing a JSON error object even on exit 0.
	var errObj struct {
		Error string `json:"error"`
	}
	if json.Unmarshal(jsonPayload, &errObj) == nil && errObj.Error != "" {
		return nil, fmt.Errorf("odl_extract.py: %s", errObj.Error)
	}

	elements, err := parseODLElementsPayload(jsonPayload)
	if err != nil {
		snippet := stdout.String()
		if len(snippet) > 300 {
			snippet = snippet[:300] + "…"
		}
		return nil, fmt.Errorf("parsing ODL JSON output: %v\nraw snippet: %s", err, snippet)
	}
	return elements, nil
}

// extractFirstJSONPayload returns the first valid JSON object/array in raw bytes.
// It is tolerant to leading text (e.g., Java INFO logs printed before JSON).
func extractFirstJSONPayload(raw []byte) ([]byte, error) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 {
		return nil, fmt.Errorf("empty output")
	}
	// Fast path: already clean JSON.
	if (trimmed[0] == '{' || trimmed[0] == '[') && json.Valid(trimmed) {
		return trimmed, nil
	}

	// Slow path: locate first valid JSON start, then use Decoder to parse one value.
	for i := 0; i < len(trimmed); i++ {
		if trimmed[i] != '{' && trimmed[i] != '[' {
			continue
		}
		candidate := trimmed[i:]
		dec := json.NewDecoder(bytes.NewReader(candidate))
		var v any
		if err := dec.Decode(&v); err != nil {
			continue
		}
		payload, err := json.Marshal(v)
		if err == nil && len(payload) > 0 {
			return payload, nil
		}
	}
	return nil, fmt.Errorf("could not locate valid JSON payload in script output")
}

// parseODLElementsPayload supports both top-level array payloads and wrapped
// object payloads used by some ODL versions.
func parseODLElementsPayload(payload []byte) ([]ODLElement, error) {
	// Parse the full tree and recursively flatten nested "kids"/"list items"
	// so deeply nested question text is not dropped.
	var root any
	if err := json.Unmarshal(payload, &root); err != nil {
		return nil, err
	}
	flat := flattenODLElements(root)
	if len(flat) == 0 {
		return nil, fmt.Errorf("no ODL elements found after recursive flatten")
	}
	return flat, nil
}

func flattenODLElements(root any) []ODLElement {
	out := make([]ODLElement, 0, 256)
	flattenODLElementsInto(root, &out)
	return out
}

func flattenODLElementsInto(node any, out *[]ODLElement) {
	switch v := node.(type) {
	case []any:
		for _, child := range v {
			flattenODLElementsInto(child, out)
		}
	case map[string]any:
		if el, ok := mapToODLElement(v); ok {
			*out = append(*out, el)
		}
		for key, child := range v {
			if key == "kids" || key == "list items" {
				flattenODLElementsInto(child, out)
				continue
			}
			switch child.(type) {
			case []any, map[string]any:
				flattenODLElementsInto(child, out)
			}
		}
	}
}

func mapToODLElement(m map[string]any) (ODLElement, bool) {
	typeVal, hasType := m["type"].(string)
	bboxRaw, hasBBox := m["bounding box"].([]any)
	if !hasType && !hasBBox {
		return ODLElement{}, false
	}

	pageNum := utils.AnyToInt(m["page number"])
	content, _ := m["content"].(string)

	var bbox [4]float64
	if hasBBox && len(bboxRaw) == 4 {
		for i := 0; i < 4; i++ {
			bbox[i] = utils.AnyToFloat64(bboxRaw[i])
		}
	}

	return ODLElement{
		Type:        typeVal,
		PageNumber:  pageNum,
		BoundingBox: bbox,
		Content:     content,
	}, true
}

// findODLScript locates scripts/odl_extract.py relative to the working directory
// or its parent (to handle running from cmd/).
func findODLScript() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("getting cwd: %w", err)
	}
	candidates := []string{
		filepath.Join(cwd, "scripts", "odl_extract.py"),
		filepath.Join(filepath.Dir(cwd), "scripts", "odl_extract.py"),
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}
	return "", fmt.Errorf(
		"odl_extract.py not found; looked in: %s",
		strings.Join(candidates, ", "),
	)
}

// odlBboxToPixelRect converts an ODL bounding box [left, bottom, right, top] (PDF points,
// Y=0 at bottom) to an image.Rectangle for a page rasterised at odlRasterDPI DPI.
// A small padding (2% of the crop dimensions) is added on every side.
func odlBboxToPixelRect(bbox [4]float64, pageImg []byte) (image.Rectangle, error) {
	cfg, _, err := image.DecodeConfig(bytes.NewReader(pageImg))
	if err != nil {
		return image.Rectangle{}, fmt.Errorf("reading page image config: %w", err)
	}
	imgW, imgH := cfg.Width, cfg.Height

	scale := float64(odlRasterDPI) / 72.0
	left, bottom, right, top := bbox[0], bbox[1], bbox[2], bbox[3]

	// PDF Y origin is at the bottom of the page; image Y origin is at the top.
	// We approximate the page height in PDF points from the rasterised image height.
	pageHeightPts := float64(imgH) / scale

	px0 := left * scale
	py0 := (pageHeightPts - top) * scale // PDF top  → image top
	px1 := right * scale
	py1 := (pageHeightPts - bottom) * scale // PDF bottom → image bottom

	// Add 2% padding relative to the crop box dimensions.
	padX := (px1 - px0) * odlCropPaddingFraction
	padY := (py1 - py0) * odlCropPaddingFraction

	ix0 := int(math.Max(0, px0-padX))
	iy0 := int(math.Max(0, py0-padY))
	ix1 := int(math.Min(float64(imgW), px1+padX))
	iy1 := int(math.Min(float64(imgH), py1+padY))

	if ix1 <= ix0 || iy1 <= iy0 {
		return image.Rectangle{}, fmt.Errorf("degenerate crop rect after padding")
	}
	return image.Rect(ix0, iy0, ix1, iy1), nil
}

// odlFigureRef records where ODL placed a picture/image fragment on a page.
type odlFigureRef struct {
	PageNumber  int
	BoundingBox [4]float64
}

// ODLFigureCrop is a PNG slice of a figure plus its width in PDF points (1 pt = 1/72 in).
type ODLFigureCrop struct {
	PNG     []byte
	WidthPt float64
}

// odlBBoxSizePt returns width and height of a PDF-point bbox [left, bottom, right, top].
func odlBBoxSizePt(bbox [4]float64) (width, height float64) {
	return bbox[2] - bbox[0], bbox[3] - bbox[1]
}

// unionODLBboxes returns the minimal axis-aligned box containing all inputs.
// Each box is [left, bottom, right, top] in PDF points (Y=0 at bottom).
func unionODLBboxes(boxes [][4]float64) ([4]float64, bool) {
	if len(boxes) == 0 {
		return [4]float64{}, false
	}
	u := boxes[0]
	for _, b := range boxes[1:] {
		if b[0] < u[0] {
			u[0] = b[0]
		}
		if b[1] < u[1] {
			u[1] = b[1]
		}
		if b[2] > u[2] {
			u[2] = b[2]
		}
		if b[3] > u[3] {
			u[3] = b[3]
		}
	}
	return u, u[2] > u[0] && u[3] > u[1]
}

// cropODLMergedFigures unions PDF-point bboxes for the given figure refs (same page)
// and returns one PNG crop covering the full diagram region.
func cropODLMergedFigures(pageImages [][]byte, refs []odlFigureRef) ([]byte, [4]float64, error) {
	if len(refs) == 0 {
		return nil, [4]float64{}, fmt.Errorf("no figure refs to merge")
	}
	page := refs[0].PageNumber
	boxes := make([][4]float64, 0, len(refs))
	for _, ref := range refs {
		if ref.PageNumber != page {
			return nil, [4]float64{}, fmt.Errorf("figures span pages %d and %d", page, ref.PageNumber)
		}
		boxes = append(boxes, ref.BoundingBox)
	}
	union, ok := unionODLBboxes(boxes)
	if !ok {
		return nil, [4]float64{}, fmt.Errorf("degenerate union bbox")
	}
	pageIdx := page - 1
	if pageIdx < 0 || pageIdx >= len(pageImages) {
		return nil, union, fmt.Errorf("page index %d out of range", pageIdx)
	}
	rect, err := odlBboxToPixelRect(union, pageImages[pageIdx])
	if err != nil {
		return nil, union, err
	}
	crop, err := cropPageToPNG(pageImages[pageIdx], rect)
	if err != nil {
		return nil, union, err
	}
	return crop, union, nil
}

// cropPageToPNG crops the given rectangle out of a PNG-encoded page image and
// returns the cropped region as a PNG-encoded byte slice.
func cropPageToPNG(pageImg []byte, rect image.Rectangle) ([]byte, error) {
	img, _, err := image.Decode(bytes.NewReader(pageImg))
	if err != nil {
		return nil, fmt.Errorf("decoding page image: %w", err)
	}
	type subImager interface {
		SubImage(image.Rectangle) image.Image
	}
	si, ok := img.(subImager)
	if !ok {
		return nil, fmt.Errorf("image type %T does not support SubImage", img)
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, si.SubImage(rect)); err != nil {
		return nil, fmt.Errorf("re-encoding cropped image: %w", err)
	}
	return buf.Bytes(), nil
}

// sortODLElementsReadingOrder attempts a deterministic reading order for exam PDFs.
// For each page it sorts in a left-column then right-column flow, each column
// top-to-bottom. This avoids cross-column mixing in two-column papers.
func sortODLElementsReadingOrder(elements []ODLElement) []ODLElement {
	if len(elements) == 0 {
		return elements
	}
	perPage := map[int][]ODLElement{}
	var pages []int
	seenPage := map[int]bool{}
	for _, el := range elements {
		perPage[el.PageNumber] = append(perPage[el.PageNumber], el)
		if !seenPage[el.PageNumber] {
			seenPage[el.PageNumber] = true
			pages = append(pages, el.PageNumber)
		}
	}
	sort.Ints(pages)

	out := make([]ODLElement, 0, len(elements))
	for _, p := range pages {
		pageEls := perPage[p]
		if len(pageEls) == 0 {
			continue
		}

		minLeft := pageEls[0].BoundingBox[0]
		maxRight := pageEls[0].BoundingBox[2]
		for _, el := range pageEls[1:] {
			if el.BoundingBox[0] < minLeft {
				minLeft = el.BoundingBox[0]
			}
			if el.BoundingBox[2] > maxRight {
				maxRight = el.BoundingBox[2]
			}
		}
		midX := (minLeft + maxRight) / 2

		leftCol := make([]ODLElement, 0, len(pageEls))
		rightCol := make([]ODLElement, 0, len(pageEls))
		for _, el := range pageEls {
			centerX := (el.BoundingBox[0] + el.BoundingBox[2]) / 2
			if centerX <= midX {
				leftCol = append(leftCol, el)
			} else {
				rightCol = append(rightCol, el)
			}
		}

		sortColumn := func(col []ODLElement) {
			sort.SliceStable(col, func(i, j int) bool {
				topI := col[i].BoundingBox[3]
				topJ := col[j].BoundingBox[3]
				if math.Abs(topI-topJ) > 2.0 {
					return topI > topJ // top-to-bottom
				}
				return col[i].BoundingBox[0] < col[j].BoundingBox[0] // tie-break by left
			})
		}
		sortColumn(leftCol)
		sortColumn(rightCol)

		out = append(out, leftCol...)
		out = append(out, rightCol...)
	}
	return out
}
