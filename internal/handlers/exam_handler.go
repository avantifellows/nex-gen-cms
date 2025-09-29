package handlers

import (
	"fmt"
	"net/http"

	"github.com/avantifellows/nex-gen-cms/internal/models"
	"github.com/avantifellows/nex-gen-cms/internal/services"
	"github.com/avantifellows/nex-gen-cms/internal/views"
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

func (h *ExamsHandler) GetExams(responseWriter http.ResponseWriter, request *http.Request) {
	exams, err := h.service.GetList(examsEndPoint, examsKey, false, false)
	if err != nil {
		http.Error(responseWriter, fmt.Sprintf("Error fetching exams: %v", err), http.StatusInternalServerError)
		return
	}

	views.ExecuteTemplate(examsTemplate, responseWriter, exams, nil)
}
