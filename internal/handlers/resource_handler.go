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
const resourceEndPoint = "resource"
const resourcesKey = "resources"

const resourcesTemplate = "resources.html"
const resourceRowTemplate = "resource_row.html"
const editResourceTemplate = "edit_resource.html"

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

func (h *ResourcesHandler) EditResource(responseWriter http.ResponseWriter, request *http.Request) {
	resourceIdStr := request.URL.Query().Get("id")
	resourceId, err := utils.StringToIntType[int32](resourceIdStr)
	if err != nil {
		http.Error(responseWriter, "Invalid Resource ID", http.StatusBadRequest)
		return
	}

	selectedResourcePtr, err := h.service.GetObject(resourceIdStr,
		func(resource *models.Resource) bool {
			return resource.ID == int(resourceId)
		}, resourcesKey, resourceEndPoint)
	if err != nil {
		http.Error(responseWriter, fmt.Sprintf("Error fetching resource: %v", err), http.StatusInternalServerError)
		return
	}

	views.ExecuteTemplate(editResourceTemplate, responseWriter, selectedResourcePtr, template.FuncMap{
		"getName": getResourceName,
	})
}

func (h *ResourcesHandler) UpdateResource(responseWriter http.ResponseWriter, request *http.Request) {
	resourceIdStr := request.FormValue("id")
	resourceId, err := utils.StringToIntType[int32](resourceIdStr)
	if err != nil {
		http.Error(responseWriter, "Invalid Resource ID", http.StatusBadRequest)
		return
	}

	resourceName := request.FormValue("name")
	resourceCode := request.FormValue("code")
	resourceType := request.FormValue("type")
	resourceSubtype := request.FormValue("subtype")
	srcLink := request.FormValue("src_link")

	dummyResourcePtr := &models.Resource{}
	resourceMap := dummyResourcePtr.BuildMap(resourceCode, resourceName, resourceType, resourceSubtype, srcLink)

	_, err = h.service.UpdateObject(resourceIdStr, resourceEndPoint, resourceMap, resourcesKey,
		func(resource *models.Resource) bool {
			return resource.ID == int(resourceId)
		})
	if err != nil {
		http.Error(responseWriter, fmt.Sprintf("Error updating resource: %v", err), http.StatusInternalServerError)
		return
	}

	views.ExecuteTemplate(updateSuccessTemplate, responseWriter, "Resource", nil)
}

func getResourceName(r models.Resource, lang string) string {
	return r.GetNameByLang(lang)
}
