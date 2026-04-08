package handlers

import (
	"fmt"
	"net/http"
	"text/template"

	"github.com/avantifellows/nex-gen-cms/internal/models"
	"github.com/avantifellows/nex-gen-cms/internal/services"
	"github.com/avantifellows/nex-gen-cms/internal/views"
)

const resourcesCurriculumListEndPoint = "resources/curriculum"
const resourcesKey = "resources"

const resourcesTemplate = "resources.html"

type ResourcesHandler struct {
	service *services.Service[models.Resource]
}

func NewResourcesHandler(service *services.Service[models.Resource]) *ResourcesHandler {
	return &ResourcesHandler{
		service: service,
	}
}

func (h *ResourcesHandler) GetResources(responseWriter http.ResponseWriter, request *http.Request) {
	queryParams := "?curriculum_id=1&grade_id=3&chapter_id=1"
	resources, err := h.service.GetList(resourcesCurriculumListEndPoint+queryParams, resourcesKey, false, false)
	if err != nil {
		http.Error(responseWriter, fmt.Sprintf("Error fetching resources: %v", err), http.StatusInternalServerError)
		return
	}

	views.ExecuteTemplate(resourcesTemplate, responseWriter, resources, template.FuncMap{
		"getName": getResourceName,
	})
}

func getResourceName(r models.Resource, lang string) string {
	return r.GetNameByLang(lang)
}
