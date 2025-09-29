package handlers

import (
	"fmt"
	"net/http"

	"github.com/avantifellows/nex-gen-cms/internal/models"
	"github.com/avantifellows/nex-gen-cms/internal/services"
	"github.com/avantifellows/nex-gen-cms/internal/views"
)

const QUERY_PARAM_CURRICULUM_ID = "curriculum_id"

const getCurriculumsEndPoint = "curriculum"
const curriculumsKey = "curriculums"
const curriculumsTemplate = "curriculums.html"

type CurriculumsHandler struct {
	service *services.Service[models.Curriculum]
}

func NewCurriculumsHandler(service *services.Service[models.Curriculum]) *CurriculumsHandler {
	return &CurriculumsHandler{
		service: service,
	}
}

func (h *CurriculumsHandler) GetCurriculums(responseWriter http.ResponseWriter, request *http.Request) {
	curriculums, err := h.service.GetList(getCurriculumsEndPoint, curriculumsKey, false, false)
	if err != nil {
		http.Error(responseWriter, fmt.Sprintf("Error fetching curriculums: %v", err), http.StatusInternalServerError)
		return
	}

	views.ExecuteTemplate(curriculumsTemplate, responseWriter, curriculums, nil)
}
