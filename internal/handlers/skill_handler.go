package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/avantifellows/nex-gen-cms/internal/models"
	"github.com/avantifellows/nex-gen-cms/internal/services"
	"github.com/avantifellows/nex-gen-cms/internal/views"
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
	selectedSkillIds := request.URL.Query().Get("selected_skill_ids")
	selectedIDs := make(map[int]bool)

	if selectedSkillIds != "" {
		ids := strings.Split(selectedSkillIds, ",")
		for _, idStr := range ids {
			id, err := strconv.Atoi(idStr)
			if err == nil {
				selectedIDs[id] = true
			}
		}
	}

	skills, err := h.service.GetList(skillsEndPoint, skillsKey, false, false)
	if err != nil {
		http.Error(responseWriter, fmt.Sprintf("Error fetching skills: %v", err), http.StatusInternalServerError)
		return
	}

	// Wrap both skills & selected skill ids in a struct
	data := struct {
		Skills           *[]*models.Skill
		SelectedSkillIds map[int]bool
	}{
		Skills:           skills,
		SelectedSkillIds: selectedIDs,
	}

	views.ExecuteTemplate(skillsTemplate, responseWriter, data, nil)
}
