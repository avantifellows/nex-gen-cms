package handlers

import (
	"fmt"
	"html"
	"log"
	"net/http"

	"github.com/avantifellows/nex-gen-cms/internal/curriculumconfig"
	"github.com/avantifellows/nex-gen-cms/internal/views"
)

const curriculumConfigTemplate = "curriculum_config.html"

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
	data := map[string]any{
		"Readiness": readiness,
	}
	views.ExecuteTemplates(w, data, nil, baseTemplate, curriculumConfigTemplate)
}

func (h *CurriculumConfigHandler) Table(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodGet) {
		return
	}
	if _, ok := h.ensureReady(w, r, false); !ok {
		return
	}
	h.placeholder(w, "Curriculum Config table is not available in this slice")
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
		views.ExecuteTemplates(w, data, nil, baseTemplate, curriculumConfigTemplate)
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
