package handlers

import (
	"fmt"
	"net/http"

	"github.com/avantifellows/nex-gen-cms/internal/models"
	local_repo "github.com/avantifellows/nex-gen-cms/internal/repositories/local"
	"github.com/avantifellows/nex-gen-cms/internal/services"
)

const getSubjectsEndPoint = "/subject"
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

func (h *SubjectsHandler) GetSubjects(w http.ResponseWriter, r *http.Request) {
	subjects, err := h.service.GetList(getSubjectsEndPoint, subjectsKey, false)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error fetching subjects: %v", err), http.StatusInternalServerError)
		return
	}

	// Load subjects.html
	local_repo.ExecuteTemplate(subjectsTemplate, w, subjects)
}
