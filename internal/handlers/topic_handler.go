package handlers

import (
	"fmt"
	"net/http"
	"slices"
	"strings"
	"text/template"

	"github.com/avantifellows/nex-gen-cms/internal/constants"
	"github.com/avantifellows/nex-gen-cms/internal/dto"
	"github.com/avantifellows/nex-gen-cms/internal/handlers/handlerutils"
	"github.com/avantifellows/nex-gen-cms/internal/models"
	"github.com/avantifellows/nex-gen-cms/internal/services"
	"github.com/avantifellows/nex-gen-cms/internal/views"
	"github.com/avantifellows/nex-gen-cms/utils"
)

const QUERY_PARAM_TOPIC_ID = "topic_id"

const topicsTemplate = "topics.html"
const topicRowTemplate = "topic_row.html"
const addTopicTemplate = "add_topic.html"
const editTopicTemplate = "edit_topic.html"
const topicDropdownTemplate = "topic_dropdown.html"
const topicTemplate = "topic.html"

type TopicsHandler struct {
	service *services.Service[models.Topic]
}

func NewTopicsHandler(service *services.Service[models.Topic]) *TopicsHandler {
	return &TopicsHandler{
		service: service,
	}
}

func (h *TopicsHandler) OpenAddTopic(responseWriter http.ResponseWriter, request *http.Request) {
	chapterId := request.URL.Query().Get("chapterId")
	views.ExecuteTemplate(addTopicTemplate, responseWriter, chapterId, nil)
}

func (h *TopicsHandler) AddTopic(responseWriter http.ResponseWriter, request *http.Request) {
	topicCode := request.FormValue("code")
	topicName := request.FormValue("name")
	chapterIdStr := request.FormValue("chapter_id")
	chapterId, err := utils.StringToIntType[int16](chapterIdStr)
	if err != nil {
		http.Error(responseWriter, "Invalid Chapter ID", http.StatusBadRequest)
		return
	}
	curriculumIdStr := request.FormValue(CURRICULUM_DROPDOWN_NAME)
	curriculumId, err := utils.StringToIntType[int16](curriculumIdStr)
	if err != nil {
		http.Error(responseWriter, "Invalid Curriculum ID", http.StatusBadRequest)
		return
	}
	newTopicPtr := models.NewTopic(topicCode, topicName, chapterId, curriculumId)

	newTopicPtr, err = h.service.AddObject(newTopicPtr, handlerutils.TopicsKey, handlerutils.TopicsEndPoint)
	if err != nil {
		http.Error(responseWriter, fmt.Sprintf("Error adding topic: %v", err), http.StatusInternalServerError)
		return
	}

	topicPtrs := []*models.Topic{newTopicPtr}
	views.ExecuteTemplate(topicRowTemplate, responseWriter, topicPtrs, template.FuncMap{
		"getName": getTopicName,
	})
}

func getTopicName(t models.Topic, lang string) string {
	return t.GetNameByLang(lang)
}

func (h *TopicsHandler) ArchiveTopic(responseWriter http.ResponseWriter, request *http.Request) {
	topicIdStr := request.URL.Query().Get("id")
	topicId, err := utils.StringToIntType[int16](topicIdStr)
	if err != nil {
		http.Error(responseWriter, "Invalid Topic ID", http.StatusBadRequest)
		return
	}

	topicMap := map[string]any{
		"cms_status_id": constants.StatusArchived,
	}

	err = h.service.ArchiveObject(topicIdStr, handlerutils.TopicsEndPoint, topicMap, handlerutils.TopicsKey,
		func(topic *models.Topic) bool {
			return (*topic).ID != topicId
		})

	// If http error is thrown from here then target row won't be removed by htmx code
	if err != nil {
		http.Error(responseWriter, err.Error(), http.StatusInternalServerError)
	}
}

func (h *TopicsHandler) EditTopic(responseWriter http.ResponseWriter, request *http.Request) {
	selectedTopicPtr, code, err := handlerutils.GetTopicById(request.URL.Query().Get("id"), h.service)
	if err != nil {
		http.Error(responseWriter, err.Error(), code)
		return
	}

	views.ExecuteTemplate(editTopicTemplate, responseWriter, selectedTopicPtr, template.FuncMap{
		"getName": getTopicName,
	})
}

func (h *TopicsHandler) UpdateTopic(responseWriter http.ResponseWriter, request *http.Request) {
	topicIdStr := request.FormValue("id")
	topicId, err := utils.StringToIntType[int16](topicIdStr)
	if err != nil {
		http.Error(responseWriter, "Invalid Topic ID", http.StatusBadRequest)
		return
	}

	topicName := request.FormValue("name")
	topicCode := request.FormValue("code")

	dummyTopicPtr := &models.Topic{}
	topicMap := dummyTopicPtr.BuildMap(topicCode, topicName)

	_, err = h.service.UpdateObject(topicIdStr, handlerutils.TopicsEndPoint, topicMap, handlerutils.TopicsKey,
		func(topic *models.Topic) bool {
			return topic.ID == topicId
		})
	if err != nil {
		http.Error(responseWriter, fmt.Sprintf("Error updating topic: %v", err), http.StatusInternalServerError)
	}

	views.ExecuteTemplate(updateSuccessTemplate, responseWriter, "Topic", nil)
}

func sortTopics(topics []*models.Topic, sortColumn string, sortOrder string) {
	slices.SortStableFunc(topics, func(t1, t2 *models.Topic) int {
		var sortResult int
		switch sortColumn {
		case "1":
			t1Suffix := utils.ExtractNumericSuffix(t1.Code)
			t2Suffix := utils.ExtractNumericSuffix(t2.Code)
			// if numeric suffix found for both topics then perform their integer comparison
			if t1Suffix > 0 && t2Suffix > 0 {
				sortResult = t1Suffix - t2Suffix
			} else {
				// perform string comparison of codes, because numeric suffixes could not be found
				sortResult = strings.Compare(t1.Code, t2.Code)
			}
		case "2":
			sortResult = strings.Compare(t1.GetNameByLang("en"), t2.GetNameByLang("en"))
		default:
			sortResult = 0
		}

		if constants.SortOrder(sortOrder) == constants.SortOrderDesc {
			sortResult = -sortResult
		}
		return sortResult
	})
}

func (h *TopicsHandler) GetTopic(responseWriter http.ResponseWriter, request *http.Request) {
	selectedTopicPtr, code, err := handlerutils.GetTopicById(request.URL.Query().Get("id"), h.service)
	if err != nil {
		http.Error(responseWriter, err.Error(), code)
		return
	}

	curriculumId, gradeId, subjectId := getCurriculumGradeSubjectIds(request.URL.Query())
	data := dto.HomeData{
		CurriculumID: curriculumId,
		GradeID:      gradeId,
		SubjectID:    subjectId,
		TopicPtr:     selectedTopicPtr,
	}
	views.ExecuteTemplates(responseWriter, data, template.FuncMap{
		"getName": getTopicName,
	}, baseTemplate, topicTemplate)
}
