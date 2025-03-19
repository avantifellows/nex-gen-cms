package handlers

import (
	"fmt"
	"net/http"

	"github.com/avantifellows/nex-gen-cms/internal/models"
	local_repo "github.com/avantifellows/nex-gen-cms/internal/repositories/local"
	"github.com/avantifellows/nex-gen-cms/internal/services"
)

const getCurriculumsEndPoint = "/curriculum"
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

	// Load curriculums.html
	local_repo.ExecuteTemplate(curriculumsTemplate, responseWriter, curriculums, nil)
}
