package handlers

import (
	"net/http"

	"github.com/avantifellows/nex-gen-cms/internal/models"
	"github.com/avantifellows/nex-gen-cms/internal/services"
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

func (h *GradesHandler) GetGrades(responseWriter http.ResponseWriter, _ *http.Request) {
	renderEntityList(responseWriter, h.service, getGradesEndPoint, gradesKey, gradesTemplate, "grades")
}
