package handlers

import (
	"fmt"
	"html"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/avantifellows/nex-gen-cms/internal/curriculumconfig"
	"github.com/avantifellows/nex-gen-cms/internal/views"
)

const (
	curriculumConfigTemplate      = "curriculum_config.html"
	curriculumConfigTableTemplate = "curriculum_config_table.html"
)

type CurriculumConfigHandler struct {
	repo curriculumconfig.Repository
}

func NewCurriculumConfigHandler(repo curriculumconfig.Repository) *CurriculumConfigHandler {
	return &CurriculumConfigHandler{repo: repo}
}

func (h *CurriculumConfigHandler) Page(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodGet) {
		return
	}
	readiness, ok := h.ensureReady(w, r, true)
	if !ok && !readiness.Ready {
		return
	}
	query := listQueryFromRequest(r)
	options, err := h.repo.FilterOptions(r.Context())
	if err != nil {
		log.Printf("curriculum config filter options: %v", err)
		http.Error(w, "Could not load Curriculum Config filters", http.StatusInternalServerError)
		return
	}
	result, err := h.repo.List(r.Context(), query)
	if err != nil {
		log.Printf("curriculum config list: %v", err)
		http.Error(w, "Could not load Curriculum Config", http.StatusInternalServerError)
		return
	}
	data := map[string]any{
		"Readiness": readiness,
		"Result":    result,
		"Options":   options,
		"Query":     query,
	}
	views.ExecuteTemplates(w, data, curriculumConfigFuncMap(), baseTemplate, curriculumConfigTemplate, curriculumConfigTableTemplate)
}

func (h *CurriculumConfigHandler) Table(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodGet) {
		return
	}
	if _, ok := h.ensureReady(w, r, false); !ok {
		return
	}
	query := listQueryFromRequest(r)
	result, err := h.repo.List(r.Context(), query)
	if err != nil {
		log.Printf("curriculum config list: %v", err)
		http.Error(w, "Could not load Curriculum Config", http.StatusInternalServerError)
		return
	}
	views.ExecuteTemplate(curriculumConfigTableTemplate, w, tableViewData{Result: result, Query: query}, curriculumConfigFuncMap())
}

func (h *CurriculumConfigHandler) New(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodGet) {
		return
	}
	if _, ok := h.ensureReady(w, r, false); !ok {
		return
	}
	h.placeholder(w, "Add LMS Chapter Exam Config is not available in this slice")
}

func (h *CurriculumConfigHandler) Edit(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodGet) {
		return
	}
	if _, ok := h.ensureReady(w, r, false); !ok {
		return
	}
	h.placeholder(w, "Edit LMS Chapter Exam Config is not available in this slice")
}

func (h *CurriculumConfigHandler) Remove(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodGet) {
		return
	}
	if _, ok := h.ensureReady(w, r, false); !ok {
		return
	}
	h.placeholder(w, "Remove from syllabus is not available in this slice")
}

func (h *CurriculumConfigHandler) ChapterOptions(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodGet) {
		return
	}
	if _, ok := h.ensureReady(w, r, false); !ok {
		return
	}
	h.placeholder(w, "Chapter options are not available in this slice")
}

func (h *CurriculumConfigHandler) Impact(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodGet) {
		return
	}
	if _, ok := h.ensureReady(w, r, false); !ok {
		return
	}
	h.placeholder(w, "Impact preview is not available in this slice")
}

func (h *CurriculumConfigHandler) Create(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodPost) {
		return
	}
	if _, ok := h.ensureMutationReady(w, r); !ok {
		return
	}
	h.placeholder(w, "Create LMS Chapter Exam Config is not available in this slice")
}

func (h *CurriculumConfigHandler) Update(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodPost) {
		return
	}
	if _, ok := h.ensureMutationReady(w, r); !ok {
		return
	}
	h.placeholder(w, "Update LMS Chapter Exam Config is not available in this slice")
}

func (h *CurriculumConfigHandler) RemoveFromSyllabus(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodPost) {
		return
	}
	if _, ok := h.ensureMutationReady(w, r); !ok {
		return
	}
	h.placeholder(w, "Remove from syllabus is not available in this slice")
}

func (h *CurriculumConfigHandler) Export(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodGet) {
		return
	}
	if _, ok := h.ensureReady(w, r, false); !ok {
		return
	}
	h.placeholder(w, "Curriculum Config export is not available in this slice")
}

func (h *CurriculumConfigHandler) ensureMutationReady(w http.ResponseWriter, r *http.Request) (curriculumconfig.Readiness, bool) {
	readiness, ok := h.ensureReady(w, r, false)
	if !ok {
		return readiness, false
	}
	if !readiness.MutationReady {
		h.writeUnavailable(w, false, "Curriculum Config mutations unavailable", readiness.MutationReasons)
		return readiness, false
	}
	return readiness, true
}

func (h *CurriculumConfigHandler) ensureReady(w http.ResponseWriter, r *http.Request, fullPage bool) (curriculumconfig.Readiness, bool) {
	readiness, err := h.repo.SchemaReadiness(r.Context())
	if err != nil {
		log.Printf("curriculum config readiness: %v", err)
		h.writeUnavailable(w, fullPage, "Curriculum Config unavailable", []string{"Schema readiness could not be verified"})
		return readiness, false
	}
	if !readiness.Ready {
		h.writeUnavailable(w, fullPage, "Curriculum Config unavailable", readiness.Reasons)
		return readiness, false
	}
	return readiness, true
}

func (h *CurriculumConfigHandler) placeholder(w http.ResponseWriter, message string) {
	w.WriteHeader(http.StatusServiceUnavailable)
	_, _ = fmt.Fprintf(w, `<section class="panel" role="status"><h2>Unavailable</h2><p>%s.</p></section>`, message)
}

func (h *CurriculumConfigHandler) writeUnavailable(w http.ResponseWriter, fullPage bool, title string, reasons []string) {
	w.WriteHeader(http.StatusServiceUnavailable)
	if len(reasons) == 0 {
		reasons = []string{"Required LMS Chapter Exam Config schema is unavailable"}
	}
	data := map[string]any{
		"UnavailableTitle":   title,
		"UnavailableReasons": reasons,
		"Readiness": curriculumconfig.Readiness{
			Ready:           false,
			MutationReady:   false,
			Reasons:         reasons,
			MutationReasons: reasons,
		},
	}
	if fullPage {
		views.ExecuteTemplates(w, data, curriculumConfigFuncMap(), baseTemplate, curriculumConfigTemplate, curriculumConfigTableTemplate)
		return
	}
	_, _ = fmt.Fprintf(w, `<section class="panel" role="status"><h2>%s</h2><ul>`, html.EscapeString(title))
	for _, reason := range reasons {
		_, _ = fmt.Fprintf(w, "<li>%s</li>", html.EscapeString(reason))
	}
	_, _ = fmt.Fprint(w, "</ul></section>")
}

func allowMethod(w http.ResponseWriter, r *http.Request, method string) bool {
	if r.Method == method {
		return true
	}
	w.Header().Set("Allow", method)
	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	return false
}

type tableViewData struct {
	Result curriculumconfig.ListResult
	Query  curriculumconfig.ListQuery
}

func listQueryFromRequest(r *http.Request) curriculumconfig.ListQuery {
	values := r.URL.Query()
	return curriculumconfig.NormalizeListQuery(curriculumconfig.ListQuery{
		ExamTrack:      queryValue(values.Get("exam_track"), "jee_main"),
		Grade:          strings.TrimSpace(values.Get("grade")),
		Subject:        strings.TrimSpace(values.Get("subject")),
		Search:         strings.TrimSpace(values.Get("search")),
		ChapterID:      strings.TrimSpace(values.Get("chapter_id")),
		SyllabusStatus: queryValue(values.Get("syllabus_status"), "in_syllabus"),
		Page:           positiveInt(values.Get("page"), 1),
		Limit:          positiveInt(values.Get("limit"), 50),
		Sort:           queryValue(values.Get("sort"), "curriculum"),
		Direction:      queryValue(values.Get("dir"), "asc"),
	})
}

func queryValue(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}

func positiveInt(value string, fallback int) int {
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || parsed < 1 {
		return fallback
	}
	return parsed
}

func curriculumConfigFuncMap() template.FuncMap {
	return template.FuncMap{
		"curriculumConfigPageURL": func(query curriculumconfig.ListQuery, page int) string {
			query.Page = page
			return "/admin/curriculum-config/table?" + encodeListQuery(query)
		},
		"curriculumConfigSortURL": func(query curriculumconfig.ListQuery, sort string) string {
			if query.Sort == sort && query.Direction == "asc" {
				query.Direction = "desc"
			} else {
				query.Direction = "asc"
			}
			query.Sort = sort
			query.Page = 1
			return "/admin/curriculum-config/table?" + encodeListQuery(query)
		},
		"minus": func(left, right int) int {
			return left - right
		},
		"plus": func(left, right int) int {
			return left + right
		},
		"examTrackLabel": func(value string) string {
			switch value {
			case "jee_main":
				return "JEE Main"
			case "jee_advanced":
				return "JEE Advanced"
			case "neet":
				return "NEET"
			default:
				return value
			}
		},
		"syllabusStatusLabel": func(inSyllabus bool) string {
			if inSyllabus {
				return "In syllabus"
			}
			return "Out of syllabus"
		},
	}
}

func encodeListQuery(query curriculumconfig.ListQuery) string {
	query = curriculumconfig.NormalizeListQuery(query)
	values := url.Values{}
	values.Set("exam_track", query.ExamTrack)
	values.Set("grade", query.Grade)
	values.Set("subject", query.Subject)
	values.Set("search", query.Search)
	values.Set("chapter_id", query.ChapterID)
	values.Set("syllabus_status", query.SyllabusStatus)
	values.Set("page", strconv.Itoa(query.Page))
	values.Set("limit", strconv.Itoa(query.Limit))
	values.Set("sort", query.Sort)
	values.Set("dir", query.Direction)

	order := []string{"exam_track", "grade", "subject", "search", "chapter_id", "syllabus_status", "page", "limit", "sort", "dir"}
	parts := make([]string, 0, len(order))
	for _, key := range order {
		parts = append(parts, url.QueryEscape(key)+"="+url.QueryEscape(values.Get(key)))
	}
	return strings.Join(parts, "&")
}
