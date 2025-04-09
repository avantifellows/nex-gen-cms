package handlers

import (
	"fmt"
	"html/template"
	htmlTpl "html/template"
	"log"
	"net/http"
	textTpl "text/template"

	"github.com/avantifellows/nex-gen-cms/internal/dto"
	"github.com/avantifellows/nex-gen-cms/internal/models"
	local_repo "github.com/avantifellows/nex-gen-cms/internal/repositories/local"
	"github.com/avantifellows/nex-gen-cms/internal/services"
	"github.com/avantifellows/nex-gen-cms/utils"
)

const problemsKey = "problems"
const skillsKey = "skills"

const problemsEndPoint = "/problem"
const skillsEndPoint = "/skill"

const problemTemplate = "problem.html"
const srcProblemRowTemplate = "src_problem_row.html"

type ProblemsHandler struct {
	problemsService *services.Service[models.Problem]
	skillsService   *services.Service[models.Skill]
}

func NewProblemsHandler(problemsService *services.Service[models.Problem],
	skillsService *services.Service[models.Skill]) *ProblemsHandler {
	return &ProblemsHandler{problemsService: problemsService, skillsService: skillsService}
}

func (h *ProblemsHandler) GetProblem(responseWriter http.ResponseWriter, request *http.Request) {
	urlValues := request.URL.Query()
	problemIdStr := urlValues.Get("id")
	problemId := utils.StringToInt(problemIdStr)

	selectedProblemPtr, err := h.problemsService.GetObject(problemIdStr,
		func(problem *models.Problem) bool {
			return problem.ID == problemId
		}, problemsKey, problemsEndPoint)
	if err != nil {
		http.Error(responseWriter, err.Error(), http.StatusInternalServerError)
	}

	skills, err := h.skillsService.GetList(skillsEndPoint, skillsKey, false, false)
	if err != nil {
		http.Error(responseWriter, fmt.Sprintf("Error fetching skills: %v", err), http.StatusInternalServerError)
	} else {
		// Create a map to quickly lookup skills by their ID
		skillPtrsMap := make(map[int16]*models.Skill)

		// Fill the map with the address of each skill
		for _, skillPtr := range *skills {
			skillPtrsMap[skillPtr.ID] = skillPtr
		}

		// Loop through skill ids and add corresponding skills
		for _, skillId := range selectedProblemPtr.SkillIDs {
			selectedProblemPtr.Skills = append(selectedProblemPtr.Skills, *skillPtrsMap[skillId])
		}
	}

	data := dto.HomeData{
		ProblemPtr: selectedProblemPtr,
	}

	local_repo.ExecuteTemplates(responseWriter, data, textTpl.FuncMap{
		"add":         utils.Add,
		"stringToInt": utils.StringToInt,
	}, baseTemplate, problemTemplate)
}

func (h *ProblemsHandler) GetTopicProblems(responseWriter http.ResponseWriter, request *http.Request) {
	urlValues := request.URL.Query()
	topicIdStr := urlValues.Get("topic-dropdown")
	topicId, err := utils.StringToIntType[int16](topicIdStr)
	log.Println("topic id = ", topicId)
	if err != nil {
		// http.Error(responseWriter, "Invalid Topic ID", http.StatusBadRequest)
		return
	}

	var problems = []models.Problem{
		{
			Code: "P3156",
			MetaData: models.ProbMetaData{
				Question: htmlTpl.HTML("If R is the radius of the Earth..."),
			},
		},
		{
			Code: "P3195",
			MetaData: models.ProbMetaData{
				Question: template.HTML("The acceleration due to gravity..."),
			},
		},
		{
			Code: "P3201",
			MetaData: models.ProbMetaData{
				Question: template.HTML("Suppose the Earth suddenly shrinks..."),
			},
		},
	}
	local_repo.ExecuteTemplate(srcProblemRowTemplate, responseWriter, problems, nil)
}
