package handlers

import (
	"net/http"
	"text/template"

	"github.com/avantifellows/nex-gen-cms/internal/dto"
	"github.com/avantifellows/nex-gen-cms/internal/models"
	local_repo "github.com/avantifellows/nex-gen-cms/internal/repositories/local"
	"github.com/avantifellows/nex-gen-cms/internal/services"
	"github.com/avantifellows/nex-gen-cms/utils"
)

const problemsKey = "problems"

const problemsEndPoint = "/problem"

const problemTemplate = "problem.html"

type ProblemsHandler struct {
	service *services.Service[models.Problem]
}

func NewProblemsHandler(service *services.Service[models.Problem]) *ProblemsHandler {
	return &ProblemsHandler{service: service}
}

func (h *ProblemsHandler) GetProblem(responseWriter http.ResponseWriter, request *http.Request) {
	urlValues := request.URL.Query()
	problemIdStr := urlValues.Get("id")
	problemId := utils.StringToInt(problemIdStr)

	selectedProblemPtr, err := h.service.GetObject(problemIdStr,
		func(problem *models.Problem) bool {
			return problem.ID == problemId
		}, problemsKey, problemsEndPoint)
	if err != nil {
		http.Error(responseWriter, err.Error(), http.StatusInternalServerError)
	}

	data := dto.HomeData{
		ProblemPtr: selectedProblemPtr,
	}

	local_repo.ExecuteTemplates(baseTemplate, problemTemplate, responseWriter, data, template.FuncMap{
		"add": utils.Add,
	})
}
