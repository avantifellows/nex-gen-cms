package handlers

import (
	"fmt"
	"html/template"
	"net/http"

	"github.com/avantifellows/nex-gen-cms/internal/handlers/handlerutils"
	"github.com/avantifellows/nex-gen-cms/internal/models"
	"github.com/avantifellows/nex-gen-cms/internal/services"
	"github.com/avantifellows/nex-gen-cms/internal/views"
)

const subjectsTemplate = "subjects.html"

type SubjectsHandler struct {
	service *services.Service[models.Subject]
}

func NewSubjectsHandler(service *services.Service[models.Subject]) *SubjectsHandler {
	return &SubjectsHandler{
		service: service,
	}
}

func (h *SubjectsHandler) GetSubjects(responseWriter http.ResponseWriter, request *http.Request) {
	subjects, err := h.service.GetList(handlerutils.SubjectsEndPoint, handlerutils.SubjectsKey, false, false)
	if err != nil {
		http.Error(responseWriter, fmt.Sprintf("Error fetching subjects: %v", err), http.StatusInternalServerError)
		return
	}

	views.ExecuteTemplate(subjectsTemplate, responseWriter, subjects, template.FuncMap{
		"getName": getSubjectName,
	})
}

func getSubjectName(s models.Subject, lang string) string {
	return s.GetNameByLang(lang)
}

func getParentSubjectName(s models.Subject, lang string) string {
	if s.ParentID != 0 {
		return s.GetParentNameByLang(lang)
	} else {
		return s.GetNameByLang(lang)
	}
}

func getParentSubjectId(s models.Subject) int8 {
	if s.ParentID != 0 {
		return s.ParentID
	} else {
		return s.ID
	}
}
