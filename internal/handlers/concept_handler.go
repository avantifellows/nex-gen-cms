package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"text/template"

	"github.com/avantifellows/nex-gen-cms/internal/models"
	"github.com/avantifellows/nex-gen-cms/internal/services"
	"github.com/avantifellows/nex-gen-cms/internal/views"
	"github.com/avantifellows/nex-gen-cms/utils"
)

const conceptsEndPoint = "concept"

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

	excludeStr := urlVals.Get("exclude")
	var filtered *[]*models.Concept

	if excludeStr == "" {
		// no exclusion, just reuse
		filtered = concepts
	} else {
		excludeIds := make(map[int32]struct{})
		for _, idStr := range strings.Split(excludeStr, ",") {
			if idStr == "" {
				continue
			}
			if id, err := utils.StringToIntType[int32](idStr); err == nil {
				excludeIds[id] = struct{}{}
			}
		}
		tmp := make([]*models.Concept, 0, len(*concepts))
		for _, c := range *concepts {
			if _, found := excludeIds[c.ID]; !found {
				tmp = append(tmp, c)
			}
		}
		filtered = &tmp
	}

	// send only data if mode is data
	if urlVals.Get("mode") == "data" {
		responseWriter.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(responseWriter).Encode(filtered); err != nil {
			http.Error(responseWriter, fmt.Sprintf("Error encoding concepts: %v", err), http.StatusInternalServerError)
		}
	} else {
		views.ExecuteTemplate(conceptRowTemplate, responseWriter, filtered, template.FuncMap{
			"getName": getConceptName,
		})
	}
}

func getConceptName(c models.Concept, lang string) string {
	return c.GetNameByLang(lang)
}
