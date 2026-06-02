package handlers

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"html"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/avantifellows/nex-gen-cms/internal/auth"
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
	writeAddPanel(w, listQueryFromRequest(r), nil, curriculumconfig.ImpactResult{})
}

func (h *CurriculumConfigHandler) Edit(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodGet) {
		return
	}
	if _, ok := h.ensureReady(w, r, false); !ok {
		return
	}
	id := int64(positiveInt(r.URL.Query().Get("id"), 0))
	if id < 1 {
		http.Error(w, "Config id is required", http.StatusUnprocessableEntity)
		return
	}
	row, err := h.repo.Get(r.Context(), id)
	if err != nil {
		writeUpdateError(w, http.StatusNotFound, "Could not load LMS Chapter Exam Config", err.Error())
		return
	}
	writeEditPanel(w, listQueryFromRequest(r), row, nil, curriculumconfig.ImpactResult{})
}

func (h *CurriculumConfigHandler) Remove(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodGet) {
		return
	}
	if _, ok := h.ensureReady(w, r, false); !ok {
		return
	}
	id := int64(positiveInt(r.URL.Query().Get("id"), 0))
	if id < 1 {
		http.Error(w, "Config id is required", http.StatusUnprocessableEntity)
		return
	}
	row, err := h.repo.Get(r.Context(), id)
	if err != nil {
		writeRemoveError(w, http.StatusNotFound, "Could not load LMS Chapter Exam Config", err.Error())
		return
	}
	if !row.IsInSyllabus {
		writeRemoveError(w, http.StatusUnprocessableEntity, "Could not remove LMS Chapter Exam Config", "LMS Chapter Exam Config is already out of syllabus.")
		return
	}
	impact, err := h.repo.Impact(r.Context(), curriculumconfig.ImpactQuery{
		ConfigID:          row.ID,
		ChapterID:         row.ChapterID,
		ExamTrack:         row.ExamTrack,
		IsInSyllabus:      false,
		PrescribedMinutes: 0,
		CoverageSequence:  row.CoverageSequence,
	})
	if err != nil {
		log.Printf("curriculum config remove impact: %v", err)
		http.Error(w, "Could not load impact preview", http.StatusInternalServerError)
		return
	}
	writeRemovePanel(w, listQueryFromRequest(r), row, impact.Warnings, impact)
}

func (h *CurriculumConfigHandler) ChapterOptions(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodGet) {
		return
	}
	if _, ok := h.ensureReady(w, r, false); !ok {
		return
	}
	values := r.URL.Query()
	options, err := h.repo.ChapterOptions(r.Context(), curriculumconfig.ChapterOptionsQuery{
		ExamTrack: queryValue(values.Get("exam_track"), curriculumconfig.DefaultExamTrack),
		Grade:     queryValue(values.Get("grade"), values.Get("filter_grade")),
		Subject:   queryValue(values.Get("subject"), values.Get("filter_subject")),
		Search:    queryValue(values.Get("search"), values.Get("filter_search")),
	})
	if err != nil {
		log.Printf("curriculum config chapter options: %v", err)
		http.Error(w, "Could not load chapter options", http.StatusInternalServerError)
		return
	}
	writeChapterOptions(w, options)
}

func (h *CurriculumConfigHandler) Impact(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodGet) {
		return
	}
	if _, ok := h.ensureReady(w, r, false); !ok {
		return
	}
	query := impactQueryFromValues(r.URL.Query())
	impact, err := h.repo.Impact(r.Context(), query)
	if err != nil {
		log.Printf("curriculum config impact: %v", err)
		http.Error(w, "Could not load impact preview", http.StatusInternalServerError)
		return
	}
	writeWarningAndImpact(w, impact.Warnings, impact)
}

func (h *CurriculumConfigHandler) Create(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodPost) {
		return
	}
	if _, ok := h.ensureMutationReady(w, r); !ok {
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid create form", http.StatusBadRequest)
		return
	}
	claims := auth.FromContext(r.Context())
	adminEmail := ""
	if claims != nil {
		adminEmail = claims.Email
	}
	input := createInputFromForm(r.PostForm, adminEmail)
	result, err := h.repo.Create(r.Context(), input)
	if err != nil {
		writeMutationError(w, err)
		return
	}
	query := listQueryFromForm(r.PostForm)
	listResult, err := h.repo.List(r.Context(), query)
	if err != nil {
		log.Printf("curriculum config list after create: %v", err)
		http.Error(w, "Created config but could not refresh table", http.StatusInternalServerError)
		return
	}
	writeCreateSuccess(w, result, tableViewData{Result: listResult, Query: query})
}

func (h *CurriculumConfigHandler) Update(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodPost) {
		return
	}
	if _, ok := h.ensureMutationReady(w, r); !ok {
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid update form", http.StatusBadRequest)
		return
	}
	claims := auth.FromContext(r.Context())
	adminEmail := ""
	if claims != nil {
		adminEmail = claims.Email
	}
	input, err := editInputFromForm(r.PostForm, adminEmail)
	if err != nil {
		writeUpdateError(w, http.StatusUnprocessableEntity, "Could not update LMS Chapter Exam Config", err.Error())
		return
	}
	result, err := h.repo.Edit(r.Context(), input)
	if err != nil {
		writeEditMutationError(w, err)
		return
	}
	query := listQueryFromForm(r.PostForm)
	listResult, err := h.repo.List(r.Context(), query)
	if err != nil {
		log.Printf("curriculum config list after update: %v", err)
		http.Error(w, "Updated config but could not refresh table", http.StatusInternalServerError)
		return
	}
	writeUpdateSuccess(w, result, tableViewData{Result: listResult, Query: query})
}

func (h *CurriculumConfigHandler) RemoveFromSyllabus(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodPost) {
		return
	}
	if _, ok := h.ensureMutationReady(w, r); !ok {
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid remove form", http.StatusBadRequest)
		return
	}
	claims := auth.FromContext(r.Context())
	adminEmail := ""
	if claims != nil {
		adminEmail = claims.Email
	}
	input := removeInputFromForm(r.PostForm, adminEmail)
	result, err := h.repo.RemoveFromSyllabus(r.Context(), input)
	if err != nil {
		writeRemoveMutationError(w, err)
		return
	}
	query := listQueryFromForm(r.PostForm)
	listResult, err := h.repo.List(r.Context(), query)
	if err != nil {
		log.Printf("curriculum config list after remove: %v", err)
		http.Error(w, "Removed config but could not refresh table", http.StatusInternalServerError)
		return
	}
	writeRemoveSuccess(w, result, tableViewData{Result: listResult, Query: query})
}

func (h *CurriculumConfigHandler) Export(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodGet) {
		return
	}
	if _, ok := h.ensureReady(w, r, false); !ok {
		return
	}
	query := listQueryFromRequest(r)
	query.Page = 1
	query.Limit = 100

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	rows, err := h.repo.ExportRows(ctx, query)
	if err != nil {
		log.Printf("curriculum config export: %v", err)
		http.Error(w, "Could not export Curriculum Config", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", "attachment; filename=curriculum-config-"+time.Now().Format("2006-01-02")+".csv")

	writer := csv.NewWriter(w)
	if err := writer.Write(curriculumConfigExportHeaders()); err != nil {
		log.Printf("curriculum config export header: %v", err)
		return
	}
	for _, row := range rows {
		if err := writer.Write(curriculumConfigExportRecord(row)); err != nil {
			log.Printf("curriculum config export row: %v", err)
			return
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		log.Printf("curriculum config export flush: %v", err)
	}
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

func curriculumConfigExportHeaders() []string {
	return []string{
		"chapter_code",
		"chapter_name",
		"grade",
		"subject",
		"exam_track",
		"is_in_syllabus",
		"prescribed_minutes",
		"prescribed_hours",
		"coverage_sequence",
		"updated_by_email",
		"updated_at",
	}
}

func curriculumConfigExportRecord(row curriculumconfig.ExportRow) []string {
	return []string{
		formulaEscapeCSVCell(row.ChapterCode),
		formulaEscapeCSVCell(row.ChapterName),
		formulaEscapeCSVCell(row.Grade),
		formulaEscapeCSVCell(row.Subject),
		formulaEscapeCSVCell(row.ExamTrack),
		strconv.FormatBool(row.IsInSyllabus),
		strconv.Itoa(row.PrescribedMinutes),
		formulaEscapeCSVCell(row.PrescribedHours),
		strconv.Itoa(row.CoverageSequence),
		formulaEscapeCSVCell(row.UpdatedByEmail),
		row.UpdatedAt.Format(time.RFC3339),
	}
}

func formulaEscapeCSVCell(value string) string {
	if value == "" {
		return value
	}
	switch value[0] {
	case '=', '+', '-', '@', '\t', '\r':
		return "'" + value
	default:
		return value
	}
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

func listQueryFromForm(values url.Values) curriculumconfig.ListQuery {
	return curriculumconfig.NormalizeListQuery(curriculumconfig.ListQuery{
		ExamTrack:      queryValue(values.Get("filter_exam_track"), curriculumconfig.DefaultExamTrack),
		Grade:          strings.TrimSpace(values.Get("filter_grade")),
		Subject:        strings.TrimSpace(values.Get("filter_subject")),
		Search:         strings.TrimSpace(values.Get("filter_search")),
		ChapterID:      strings.TrimSpace(values.Get("filter_chapter_id")),
		SyllabusStatus: queryValue(values.Get("filter_syllabus_status"), curriculumconfig.DefaultSyllabusStatus),
		Page:           positiveInt(values.Get("filter_page"), curriculumconfig.DefaultPage),
		Limit:          positiveInt(values.Get("filter_limit"), curriculumconfig.DefaultLimit),
		Sort:           queryValue(values.Get("filter_sort"), curriculumconfig.DefaultSort),
		Direction:      queryValue(values.Get("filter_dir"), curriculumconfig.DefaultDirection),
	})
}

func createInputFromForm(values url.Values, adminEmail string) curriculumconfig.CreateInput {
	return curriculumconfig.CreateInput{
		ChapterID:         int64(positiveInt(values.Get("chapter_id"), 0)),
		ExamTrack:         queryValue(values.Get("exam_track"), curriculumconfig.DefaultExamTrack),
		IsInSyllabus:      syllabusStatusFromForm(values),
		PrescribedMinutes: nonNegativeInt(values.Get("prescribed_minutes"), 0),
		CoverageSequence:  positiveInt(values.Get("coverage_sequence"), 0),
		AdminEmail:        adminEmail,
	}
}

func editInputFromForm(values url.Values, adminEmail string) (curriculumconfig.EditInput, error) {
	if strings.TrimSpace(values.Get("chapter_id")) != "" || strings.TrimSpace(values.Get("exam_track")) != "" {
		return curriculumconfig.EditInput{}, errors.New("Chapter and exam-track identity cannot be changed from the edit form")
	}
	return curriculumconfig.EditInput{
		ID:                int64(positiveInt(values.Get("id"), 0)),
		IsInSyllabus:      syllabusStatusFromForm(values),
		PrescribedMinutes: nonNegativeInt(values.Get("prescribed_minutes"), 0),
		CoverageSequence:  positiveInt(values.Get("coverage_sequence"), 0),
		LockToken:         strings.TrimSpace(values.Get("lock_token")),
		AdminEmail:        adminEmail,
	}, nil
}

func removeInputFromForm(values url.Values, adminEmail string) curriculumconfig.RemoveInput {
	return curriculumconfig.RemoveInput{
		ID:         int64(positiveInt(values.Get("id"), 0)),
		LockToken:  strings.TrimSpace(values.Get("lock_token")),
		AdminEmail: adminEmail,
	}
}

func boolFormValue(value string, fallback bool) bool {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case "true", "on", "1", "yes":
		return true
	case "false", "off", "0", "no":
		return false
	default:
		return fallback
	}
}

func syllabusStatusFromForm(values url.Values) bool {
	switch strings.TrimSpace(strings.ToLower(values.Get("syllabus_status"))) {
	case "in_syllabus":
		return true
	case "out_of_syllabus":
		return false
	default:
		return boolFormValue(values.Get("is_in_syllabus"), false)
	}
}

func impactQueryFromValues(values url.Values) curriculumconfig.ImpactQuery {
	return curriculumconfig.ImpactQuery{
		ConfigID:          int64(positiveInt(values.Get("config_id"), 0)),
		ChapterID:         int64(positiveInt(values.Get("chapter_id"), 0)),
		ExamTrack:         queryValue(values.Get("exam_track"), curriculumconfig.DefaultExamTrack),
		IsInSyllabus:      impactInSyllabusFromValues(values),
		PrescribedMinutes: nonNegativeInt(values.Get("prescribed_minutes"), 0),
		CoverageSequence:  positiveInt(values.Get("coverage_sequence"), 0),
	}
}

func impactInSyllabusFromValues(values url.Values) bool {
	switch strings.TrimSpace(strings.ToLower(values.Get("syllabus_status"))) {
	case "in_syllabus":
		return true
	case "out_of_syllabus":
		return false
	default:
		return values.Get("is_in_syllabus") != "false"
	}
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

func nonNegativeInt(value string, fallback int) int {
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || parsed < 0 {
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
		"curriculumConfigNewURL": func(query curriculumconfig.ListQuery) string {
			return "/admin/curriculum-config/new?" + encodeListQuery(query)
		},
		"curriculumConfigExportURL": func(query curriculumconfig.ListQuery) string {
			return "/admin/curriculum-config/export?" + encodeListQuery(query)
		},
		"curriculumConfigEditURL": func(query curriculumconfig.ListQuery, id int64) string {
			return "/admin/curriculum-config/edit?id=" + strconv.FormatInt(id, 10) + "&" + encodeListQuery(query)
		},
		"curriculumConfigRemoveURL": func(query curriculumconfig.ListQuery, id int64) string {
			return "/admin/curriculum-config/remove?id=" + strconv.FormatInt(id, 10) + "&" + encodeListQuery(query)
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

func writeAddPanel(w http.ResponseWriter, query curriculumconfig.ListQuery, warnings []curriculumconfig.Warning, impact curriculumconfig.ImpactResult) {
	query = curriculumconfig.NormalizeListQuery(query)
	fmt.Fprint(w, `<section id="curriculum-config-add-panel" class="space-y-4">`)
	fmt.Fprint(w, `<header><h2 class="text-lg font-semibold text-ink">Add LMS Chapter Exam Config</h2></header>`)
	writeAppliedFilterMirror(w, query)
	writeWarningAndImpact(w, warnings, impact)
	fmt.Fprint(w, `<form id="curriculum-config-create-form" hx-post="/admin/curriculum-config/create" hx-target="#curriculum-config-add-panel" hx-swap="outerHTML" class="space-y-3">`)
	writeAppliedFilterHiddenFields(w, query)
	fmt.Fprintf(w, `<input type="hidden" name="exam_track" value="%s">`, html.EscapeString(query.ExamTrack))
	fmt.Fprint(w, `<label class="block text-sm font-medium text-ink" for="curriculum-config-new-chapter">Chapter</label>`)
	fmt.Fprint(w, `<input id="curriculum-config-new-chapter" class="form-control w-full" name="search" hx-get="/admin/curriculum-config/chapter-options" hx-target="#curriculum-config-chapter-options" hx-swap="innerHTML" hx-include="#curriculum-config-create-form">`)
	fmt.Fprint(w, `<div id="curriculum-config-chapter-options" class="text-sm text-ink-muted"></div>`)
	fmt.Fprint(w, `<label class="block text-sm font-medium text-ink" for="curriculum-config-new-syllabus-status">Syllabus status</label>`)
	fmt.Fprint(w, `<select id="curriculum-config-new-syllabus-status" class="form-select w-full" name="syllabus_status" hx-get="/admin/curriculum-config/impact" hx-target="#curriculum-config-impact-preview" hx-swap="innerHTML" hx-include="#curriculum-config-create-form"><option value="in_syllabus" selected>In syllabus</option><option value="out_of_syllabus">Out of syllabus</option></select>`)
	fmt.Fprint(w, `<label class="block text-sm font-medium text-ink" for="curriculum-config-new-minutes">Prescribed minutes</label>`)
	fmt.Fprint(w, `<input id="curriculum-config-new-minutes" class="form-control w-full" name="prescribed_minutes" inputmode="numeric" value="0" hx-get="/admin/curriculum-config/impact" hx-target="#curriculum-config-impact-preview" hx-swap="innerHTML" hx-include="#curriculum-config-create-form">`)
	fmt.Fprint(w, `<label class="block text-sm font-medium text-ink" for="curriculum-config-new-coverage">Coverage order</label>`)
	fmt.Fprint(w, `<input id="curriculum-config-new-coverage" class="form-control w-full" name="coverage_sequence" inputmode="numeric" hx-get="/admin/curriculum-config/impact" hx-target="#curriculum-config-impact-preview" hx-swap="innerHTML" hx-include="#curriculum-config-create-form">`)
	fmt.Fprint(w, `<div id="curriculum-config-impact-preview"></div>`)
	fmt.Fprint(w, `<div class="flex justify-end"><button type="submit" class="btn-primary btn-sm">Create config</button></div>`)
	fmt.Fprint(w, `</form></section>`)
}

func writeEditPanel(w http.ResponseWriter, query curriculumconfig.ListQuery, row *curriculumconfig.ListRow, warnings []curriculumconfig.Warning, impact curriculumconfig.ImpactResult) {
	query = curriculumconfig.NormalizeListQuery(query)
	if row == nil {
		writeUpdateError(w, http.StatusNotFound, "Could not load LMS Chapter Exam Config", "LMS Chapter Exam Config does not exist")
		return
	}
	fmt.Fprint(w, `<section id="curriculum-config-edit-panel" class="space-y-4">`)
	fmt.Fprint(w, `<header><h2 class="text-lg font-semibold text-ink">Edit LMS Chapter Exam Config</h2></header>`)
	fmt.Fprintf(w, `<div class="rounded-md border border-border bg-bg-card-alt p-3 text-sm"><div class="font-medium">%s</div><div>%s</div><div>Chapter ID %d</div><div>Grade %s · %s</div><div>%s</div></div>`, html.EscapeString(row.ChapterCode), html.EscapeString(row.ChapterName), row.ChapterID, html.EscapeString(row.Grade), html.EscapeString(row.Subject), html.EscapeString(examTrackLabelForView(row.ExamTrack)))
	writeWarningAndImpact(w, warnings, impact)
	fmt.Fprint(w, `<form id="curriculum-config-edit-form" hx-post="/admin/curriculum-config/update" hx-target="#curriculum-config-edit-panel" hx-swap="outerHTML" class="space-y-3">`)
	writeAppliedFilterHiddenFields(w, query)
	fmt.Fprintf(w, `<input type="hidden" name="id" value="%d">`, row.ID)
	fmt.Fprintf(w, `<input type="hidden" name="lock_token" value="%s">`, html.EscapeString(row.LockToken))
	fmt.Fprint(w, `<label class="block text-sm font-medium text-ink" for="curriculum-config-edit-syllabus-status">Syllabus status</label>`)
	fmt.Fprint(w, `<select id="curriculum-config-edit-syllabus-status" class="form-select w-full" name="syllabus_status">`)
	fmt.Fprintf(w, `<option value="in_syllabus" %s>In syllabus</option>`, selectedAttr(row.IsInSyllabus))
	fmt.Fprintf(w, `<option value="out_of_syllabus" %s>Out of syllabus</option>`, selectedAttr(!row.IsInSyllabus))
	fmt.Fprint(w, `</select>`)
	fmt.Fprint(w, `<label class="block text-sm font-medium text-ink" for="curriculum-config-edit-minutes">Prescribed minutes</label>`)
	fmt.Fprintf(w, `<input id="curriculum-config-edit-minutes" class="form-control w-full" name="prescribed_minutes" inputmode="numeric" value="%d">`, row.PrescribedMinutes)
	fmt.Fprint(w, `<label class="block text-sm font-medium text-ink" for="curriculum-config-edit-coverage">Coverage order</label>`)
	fmt.Fprintf(w, `<input id="curriculum-config-edit-coverage" class="form-control w-full" name="coverage_sequence" inputmode="numeric" value="%d" hx-get="/admin/curriculum-config/impact" hx-target="#curriculum-config-impact-preview" hx-swap="innerHTML" hx-include="#curriculum-config-edit-form" hx-vals='{"config_id":%d,"chapter_id":%d,"exam_track":"%s"}'>`, row.CoverageSequence, row.ID, row.ChapterID, html.EscapeString(row.ExamTrack))
	fmt.Fprint(w, `<div id="curriculum-config-impact-preview"></div>`)
	fmt.Fprint(w, `<div class="flex justify-end"><button type="submit" class="btn-primary btn-sm">Update config</button></div>`)
	fmt.Fprint(w, `</form></section>`)
}

func writeRemovePanel(w http.ResponseWriter, query curriculumconfig.ListQuery, row *curriculumconfig.ListRow, warnings []curriculumconfig.Warning, impact curriculumconfig.ImpactResult) {
	query = curriculumconfig.NormalizeListQuery(query)
	if row == nil {
		writeRemoveError(w, http.StatusNotFound, "Could not load LMS Chapter Exam Config", "LMS Chapter Exam Config does not exist")
		return
	}
	fmt.Fprint(w, `<section id="curriculum-config-remove-panel" class="space-y-4">`)
	fmt.Fprint(w, `<header><h2 class="text-lg font-semibold text-ink">Remove from syllabus</h2></header>`)
	fmt.Fprintf(w, `<div class="rounded-md border border-border bg-bg-card-alt p-3 text-sm"><div class="font-medium">%s</div><div>%s</div><div>Chapter ID %d</div><div>Grade %s · %s</div><div>%s</div><div>Coverage order %d</div></div>`, html.EscapeString(row.ChapterCode), html.EscapeString(row.ChapterName), row.ChapterID, html.EscapeString(row.Grade), html.EscapeString(row.Subject), html.EscapeString(examTrackLabelForView(row.ExamTrack)), row.CoverageSequence)
	writeWarningAndImpact(w, warnings, impact)
	fmt.Fprint(w, `<form id="curriculum-config-remove-form" hx-post="/admin/curriculum-config/remove-from-syllabus" hx-target="#curriculum-config-remove-panel" hx-swap="outerHTML" class="space-y-3">`)
	writeAppliedFilterHiddenFields(w, query)
	fmt.Fprintf(w, `<input type="hidden" name="id" value="%d">`, row.ID)
	fmt.Fprintf(w, `<input type="hidden" name="lock_token" value="%s">`, html.EscapeString(row.LockToken))
	fmt.Fprint(w, `<p class="text-sm text-ink-muted">This will set the row out of syllabus, force prescribed minutes to 0, and preserve coverage order.</p>`)
	fmt.Fprint(w, `<div class="flex justify-end"><button type="submit" class="btn-danger btn-sm">Remove from syllabus</button></div>`)
	fmt.Fprint(w, `</form></section>`)
}

func selectedAttr(selected bool) string {
	if selected {
		return "selected"
	}
	return ""
}

func writeAppliedFilterMirror(w http.ResponseWriter, query curriculumconfig.ListQuery) {
	fmt.Fprint(w, `<form id="curriculum-config-add-applied-filters" class="hidden" aria-hidden="true">`)
	fmt.Fprintf(w, `<input type="hidden" name="exam_track" value="%s">`, html.EscapeString(query.ExamTrack))
	fmt.Fprintf(w, `<input type="hidden" name="grade" value="%s">`, html.EscapeString(query.Grade))
	fmt.Fprintf(w, `<input type="hidden" name="subject" value="%s">`, html.EscapeString(query.Subject))
	fmt.Fprintf(w, `<input type="hidden" name="search" value="%s">`, html.EscapeString(query.Search))
	fmt.Fprintf(w, `<input type="hidden" name="chapter_id" value="%s">`, html.EscapeString(query.ChapterID))
	fmt.Fprintf(w, `<input type="hidden" name="syllabus_status" value="%s">`, html.EscapeString(query.SyllabusStatus))
	fmt.Fprintf(w, `<input type="hidden" name="page" value="%d">`, query.Page)
	fmt.Fprintf(w, `<input type="hidden" name="limit" value="%d">`, query.Limit)
	fmt.Fprintf(w, `<input type="hidden" name="sort" value="%s">`, html.EscapeString(query.Sort))
	fmt.Fprintf(w, `<input type="hidden" name="dir" value="%s">`, html.EscapeString(query.Direction))
	fmt.Fprint(w, `</form>`)
}

func writeAppliedFilterHiddenFields(w http.ResponseWriter, query curriculumconfig.ListQuery) {
	query = curriculumconfig.NormalizeListQuery(query)
	fmt.Fprintf(w, `<input type="hidden" name="filter_exam_track" value="%s">`, html.EscapeString(query.ExamTrack))
	fmt.Fprintf(w, `<input type="hidden" name="filter_grade" value="%s">`, html.EscapeString(query.Grade))
	fmt.Fprintf(w, `<input type="hidden" name="filter_subject" value="%s">`, html.EscapeString(query.Subject))
	fmt.Fprintf(w, `<input type="hidden" name="filter_search" value="%s">`, html.EscapeString(query.Search))
	fmt.Fprintf(w, `<input type="hidden" name="filter_chapter_id" value="%s">`, html.EscapeString(query.ChapterID))
	fmt.Fprintf(w, `<input type="hidden" name="filter_syllabus_status" value="%s">`, html.EscapeString(query.SyllabusStatus))
	fmt.Fprintf(w, `<input type="hidden" name="filter_page" value="%d">`, query.Page)
	fmt.Fprintf(w, `<input type="hidden" name="filter_limit" value="%d">`, query.Limit)
	fmt.Fprintf(w, `<input type="hidden" name="filter_sort" value="%s">`, html.EscapeString(query.Sort))
	fmt.Fprintf(w, `<input type="hidden" name="filter_dir" value="%s">`, html.EscapeString(query.Direction))
}

func writeWarningAndImpact(w http.ResponseWriter, warnings []curriculumconfig.Warning, impact curriculumconfig.ImpactResult) {
	if len(warnings) > 0 {
		fmt.Fprint(w, `<div id="curriculum-config-warnings" class="rounded-md border border-warning bg-warning-subtle p-3"><ul class="list-disc pl-5 text-sm">`)
		for _, warning := range warnings {
			fmt.Fprintf(w, `<li data-warning-code="%s">%s</li>`, html.EscapeString(warning.Code), html.EscapeString(warning.Message))
		}
		fmt.Fprint(w, `</ul></div>`)
	}
	if impact.Unavailable {
		fmt.Fprint(w, `<div id="curriculum-config-impact" role="status">Impact counts unavailable</div>`)
		return
	}
	if impact.SummaryRows != 0 || impact.ActiveLogs != 0 || impact.ChapterCompletions != 0 {
		fmt.Fprintf(w, `<div id="curriculum-config-impact" role="status"><span>Summary rows: %d</span><span>Active logs: %d</span><span>Chapter completions: %d</span></div>`, impact.SummaryRows, impact.ActiveLogs, impact.ChapterCompletions)
	}
}

func writeChapterOptions(w http.ResponseWriter, options []curriculumconfig.ChapterOption) {
	if len(options) == 0 {
		fmt.Fprint(w, `<div role="status">No chapters match the current search.</div>`)
		return
	}
	fmt.Fprint(w, `<div class="space-y-2">`)
	for _, option := range options {
		fmt.Fprintf(w, `<label class="block rounded-md border border-border p-3"><input type="radio" name="chapter_id" value="%d" hx-get="/admin/curriculum-config/impact" hx-target="#curriculum-config-impact-preview" hx-swap="innerHTML" hx-include="#curriculum-config-create-form"> <span class="font-medium">%s</span> <span>%s</span> <span>Grade %s</span> <span>%s</span> <span>%d topics</span>`, option.ChapterID, html.EscapeString(option.ChapterCode), html.EscapeString(option.ChapterName), html.EscapeString(option.Grade), html.EscapeString(option.Subject), option.TopicCount)
		if option.HasDuplicateConfig {
			fmt.Fprintf(w, `<div class="text-sm text-warning">Chapter already has a %s config.</div>`, html.EscapeString(examTrackLabelForView(option.ExistingExamTrack)))
		}
		if option.HasZeroTopicWarning {
			fmt.Fprint(w, `<div class="text-sm text-warning">This chapter has no topics.</div>`)
		}
		fmt.Fprint(w, `</label>`)
	}
	fmt.Fprint(w, `</div>`)
}

func examTrackLabelForView(value string) string {
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
}

func writeCreateSuccess(w http.ResponseWriter, result curriculumconfig.MutationResult, table tableViewData) {
	fmt.Fprint(w, `<section id="curriculum-config-create-result" class="space-y-3">`)
	fmt.Fprint(w, `<div role="status" class="rounded-md border border-success bg-success-subtle p-3">Created LMS Chapter Exam Config.</div>`)
	writeWarningAndImpact(w, result.Warnings, result.Impact)
	fmt.Fprint(w, `</section>`)
	fmt.Fprint(w, `<div id="curriculum-config-table" hx-swap-oob="true" class="rounded-md border border-border bg-bg-card p-4">`)
	views.ExecuteTemplate(curriculumConfigTableTemplate, w, table, curriculumConfigFuncMap())
	fmt.Fprint(w, `</div>`)
}

func writeUpdateSuccess(w http.ResponseWriter, result curriculumconfig.MutationResult, table tableViewData) {
	fmt.Fprint(w, `<section id="curriculum-config-update-result" class="space-y-3">`)
	fmt.Fprint(w, `<div role="status" class="rounded-md border border-success bg-success-subtle p-3">Updated LMS Chapter Exam Config.</div>`)
	writeWarningAndImpact(w, result.Warnings, result.Impact)
	fmt.Fprint(w, `</section>`)
	fmt.Fprint(w, `<div id="curriculum-config-table" hx-swap-oob="true" class="rounded-md border border-border bg-bg-card p-4">`)
	views.ExecuteTemplate(curriculumConfigTableTemplate, w, table, curriculumConfigFuncMap())
	fmt.Fprint(w, `</div>`)
}

func writeRemoveSuccess(w http.ResponseWriter, result curriculumconfig.MutationResult, table tableViewData) {
	fmt.Fprint(w, `<section id="curriculum-config-remove-result" class="space-y-3">`)
	fmt.Fprint(w, `<div role="status" class="rounded-md border border-success bg-success-subtle p-3">Removed LMS Chapter Exam Config from syllabus.</div>`)
	writeWarningAndImpact(w, result.Warnings, result.Impact)
	fmt.Fprint(w, `</section>`)
	fmt.Fprint(w, `<div id="curriculum-config-table" hx-swap-oob="true" class="rounded-md border border-border bg-bg-card p-4">`)
	views.ExecuteTemplate(curriculumConfigTableTemplate, w, table, curriculumConfigFuncMap())
	fmt.Fprint(w, `</div>`)
}

func writeMutationError(w http.ResponseWriter, err error) {
	message := err.Error()
	status := http.StatusUnprocessableEntity
	switch {
	case strings.Contains(message, "duplicate LMS Chapter Exam Config"):
		status = http.StatusConflict
	case strings.Contains(message, "does not exist"):
		status = http.StatusUnprocessableEntity
	}
	w.WriteHeader(status)
	fmt.Fprintf(w, `<section id="curriculum-config-add-panel" role="alert"><h2>Could not create LMS Chapter Exam Config</h2><p>%s</p></section>`, html.EscapeString(message))
}

func writeEditMutationError(w http.ResponseWriter, err error) {
	if errors.Is(err, curriculumconfig.ErrStaleLock) || strings.Contains(err.Error(), "lock token") {
		writeUpdateError(w, http.StatusConflict, "Could not update LMS Chapter Exam Config", "This LMS Chapter Exam Config changed while you were editing. Reload the row and try again.")
		return
	}
	writeUpdateError(w, http.StatusUnprocessableEntity, "Could not update LMS Chapter Exam Config", err.Error())
}

func writeRemoveMutationError(w http.ResponseWriter, err error) {
	if errors.Is(err, curriculumconfig.ErrStaleLock) || strings.Contains(err.Error(), "lock token") {
		writeRemoveError(w, http.StatusConflict, "Could not remove LMS Chapter Exam Config", "This LMS Chapter Exam Config changed while you were removing it. Reload the row and try again.")
		return
	}
	writeRemoveError(w, http.StatusUnprocessableEntity, "Could not remove LMS Chapter Exam Config", err.Error())
}

func writeUpdateError(w http.ResponseWriter, status int, title string, message string) {
	w.WriteHeader(status)
	fmt.Fprintf(w, `<section id="curriculum-config-edit-panel" role="alert"><h2>%s</h2><p>%s</p></section>`, html.EscapeString(title), html.EscapeString(message))
}

func writeRemoveError(w http.ResponseWriter, status int, title string, message string) {
	w.WriteHeader(status)
	fmt.Fprintf(w, `<section id="curriculum-config-remove-panel" role="alert"><h2>%s</h2><p>%s</p></section>`, html.EscapeString(title), html.EscapeString(message))
}
