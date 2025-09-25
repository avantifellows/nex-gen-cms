package handlers

import (
	"net/http"

	"github.com/avantifellows/nex-gen-cms/internal/views"
)

const modulesTemplate = "modules.html"

type ModulesHandler struct {
	// service *services.Service[models.Test]
}

func NewModulesHandler( /*service *services.Service[models.Test]*/ ) *ModulesHandler {
	return &ModulesHandler{
		// service: service,
	}
}

func (h *ModulesHandler) LoadModules(responseWriter http.ResponseWriter, request *http.Request) {
	views.ExecuteTemplates(responseWriter, nil, nil, baseTemplate, modulesTemplate)
}
