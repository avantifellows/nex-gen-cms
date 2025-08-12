package handlerutils

import (
	"fmt"
	"net/http"

	"github.com/avantifellows/nex-gen-cms/internal/models"
	"github.com/avantifellows/nex-gen-cms/internal/services"
	"github.com/avantifellows/nex-gen-cms/utils"
)

const TopicsEndPoint = "topic"
const TopicsKey = "topics"

func GetTopicById(topicIdStr string, topicsService *services.Service[models.Topic]) (*models.Topic, int, error) {
	topicId, err := utils.StringToIntType[int16](topicIdStr)
	if err != nil {
		return nil, http.StatusBadRequest, fmt.Errorf("invalid Topic ID: %w", err)
	}

	selectedTopicPtr, err := topicsService.GetObject(topicIdStr,
		func(topic *models.Topic) bool {
			return (*topic).ID == topicId
		}, TopicsKey, TopicsEndPoint)
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("error fetching topic: %v", err)
	}

	return selectedTopicPtr, http.StatusOK, nil
}
