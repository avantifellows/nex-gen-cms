package handlers

import (
	"net/http"

	local_repo "github.com/avantifellows/nex-gen-cms/internal/repositories/local"
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
	local_repo.ExecuteTemplates(baseTemplate, modulesTemplate, responseWriter, nil)
}
