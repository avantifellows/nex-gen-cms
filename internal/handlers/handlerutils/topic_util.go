package handlerutils

import (
	"github.com/avantifellows/nex-gen-cms/internal/models"
	"github.com/avantifellows/nex-gen-cms/internal/services"
	"github.com/avantifellows/nex-gen-cms/utils"
)

const TopicsEndPoint = "topic"
const TopicsKey = "topics"

func GetTopicByID(topicIDStr string, topicsService *services.Service[models.Topic]) (*models.Topic, int, error) {
	return GetEntityByID(
		topicIDStr,
		topicsService,
		TopicsKey,
		TopicsEndPoint,
		utils.StringToIntType[int16],
		func(t *models.Topic, id int16) bool { return t.ID == id },
		"Topic",
	)
}
