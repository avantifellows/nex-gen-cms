package handlers

import (
	"net/http"

	"github.com/avantifellows/nex-gen-cms/internal/models"
	"github.com/avantifellows/nex-gen-cms/internal/services"
)

const examsEndPoint = "exam"
const examsKey = "exams"
const examsTemplate = "exams.html"

type ExamsHandler struct {
	service *services.Service[models.Exam]
}

func NewExamsHandler(service *services.Service[models.Exam]) *ExamsHandler {
	return &ExamsHandler{
		service: service,
	}
}

func (h *ExamsHandler) GetExams(responseWriter http.ResponseWriter, _ *http.Request) {
	renderEntityList(responseWriter, h.service, examsEndPoint, examsKey, examsTemplate, "exams")
}
