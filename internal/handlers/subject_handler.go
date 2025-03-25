package handlers

import (
	"fmt"
	"html/template"
	"net/http"

	"github.com/avantifellows/nex-gen-cms/internal/models"
	local_repo "github.com/avantifellows/nex-gen-cms/internal/repositories/local"
	"github.com/avantifellows/nex-gen-cms/internal/services"
)

const subjectsEndPoint = "/subject"
const subjectsKey = "subjects"
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
	subjects, err := h.service.GetList(subjectsEndPoint, subjectsKey, false, false)
	if err != nil {
		http.Error(responseWriter, fmt.Sprintf("Error fetching subjects: %v", err), http.StatusInternalServerError)
		return
	}

	// Load subjects.html
	local_repo.ExecuteTemplate(subjectsTemplate, responseWriter, subjects, template.FuncMap{
		"getName": getName,
	})
}

func getName(s models.Subject, lang string) string {
	return s.GetNameByLang(lang)
}
