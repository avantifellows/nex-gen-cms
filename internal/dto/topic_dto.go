package dto

import "github.com/avantifellows/nex-gen-cms/internal/models"

type TopicData struct {
	HomeData
	TopicPtr *models.Topic
}
