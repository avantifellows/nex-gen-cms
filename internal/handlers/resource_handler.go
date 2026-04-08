package handlers

import (
	"fmt"
	"net/http"
	"strings"
	"text/template"

	"github.com/avantifellows/nex-gen-cms/internal/models"
	"github.com/avantifellows/nex-gen-cms/internal/services"
	"github.com/avantifellows/nex-gen-cms/internal/views"
	"github.com/avantifellows/nex-gen-cms/utils"
)

const resourcesCurriculumListEndPoint = "resources/curriculum"
const resourcesKey = "resources"

const resourcesTemplate = "resources.html"
const resourceRowTemplate = "resource_row.html"

type ResourcesHandler struct {
	service *services.Service[models.Resource]
}

func NewResourcesHandler(service *services.Service[models.Resource]) *ResourcesHandler {
	return &ResourcesHandler{
		service: service,
	}
}

func (h *ResourcesHandler) GetResources(responseWriter http.ResponseWriter, request *http.Request) {
	urlVals := request.URL.Query()
	curriculumIdStr := urlVals.Get("curriculum-dropdown")
	gradeIdStr := urlVals.Get("grade-dropdown")
	chapterIdStr := urlVals.Get("chapter_id")

	curriculumId, err := utils.StringToIntType[int16](curriculumIdStr)
	if err != nil {
		http.Error(responseWriter, "Invalid Curriculum ID", http.StatusBadRequest)
		return
	}
	gradeId, err := utils.StringToIntType[int8](gradeIdStr)
	if err != nil {
		http.Error(responseWriter, "Invalid Grade ID", http.StatusBadRequest)
		return
	}
	chapterId, err := utils.StringToIntType[int16](chapterIdStr)
	if err != nil {
		http.Error(responseWriter, "Invalid Chapter ID", http.StatusBadRequest)
		return
	}

	queryParams := fmt.Sprintf("?curriculum_id=%d&grade_id=%d&chapter_id=%d", curriculumId, gradeId, chapterId)
	resources, err := h.service.GetList(resourcesCurriculumListEndPoint+queryParams, resourcesKey, false, false)
	if err != nil {
		http.Error(responseWriter, fmt.Sprintf("Error fetching resources: %v", err), http.StatusInternalServerError)
		return
	}

	filteredResources := make([]*models.Resource, 0, len(*resources))
	// remove test & problem resources because we are already managing those via separate tabs
	for _, resource := range *resources {
		resourceType := strings.ToLower(resource.Type)
		if resourceType == "problem" || resourceType == "test" {
			continue
		}
		filteredResources = append(filteredResources, resource)
	}

	views.ExecuteTemplate(resourceRowTemplate, responseWriter, &filteredResources, template.FuncMap{
		"getName": getResourceName,
	})
}

func getResourceName(r models.Resource, lang string) string {
	return r.GetNameByLang(lang)
}
