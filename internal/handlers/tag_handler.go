package handlers

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/avantifellows/nex-gen-cms/internal/models"
	local_repo "github.com/avantifellows/nex-gen-cms/internal/repositories/local"
	"github.com/avantifellows/nex-gen-cms/internal/services"
)

const tagsEndPoint = "tag"

const tagsKey = "tags"

const tagRowTemplate = "tag_row.html"

type TagsHandler struct {
	service *services.Service[models.Tag]
}

func NewTagsHandler(service *services.Service[models.Tag]) *TagsHandler {
	return &TagsHandler{service: service}
}

func (h *TagsHandler) GetTags(responseWriter http.ResponseWriter, request *http.Request) {
	query := strings.ToLower(request.URL.Query().Get("q"))
	selectedTags := request.URL.Query()["selected"] // []string
	// Normalize selected tags for O(1) performance while checking for existence inside this map
	// against O(n) when checked in array using linear search
	selectedTagsMap := make(map[string]bool)
	for _, tag := range selectedTags {
		selectedTagsMap[strings.ToLower(tag)] = true
	}

	tags, err := h.service.GetList(tagsEndPoint, tagsKey, false, false)
	if err != nil {
		http.Error(responseWriter, fmt.Sprintf("Error fetching tags: %v", err), http.StatusInternalServerError)
		return
	}

	// Filter tags
	var filteredTags []*models.Tag
	for _, tag := range *tags {
		tagLower := strings.ToLower(tag.Name)
		if strings.Contains(tagLower, query) && !selectedTagsMap[tagLower] {
			filteredTags = append(filteredTags, tag)
		}
	}

	local_repo.ExecuteTemplate(tagRowTemplate, responseWriter, filteredTags, nil)
}
