package handlers

import (
	"fmt"
	"net/http"
	"slices"
	"strings"

	"github.com/avantifellows/nex-gen-cms/internal/constants"
	"github.com/avantifellows/nex-gen-cms/internal/models"
	local_repo "github.com/avantifellows/nex-gen-cms/internal/repositories/local"
	"github.com/avantifellows/nex-gen-cms/internal/services"
	"github.com/avantifellows/nex-gen-cms/utils"
)

const topicsEndPoint = "/topic"
const topicsKey = "topics"

const topicsTemplate = "topics.html"
const topicRowTemplate = "topic_row.html"
const addTopicTemplate = "add_topic.html"
const editTopicTemplate = "edit_topic.html"

type TopicsHandler struct {
	service *services.Service[models.Topic]
}

func NewTopicsHandler(service *services.Service[models.Topic]) *TopicsHandler {
	return &TopicsHandler{
		service: service,
	}
}

func (h *TopicsHandler) OpenAddTopic(w http.ResponseWriter, r *http.Request) {
	chapterId := r.URL.Query().Get("chapterId")
	local_repo.ExecuteTemplate(addTopicTemplate, w, chapterId)
}

func (h *TopicsHandler) AddTopic(w http.ResponseWriter, r *http.Request) {
	topicCode := r.FormValue("code")
	topicName := r.FormValue("name")
	chapterIdStr := r.FormValue("chapter_id")
	chapterId, err := utils.StringToIntType[int16](chapterIdStr)
	if err != nil {
		http.Error(w, "Invalid Chapter ID", http.StatusBadRequest)
		return
	}
	curriculumIdStr := r.FormValue(CURRICULUM_DROPDOWN_NAME)
	curriculumId, err := utils.StringToIntType[int16](curriculumIdStr)
	if err != nil {
		http.Error(w, "Invalid Curriculum ID", http.StatusBadRequest)
		return
	}
	newTopicPtr := models.NewTopic(topicCode, topicName, chapterId, curriculumId)

	newTopicPtr, err = h.service.AddObject(newTopicPtr, topicsKey, topicsEndPoint)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error adding topic: %v", err), http.StatusInternalServerError)
		return
	}

	topicPtrs := []*models.Topic{newTopicPtr}
	local_repo.ExecuteTemplate(topicRowTemplate, w, topicPtrs)
}

func (h *TopicsHandler) DeleteTopic(w http.ResponseWriter, r *http.Request) {
	topicIdStr := r.URL.Query().Get("id")
	topicId, err := utils.StringToIntType[int16](topicIdStr)
	if err != nil {
		http.Error(w, "Invalid Topic ID", http.StatusBadRequest)
		return
	}
	err = h.service.DeleteObject(topicIdStr, func(t *models.Topic) bool {
		return t.ID != topicId
	}, topicsKey, topicsEndPoint)

	// If http error is thrown from here then target row won't be removed by htmx code
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (h *TopicsHandler) EditTopic(w http.ResponseWriter, r *http.Request) {
	selectedTopicPtr, code, err := h.getTopic(r)
	if err != nil {
		http.Error(w, err.Error(), code)
		return
	}

	local_repo.ExecuteTemplate(editTopicTemplate, w, selectedTopicPtr)
}

func (h *TopicsHandler) getTopic(r *http.Request) (*models.Topic, int, error) {
	topicIdStr := r.URL.Query().Get("id")
	topicId, err := utils.StringToIntType[int16](topicIdStr)
	if err != nil {
		return nil, http.StatusBadRequest, fmt.Errorf("invalid Topic ID: %w", err)
	}

	selectedTopicPtr, err := h.service.GetObject(topicIdStr,
		func(topic *models.Topic) bool {
			return (*topic).ID == topicId
		}, topicsKey, topicsEndPoint)
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("error fetching topic: %v", err)
	}

	return selectedTopicPtr, http.StatusOK, nil
}

func (h *TopicsHandler) UpdateTopic(w http.ResponseWriter, r *http.Request) {
	topicIdStr := r.FormValue("id")
	topicId, err := utils.StringToIntType[int16](topicIdStr)
	if err != nil {
		http.Error(w, "Invalid Topic ID", http.StatusBadRequest)
		return
	}

	topicName := r.FormValue("name")
	topicCode := r.FormValue("code")

	dummyTopicPtr := &models.Topic{}
	topicMap := dummyTopicPtr.BuildMap(topicCode, topicName)

	_, err = h.service.UpdateObject(topicIdStr, topicsEndPoint, topicMap, topicsKey,
		func(topic *models.Topic) bool {
			return topic.ID == topicId
		})
	if err != nil {
		http.Error(w, fmt.Sprintf("Error updating topic: %v", err), http.StatusInternalServerError)
	}

	local_repo.ExecuteTemplate(updateSuccessTemplate, w, "Topic")
}

func sortTopics(topics []models.Topic) {
	slices.SortStableFunc(topics, func(t1, t2 models.Topic) int {
		var sortResult int
		switch topicSortState.Column {
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
			sortResult = strings.Compare(t1.Name, t2.Name)
		default:
			sortResult = 0
		}

		if topicSortState.Order == constants.SortOrderDesc {
			sortResult = -sortResult
		}
		return sortResult
	})
}
