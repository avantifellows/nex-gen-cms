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

func GetTopicByID(topicIDStr string, topicsService *services.Service[models.Topic]) (*models.Topic, int, error) {
	topicID, err := utils.StringToIntType[int16](topicIDStr)
	if err != nil {
		return nil, http.StatusBadRequest, fmt.Errorf("invalid Topic ID: %w", err)
	}

	selectedTopicPtr, err := topicsService.GetObject(topicIDStr,
		func(topic *models.Topic) bool {
			return (*topic).ID == topicID
		}, TopicsKey, TopicsEndPoint)
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("error fetching topic: %v", err)
	}

	return selectedTopicPtr, http.StatusOK, nil
}
