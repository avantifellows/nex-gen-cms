package handlers

import (
	"fmt"
	"net/http"
	"text/template"

	"github.com/avantifellows/nex-gen-cms/internal/models"
	local_repo "github.com/avantifellows/nex-gen-cms/internal/repositories/local"
	"github.com/avantifellows/nex-gen-cms/internal/services"
	"github.com/avantifellows/nex-gen-cms/utils"
)

const conceptsEndPoint = "/concept"

const conceptsKey = "concepts"

const conceptRowTemplate = "concept_row.html"

type ConceptsHandler struct {
	service *services.Service[models.Concept]
}

func NewConceptsHandler(service *services.Service[models.Concept]) *ConceptsHandler {
	return &ConceptsHandler{
		service: service,
	}
}

func (h *ConceptsHandler) GetConcepts(responseWriter http.ResponseWriter, request *http.Request) {
	queryParams := ""

	urlVals := request.URL.Query()
	if urlVals.Has(QUERY_PARAM_TOPIC_ID) {
		topicIdStr := urlVals.Get(QUERY_PARAM_TOPIC_ID)
		topicId, err := utils.StringToIntType[int16](topicIdStr)
		if err != nil {
			http.Error(responseWriter, fmt.Sprintf("Invalid Topic ID: %v", err), http.StatusBadRequest)
			return
		}
		queryParams = fmt.Sprintf("?"+QUERY_PARAM_TOPIC_ID+"=%d", topicId)
	}
	concepts, err := h.service.GetList(conceptsEndPoint+queryParams, conceptsKey, false, true)
	if err != nil {
		http.Error(responseWriter, fmt.Sprintf("Error fetching concepts: %v", err), http.StatusInternalServerError)
		return
	}
	local_repo.ExecuteTemplate(conceptRowTemplate, responseWriter, concepts, template.FuncMap{
		"getName": getConceptName,
	})
}

func getConceptName(c models.Concept, lang string) string {
	return c.GetNameByLang(lang)
}
