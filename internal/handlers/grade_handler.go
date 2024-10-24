package handlers

import (
	"fmt"
	"net/http"

	"github.com/avantifellows/nex-gen-cms/internal/models"
	local_repo "github.com/avantifellows/nex-gen-cms/internal/repositories/local"
	"github.com/avantifellows/nex-gen-cms/internal/services"
)

const getGradesEndPoint = "/grade"
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

func (h *GradesHandler) GetGrades(w http.ResponseWriter, r *http.Request) {
	grades, err := h.service.GetList(getGradesEndPoint, gradesKey, false)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error fetching grades: %v", err), http.StatusInternalServerError)
		return
	}

	// Load grades.html
	local_repo.ExecuteTemplate(gradesTemplate, w, grades)
}
