package handlers

import (
	"fmt"
	"html/template"
	"net/http"

	"github.com/avantifellows/nex-gen-cms/internal/handlers/handlerutils"
	"github.com/avantifellows/nex-gen-cms/internal/models"
	local_repo "github.com/avantifellows/nex-gen-cms/internal/repositories/local"
	"github.com/avantifellows/nex-gen-cms/internal/services"
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

	local_repo.ExecuteTemplate(subjectsTemplate, responseWriter, subjects, template.FuncMap{
		"getName": getSubjectName,
	})
}

func getSubjectName(s models.Subject, lang string) string {
	return s.GetNameByLang(lang)
}
