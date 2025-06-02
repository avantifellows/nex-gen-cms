package handlers

import (
	"fmt"
	"net/http"

	"github.com/avantifellows/nex-gen-cms/internal/models"
	local_repo "github.com/avantifellows/nex-gen-cms/internal/repositories/local"
	"github.com/avantifellows/nex-gen-cms/internal/services"
)

const skillsKey = "skills"

const skillsEndPoint = "/skill"

const skillsTemplate = "skills.html"

type SkillsHandler struct {
	service *services.Service[models.Skill]
}

func NewSkillsHandler(service *services.Service[models.Skill]) *SkillsHandler {
	return &SkillsHandler{
		service: service,
	}
}

func (h *SkillsHandler) GetSkills(responseWriter http.ResponseWriter, request *http.Request) {
	skills, err := h.service.GetList(skillsEndPoint, skillsKey, false, false)
	if err != nil {
		http.Error(responseWriter, fmt.Sprintf("Error fetching skills: %v", err), http.StatusInternalServerError)
		return
	}

	local_repo.ExecuteTemplate(skillsTemplate, responseWriter, skills, nil)
}
