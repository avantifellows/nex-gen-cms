package handlers

import (
	"github.com/avantifellows/nex-gen-cms/internal/models"
	"github.com/avantifellows/nex-gen-cms/internal/services"
)

const getTopicsEndPoint = "/topic"
const topicsKey = "topics"

type TopicsHandler struct {
	service *services.Service[models.Topic]
}

func NewTopicsHandler(service *services.Service[models.Topic]) *TopicsHandler {
	return &TopicsHandler{
		service: service,
	}
}
