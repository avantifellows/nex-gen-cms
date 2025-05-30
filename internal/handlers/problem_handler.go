package handlers

import (
	"fmt"
	"net/http"
	"strings"
	"text/template"

	"github.com/avantifellows/nex-gen-cms/internal/dto"
	"github.com/avantifellows/nex-gen-cms/internal/handlers/handlerutils"
	"github.com/avantifellows/nex-gen-cms/internal/models"
	local_repo "github.com/avantifellows/nex-gen-cms/internal/repositories/local"
	"github.com/avantifellows/nex-gen-cms/internal/services"
	"github.com/avantifellows/nex-gen-cms/utils"
)

const problemsKey = "problems"
const skillsKey = "skills"

const problemsEndPoint = "/problems"
const problemEndPoint = "/resource/problem/%d/en/%s"
const skillsEndPoint = "/skill"

const problemTemplate = "problem.html"
const srcProblemRowTemplate = "src_problem_row.html"
const problemsTemplate = "problems.html"
const topicProblemRowTemplate = "topic_problem_row.html"
const addProblemTemplate = "add_problem.html"

type ProblemsHandler struct {
	problemsService *services.Service[models.Problem]
	skillsService   *services.Service[models.Skill]
	subjectsService *services.Service[models.Subject]
}

func NewProblemsHandler(problemsService *services.Service[models.Problem],
	skillsService *services.Service[models.Skill], subjectsService *services.Service[models.Subject]) *ProblemsHandler {
	return &ProblemsHandler{problemsService: problemsService, skillsService: skillsService, subjectsService: subjectsService}
}

func (h *ProblemsHandler) GetProblem(responseWriter http.ResponseWriter, request *http.Request) {
	urlValues := request.URL.Query()
	problemIdStr := urlValues.Get("id")
	problemId := utils.StringToInt(problemIdStr)
	endPointWithId := fmt.Sprintf(problemEndPoint, problemId, urlValues.Get(QUERY_PARAM_CURRICULUM_ID))

	// In problemEndPoint problem id is already included in path segment, hence passing blank as first argument
	selectedProblemPtr, err := h.problemsService.GetObject("",
		func(problem *models.Problem) bool {
			return problem.ID == problemId
		}, problemsKey, endPointWithId)
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

	local_repo.ExecuteTemplates(responseWriter, data, template.FuncMap{
		"add":         utils.Add,
		"stringToInt": utils.StringToInt,
		"seq":         utils.Seq,
	}, baseTemplate, problemTemplate)
}

func (h *ProblemsHandler) GetTopicProblems(responseWriter http.ResponseWriter, request *http.Request) {
	urlValues := request.URL.Query()
	topicIdStr := urlValues.Get("topic-dropdown")
	topicId, err := utils.StringToIntType[int16](topicIdStr)
	if err != nil {
		http.Error(responseWriter, "Invalid Topic ID", http.StatusBadRequest)
		return
	}

	queryParams := fmt.Sprintf("?"+QUERY_PARAM_CURRICULUM_ID+"=%s&topic_id=%d&lang_code=en", urlValues.Get(CURRICULUM_DROPDOWN_NAME), topicId)
	problems, err := h.problemsService.GetList(problemsEndPoint+queryParams, problemsKey, false, true)
	if err != nil {
		http.Error(responseWriter, fmt.Sprintf("Error fetching problems: %v", err), http.StatusInternalServerError)
		return
	}

	subjectPtr, statusCode, err := handlerutils.FetchSelectedSubject(urlValues.Get(SUBJECT_DROPDOWN_NAME),
		h.subjectsService)
	if err != nil {
		http.Error(responseWriter, err.Error(), statusCode)
		return
	}

	// set subject on each problem
	for _, problem := range *problems {
		problem.Subject = *subjectPtr
	}

	filterProblems(problems, urlValues.Get("level-dropdown"), urlValues.Get("ptype-dropdown"), urlValues.Get("selected-ids"))

	var tmpl string
	if urlValues.Has("level-dropdown") {
		// for add/edit test screen
		tmpl = srcProblemRowTemplate
	} else {
		// for topic screen's Problems tab
		tmpl = topicProblemRowTemplate
	}
	local_repo.ExecuteTemplate(tmpl, responseWriter, problems, nil)
}

func filterProblems(problems *[]*models.Problem, difficulty string, ptype string, selectedIdsRaw string) {
	// Build map of already selected problem ids. map is used instead of slice for better performance
	selectedIds := map[int]bool{}
	for _, id := range strings.Split(selectedIdsRaw, ",") {
		selectedIds[utils.StringToInt(id)] = true
	}

	ps := *problems
	n := 0
	for _, p := range ps {
		// "" means All is selected in dropdown
		if (difficulty == "" || p.DifficultyLevel == difficulty) && (ptype == "" || p.Subtype == ptype) && !selectedIds[p.ID] {
			ps[n] = p
			n++
		}
	}
	*problems = ps[:n]
}

func (h *ProblemsHandler) LoadProblems(responseWriter http.ResponseWriter, request *http.Request) {
	topicIdStr := request.URL.Query().Get("topic_id")
	local_repo.ExecuteTemplate(problemsTemplate, responseWriter, topicIdStr, nil)
}

func (h *ProblemsHandler) AddProblem(responseWriter http.ResponseWriter, request *http.Request) {
	local_repo.ExecuteTemplates(responseWriter, nil, nil, baseTemplate, addProblemTemplate)
}
