package handlers

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"text/template"

	"github.com/avantifellows/nex-gen-cms/internal/constants"
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

var resourceTypeOptions = []string{"class", "content", "document", "quiz", "video"}
var resourceSubtypeOptions = []string{"Module", "Previous Year Questions", "Assessment", "Video Lectures"}

type addResourceTemplateData struct {
	ChapterID      string
	TopicID        string
	TypeOptions    []string
	SubtypeOptions []string
}

type editResourceTemplateData struct {
	Resource       *models.Resource
	TypeOptions    []string
	SubtypeOptions []string
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
	chapterId := request.URL.Query().Get("chapterId")
	topicId := request.URL.Query().Get("topicId")
	data := addResourceTemplateData{
		ChapterID:      chapterId,
		TopicID:        topicId,
		TypeOptions:    resourceTypeOptions,
		SubtypeOptions: resourceSubtypeOptions,
	}
	views.ExecuteTemplate(addResourceTemplate, responseWriter, data, nil)
}

func (h *ResourcesHandler) GetResources(responseWriter http.ResponseWriter, request *http.Request) {
	urlVals := request.URL.Query()
	curriculumIdStr := urlVals.Get(CURRICULUM_DROPDOWN_NAME)
	gradeIdStr := urlVals.Get(GRADE_DROPDOWN_NAME)
	chapterIdStr := urlVals.Get("chapter_id")
	topicIdStr := urlVals.Get("topic_id")

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

	isTopicRequest := strings.TrimSpace(topicIdStr) != ""
	var queryParams string
	if isTopicRequest {
		topicId, err := utils.StringToIntType[int16](topicIdStr)
		if err != nil {
			http.Error(responseWriter, "Invalid Topic ID", http.StatusBadRequest)
			return
		}
		queryParams = fmt.Sprintf("?curriculum_id=%d&grade_id=%d&topic_id=%d", curriculumId, gradeId, topicId)
	} else {
		chapterId, err := utils.StringToIntType[int16](chapterIdStr)
		if err != nil {
			http.Error(responseWriter, "Invalid Chapter ID", http.StatusBadRequest)
			return
		}
		queryParams = fmt.Sprintf("?curriculum_id=%d&grade_id=%d&chapter_id=%d", curriculumId, gradeId, chapterId)
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
		if resource.StatusID == constants.StatusArchived || resourceType == "problem" || resourceType == "test" {
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
	resourceIdStr := request.URL.Query().Get("id")
	resourceId, err := utils.StringToIntType[int32](resourceIdStr)
	if err != nil {
		http.Error(responseWriter, "Invalid Resource ID", http.StatusBadRequest)
		return
	}

	selectedResourcePtr, err := h.service.GetObject(resourceIdStr,
		func(resource *models.Resource) bool {
			return resource.ID == int(resourceId)
		}, resourcesKey, resourcesEndPoint)
	if err != nil {
		http.Error(responseWriter, fmt.Sprintf("Error fetching resource: %v", err), http.StatusInternalServerError)
		return
	}

	data := editResourceTemplateData{
		Resource:       selectedResourcePtr,
		TypeOptions:    resourceTypeOptions,
		SubtypeOptions: resourceSubtypeOptions,
	}
	views.ExecuteTemplate(editResourceTemplate, responseWriter, data, template.FuncMap{
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

	_, err = h.service.UpdateObject(resourceIdStr, resourcesEndPoint, resourceMap, resourcesKey,
		func(resource *models.Resource) bool {
			return resource.ID == int(resourceId)
		})
	if err != nil {
		http.Error(responseWriter, fmt.Sprintf("Error updating resource: %v", err), http.StatusInternalServerError)
		return
	}

	views.ExecuteTemplate(updateSuccessTemplate, responseWriter, "Resource", nil)
}

func (h *ResourcesHandler) ArchiveResource(responseWriter http.ResponseWriter, request *http.Request) {
	resourceIdStr := request.URL.Query().Get("id")
	resourceId, err := utils.StringToIntType[int32](resourceIdStr)
	if err != nil {
		http.Error(responseWriter, "Invalid Resource ID", http.StatusBadRequest)
		return
	}

	resourceMap := map[string]any{
		"cms_status_id": constants.StatusArchived,
	}

	err = h.service.ArchiveObject(resourceIdStr, resourcesEndPoint, resourceMap, resourcesKey,
		func(resource *models.Resource) bool {
			return resource.ID != int(resourceId)
		})

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

	chapterIdStr := request.FormValue("chapter_id")
	chapterId, err := utils.StringToIntType[int16](chapterIdStr)
	if err != nil {
		http.Error(responseWriter, "Invalid Chapter ID", http.StatusBadRequest)
		return
	}

	topicIdStr := request.FormValue("topic_id")
	var topicId int16
	if strings.TrimSpace(topicIdStr) != "" {
		topicId, err = utils.StringToIntType[int16](topicIdStr)
		if err != nil {
			http.Error(responseWriter, "Invalid Topic ID", http.StatusBadRequest)
			return
		}
	}

	curriculumIdStr := request.FormValue(CURRICULUM_DROPDOWN_NAME)
	curriculumId, err := utils.StringToIntType[int16](curriculumIdStr)
	if err != nil {
		http.Error(responseWriter, "Invalid Curriculum ID", http.StatusBadRequest)
		return
	}

	gradeIdStr := request.FormValue(GRADE_DROPDOWN_NAME)
	gradeId, err := utils.StringToIntType[int8](gradeIdStr)
	if err != nil {
		http.Error(responseWriter, "Invalid Grade ID", http.StatusBadRequest)
		return
	}

	newResourcePtr := models.NewResource(resourceCode, resourceName, resourceType, resourceSubtype, srcLink, chapterId, curriculumId, gradeId)
	if topicId != 0 {
		newResourcePtr.TopicID = topicId
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

	curriculumID, gradeID, subjectID := getCurriculumGradeSubjectIds(request.Form)
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
	var topicID int16
	if topicIDStr != "" {
		topicID, err = utils.StringToIntType[int16](topicIDStr)
		if err != nil {
			http.Error(responseWriter, fmt.Sprintf("Invalid Topic ID: %v", err), http.StatusBadRequest)
			return
		}
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
		TopicID:   topicID,
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
