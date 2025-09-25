package handlers

import (
	"fmt"
	"net/http"

	"github.com/avantifellows/nex-gen-cms/internal/models"
	"github.com/avantifellows/nex-gen-cms/internal/services"
	"github.com/avantifellows/nex-gen-cms/internal/views"
)

const getGradesEndPoint = "grade"
const gradesKey = "grades"
const gradesTemplate = "grades.html"

type GradesHandler struct {
	service *services.Service[models.Grade]
}

func NewGradesHandler(service *services.Service[models.Grade]) *GradesHandler {
	return &GradesHandler{
		service: service,
	}
}

func (h *GradesHandler) GetGrades(responseWriter http.ResponseWriter, request *http.Request) {
	grades, err := h.service.GetList(getGradesEndPoint, gradesKey, false, false)
	if err != nil {
		http.Error(responseWriter, fmt.Sprintf("Error fetching grades: %v", err), http.StatusInternalServerError)
		return
	}

	// Load grades.html
	views.ExecuteTemplate(gradesTemplate, responseWriter, grades, nil)
}
