package handlers

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"text/template"

	"github.com/avantifellows/nex-gen-cms/internal/dto"
	"github.com/avantifellows/nex-gen-cms/internal/models"
	"github.com/avantifellows/nex-gen-cms/internal/services"
	"github.com/avantifellows/nex-gen-cms/internal/views"
	"github.com/avantifellows/nex-gen-cms/utils"
)

const resourcesEndPoint = "resource"
const resourcesCurriculumEndPoint = "resources/curriculum"
const moveResourceEndPoint = "resources/move"
const resourcesKey = "resources"

const resourcesTemplate = "resources.html"
const resourceRowTemplate = "resource_row.html"
const editResourceTemplate = "edit_resource.html"
const addResourceTemplate = "add_resource.html"
const moveResourcesTemplate = "move_resources_modal.html"

var resourceTypeOptions = []string{"document", "quiz", "quiz_template", "video"}

type addResourceTemplateData struct {
	ChapterID   string
	TopicID     string
	TypeOptions []string
}

type editResourceTemplateData struct {
	Resource    *models.Resource
	TypeOptions []string
}

type ResourcesHandler struct {
	service *services.Service[models.Resource]
}

func NewResourcesHandler(service *services.Service[models.Resource]) *ResourcesHandler {
	return &ResourcesHandler{
		service: service,
	}
}

func (h *ResourcesHandler) OpenAddResource(responseWriter http.ResponseWriter, request *http.Request) {
	chapterID := request.URL.Query().Get("chapterId")
	topicID := request.URL.Query().Get("topicId")
	data := addResourceTemplateData{
		ChapterID:   chapterID,
		TopicID:     topicID,
		TypeOptions: resourceTypeOptions,
	}
	views.ExecuteTemplate(addResourceTemplate, responseWriter, data, nil)
}

func (h *ResourcesHandler) GetResources(responseWriter http.ResponseWriter, request *http.Request) {
	urlVals := request.URL.Query()
	curriculumIDStr := urlVals.Get(CurriculumDropdownName)
	gradeIDStr := urlVals.Get(GradeDropdownName)
	chapterIDStr := urlVals.Get("chapter_id")
	topicIDStr := urlVals.Get("topic_id")

	curriculumID, err := utils.StringToIntType[int16](curriculumIDStr)
	if err != nil {
		http.Error(responseWriter, "Invalid Curriculum ID", http.StatusBadRequest)
		return
	}
	gradeID, err := utils.StringToIntType[int8](gradeIDStr)
	if err != nil {
		http.Error(responseWriter, "Invalid Grade ID", http.StatusBadRequest)
		return
	}

	isTopicRequest := strings.TrimSpace(topicIDStr) != ""
	var queryParams string
	if isTopicRequest {
		topicID, err := utils.StringToIntType[int16](topicIDStr)
		if err != nil {
			http.Error(responseWriter, "Invalid Topic ID", http.StatusBadRequest)
			return
		}
		queryParams = fmt.Sprintf("?curriculum_id=%d&grade_id=%d&topic_id=%d", curriculumID, gradeID, topicID)
	} else {
		chapterID, err := utils.StringToIntType[int16](chapterIDStr)
		if err != nil {
			http.Error(responseWriter, "Invalid Chapter ID", http.StatusBadRequest)
			return
		}
		queryParams = fmt.Sprintf("?curriculum_id=%d&grade_id=%d&chapter_id=%d", curriculumID, gradeID, chapterID)
	}

	resources, err := h.service.GetList(resourcesCurriculumEndPoint+queryParams, resourcesKey, false, true)
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
		// If the request is for chapter resources, keep only rows that don't belong to a topic.
		if !isTopicRequest && resource.TopicID != 0 {
			continue
		}
		filteredResources = append(filteredResources, resource)
	}

	views.ExecuteTemplate(resourceRowTemplate, responseWriter, &filteredResources, template.FuncMap{
		"getName": getResourceName,
	})
}

func (h *ResourcesHandler) EditResource(responseWriter http.ResponseWriter, request *http.Request) {
	resourceIDStr := request.URL.Query().Get("id")
	resourceID, err := utils.StringToIntType[int32](resourceIDStr)
	if err != nil {
		http.Error(responseWriter, "Invalid Resource ID", http.StatusBadRequest)
		return
	}

	selectedResourcePtr, err := h.service.GetObject(resourceIDStr,
		func(resource *models.Resource) bool {
			return resource.ID == int(resourceID)
		}, resourcesKey, resourcesEndPoint)
	if err != nil {
		http.Error(responseWriter, fmt.Sprintf("Error fetching resource: %v", err), http.StatusInternalServerError)
		return
	}

	data := editResourceTemplateData{
		Resource:    selectedResourcePtr,
		TypeOptions: resourceTypeOptions,
	}
	views.ExecuteTemplate(editResourceTemplate, responseWriter, data, template.FuncMap{
		"getName": getResourceName,
	})
}

func (h *ResourcesHandler) UpdateResource(responseWriter http.ResponseWriter, request *http.Request) {
	resourceIDStr := request.FormValue("id")
	resourceID, err := utils.StringToIntType[int32](resourceIDStr)
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

	_, err = h.service.UpdateObject(resourceIDStr, resourcesEndPoint, resourceMap, resourcesKey,
		func(resource *models.Resource) bool {
			return resource.ID == int(resourceID)
		})
	if err != nil {
		http.Error(responseWriter, fmt.Sprintf("Error updating resource: %v", err), http.StatusInternalServerError)
		return
	}

	views.ExecuteTemplate(updateSuccessTemplate, responseWriter, "Resource", nil)
}

func (h *ResourcesHandler) DeleteResource(responseWriter http.ResponseWriter, request *http.Request) {
	resourceIDStr := request.URL.Query().Get("id")
	resourceID, err := utils.StringToIntType[int32](resourceIDStr)
	if err != nil {
		http.Error(responseWriter, "Invalid Resource ID", http.StatusBadRequest)
		return
	}

	err = h.service.DeleteObject(resourceIDStr,
		func(resource *models.Resource) bool {
			return resource.ID != int(resourceID)
		}, resourcesKey, resourcesEndPoint)

	// If http error is thrown from here then target row won't be removed by htmx code
	if err != nil {
		http.Error(responseWriter, err.Error(), http.StatusInternalServerError)
	}
}

func (h *ResourcesHandler) AddResource(responseWriter http.ResponseWriter, request *http.Request) {
	resourceCode := request.FormValue("code")
	resourceName := request.FormValue("name")
	resourceType := request.FormValue("type")
	resourceSubtype := request.FormValue("subtype")
	srcLink := request.FormValue("src_link")

	chapterIDStr := request.FormValue("chapter_id")
	chapterID, err := utils.StringToIntType[int16](chapterIDStr)
	if err != nil {
		http.Error(responseWriter, "Invalid Chapter ID", http.StatusBadRequest)
		return
	}

	topicIDStr := request.FormValue("topic_id")
	var topicID int16
	if strings.TrimSpace(topicIDStr) != "" {
		topicID, err = utils.StringToIntType[int16](topicIDStr)
		if err != nil {
			http.Error(responseWriter, "Invalid Topic ID", http.StatusBadRequest)
			return
		}
	}

	curriculumIDStr := request.FormValue(CurriculumDropdownName)
	curriculumID, err := utils.StringToIntType[int16](curriculumIDStr)
	if err != nil {
		http.Error(responseWriter, "Invalid Curriculum ID", http.StatusBadRequest)
		return
	}

	gradeIDStr := request.FormValue(GradeDropdownName)
	gradeID, err := utils.StringToIntType[int8](gradeIDStr)
	if err != nil {
		http.Error(responseWriter, "Invalid Grade ID", http.StatusBadRequest)
		return
	}

	newResourcePtr := models.NewResource(resourceCode, resourceName, resourceType, resourceSubtype, srcLink, chapterID, curriculumID, gradeID)
	if topicID != 0 {
		newResourcePtr.TopicID = topicID
	}
	newResourcePtr, err = h.service.AddObject(newResourcePtr, resourcesKey, resourcesEndPoint)
	if err != nil {
		http.Error(responseWriter, fmt.Sprintf("Error adding resource: %v", err), http.StatusInternalServerError)
		return
	}

	resourcePtrs := []*models.Resource{newResourcePtr}
	views.ExecuteTemplate(resourceRowTemplate, responseWriter, resourcePtrs, template.FuncMap{
		"getName": getResourceName,
	})
}

func (h *ResourcesHandler) LoadMoveResources(responseWriter http.ResponseWriter, request *http.Request) {
	resourceIDs := request.FormValue("resource_ids")
	if strings.TrimSpace(resourceIDs) == "" {
		http.Error(responseWriter, "Missing resource IDs", http.StatusBadRequest)
		return
	}
	views.ExecuteTemplate(moveResourcesTemplate, responseWriter, resourceIDs, nil)
}

func (h *ResourcesHandler) MoveResource(responseWriter http.ResponseWriter, request *http.Request) {
	err := request.ParseForm()
	if err != nil {
		http.Error(responseWriter, "Invalid form", http.StatusBadRequest)
		return
	}

	curriculumID, gradeID, subjectID := getCurriculumGradeSubjectIDs(request.Form)
	if curriculumID == 0 || gradeID == 0 || subjectID == 0 {
		http.Error(responseWriter, "Invalid curriculum, grade or subject ID", http.StatusBadRequest)
		return
	}

	chapterIDStr := request.Form.Get("chapter-dropdown")
	chapterID, err := utils.StringToIntType[int16](chapterIDStr)
	if err != nil {
		http.Error(responseWriter, fmt.Sprintf("Invalid Chapter ID: %v", err), http.StatusBadRequest)
		return
	}

	topicIDStr := strings.TrimSpace(request.Form.Get("topic_id"))
	var topicIDPtr *int16
	if topicIDStr != "" {
		topicID, parseErr := utils.StringToIntType[int16](topicIDStr)
		if parseErr != nil {
			http.Error(responseWriter, fmt.Sprintf("Invalid Topic ID: %v", parseErr), http.StatusBadRequest)
			return
		}
		topicIDPtr = &topicID
	}

	resourceIDsStr := strings.TrimSpace(request.Form.Get("resource_ids"))
	if resourceIDsStr == "" {
		http.Error(responseWriter, "Missing resource IDs", http.StatusBadRequest)
		return
	}
	resourceIDs := utils.StringSliceToIntSlice(strings.Split(resourceIDsStr, ","))

	requestBody := dto.MoveResourcesRequest{
		ResourceIDs: resourceIDs,
		CurriculumGrades: []models.CurriculumGrade{
			{
				CurriculumID: curriculumID,
				GradeID:      gradeID,
			},
		},
		SubjectID: subjectID,
		ChapterID: chapterID,
		TopicID:   topicIDPtr,
		LangCode:  "en",
	}

	var result any
	err = h.service.Post(moveResourceEndPoint, requestBody, &result)
	if err != nil {
		log.Println("move resource error:", err)
		http.Error(responseWriter, "Failed to move resource", http.StatusInternalServerError)
		return
	}
}

func getResourceName(r models.Resource, lang string) string {
	return r.GetNameByLang(lang)
}
