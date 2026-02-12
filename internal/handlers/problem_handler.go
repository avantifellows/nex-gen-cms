package handlers

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"text/template"

	"github.com/avantifellows/nex-gen-cms/internal/constants"
	"github.com/avantifellows/nex-gen-cms/internal/dto"
	"github.com/avantifellows/nex-gen-cms/internal/handlers/handlerutils"
	"github.com/avantifellows/nex-gen-cms/internal/models"
	"github.com/avantifellows/nex-gen-cms/internal/services"
	"github.com/avantifellows/nex-gen-cms/internal/views"
	"github.com/avantifellows/nex-gen-cms/utils"
)

const problemsKey = "problems"

const problemsEndPoint = "problems"
const problemEndPoint = "resource/problem/%d/en/%s"

const problemTemplate = "problem.html"
const srcProblemRowTemplate = "src_problem_row.html"
const problemsTemplate = "problems.html"
const topicProblemRowTemplate = "topic_problem_row.html"
const addProblemTemplate = "add_problem.html"
const problemTypeOptionsTemplate = "problem_type_options.html"
const addConceptModalTemplate = "add_concept_modal.html"
const editorTemplate = "editor.html"
const inputTagsTemplate = "input_tags.html"

type ProblemsHandler struct {
	problemsService *services.Service[models.Problem]
	skillsService   *services.Service[models.Skill]
	subjectsService *services.Service[models.Subject]
	topicsService   *services.Service[models.Topic]
	tagsService     *services.Service[models.Tag]
}

func NewProblemsHandler(problemsService *services.Service[models.Problem],
	skillsService *services.Service[models.Skill], subjectsService *services.Service[models.Subject],
	topicsService *services.Service[models.Topic], tagsService *services.Service[models.Tag]) *ProblemsHandler {
	return &ProblemsHandler{problemsService: problemsService, skillsService: skillsService,
		subjectsService: subjectsService, topicsService: topicsService, tagsService: tagsService}
}

func (h *ProblemsHandler) GetProblem(responseWriter http.ResponseWriter, request *http.Request) {
	selectedProblemPtr, code, err := h.getProblem(request.URL.Query())
	if err != nil {
		http.Error(responseWriter, err.Error(), code)
		return
	}

	data := dto.HomeData{
		ProblemPtr: selectedProblemPtr,
	}

	views.ExecuteTemplates(responseWriter, data, template.FuncMap{
		"add":         utils.Add,
		"stringToInt": utils.StringToInt,
		"seq":         utils.Seq,
		"getName":     getConceptName,
	}, baseTemplate, problemTemplate)
}

func (h *ProblemsHandler) getProblem(urlValues url.Values) (*models.Problem, int, error) {
	problemIdStr := urlValues.Get("id")
	problemId := utils.StringToInt(problemIdStr)
	endPointWithId := fmt.Sprintf(problemEndPoint, problemId, urlValues.Get(QUERY_PARAM_CURRICULUM_ID))

	// In problemEndPoint problem id is already included in path segment, hence passing blank as first argument
	selectedProblemPtr, err := h.problemsService.GetObject("",
		func(problem *models.Problem) bool {
			return problem.ID == problemId
		}, problemsKey, endPointWithId)
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("error fetching problem: %v", err)
	}

	skills, err := h.skillsService.GetList(skillsEndPoint, skillsKey, false, false)
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("error fetching skills: %v", err)
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
		return selectedProblemPtr, http.StatusOK, nil
	}
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

	// true is passed for onlyRemote, so that new tags inserted via create problem api can also be fetched
	tags, err := h.tagsService.GetList(tagsEndPoint, tagsKey, false, true)
	if err != nil {
		http.Error(responseWriter, fmt.Sprintf("Error fetching tags: %v", err), http.StatusInternalServerError)
		return
	}
	// Create a map to quickly lookup tag names by their ID
	tagsMap := make(map[int]string)
	// Fill the map with the string name of each tag
	for _, tagPtr := range *tags {
		tagsMap[tagPtr.ID] = tagPtr.Name
	}

	// set subject & tag names on each problem
	for _, problemPtr := range *problems {
		problemPtr.Subject = *subjectPtr

		// Loop through tag ids and add corresponding tag names
		for _, tagId := range problemPtr.TagIDs {
			problemPtr.TagNames = append(problemPtr.TagNames, tagsMap[tagId])
		}
	}

	levels := urlValues["level"]
	ptype := urlValues.Get("ptype-dropdown")
	selectedIds := urlValues.Get("selected-ids")
	filterProblems(problems, levels, ptype, selectedIds)

	var tmpl string
	if urlValues.Has("ptype-dropdown") {
		// for add/edit test screen
		tmpl = srcProblemRowTemplate
	} else {
		// for topic screen's Problems tab
		tmpl = topicProblemRowTemplate
	}
	views.ExecuteTemplate(tmpl, responseWriter, problems, nil)
}

func filterProblems(problems *[]*models.Problem, levels []string, ptype string, selectedIdsRaw string) {
	// Build map of already selected problem ids. map is used instead of slice for better performance
	selectedIds := map[int]bool{}
	for _, id := range strings.Split(selectedIdsRaw, ",") {
		selectedIds[utils.StringToInt(id)] = true
	}

	// Build a map of allowed difficulty levels for fast lookup
	allowedLevels := map[string]bool{}
	for _, lvl := range levels {
		if lvl != "" { // skip the empty value (All)
			allowedLevels[lvl] = true
		}
	}

	// If no specific levels selected → treat as ALL selected
	allLevelsAllowed := len(allowedLevels) == 0

	ps := *problems
	n := 0
	for _, p := range ps {
		if p.StatusID == constants.StatusArchived {
			continue
		}

		// difficulty check
		if !allLevelsAllowed && !allowedLevels[p.DifficultyLevel] {
			continue
		}

		// problem type check
		// "" means All is selected in dropdown
		if ptype != "" && p.Subtype != ptype {
			continue
		}

		// skip already selected ones
		if selectedIds[p.ID] {
			continue
		}

		ps[n] = p
		n++
	}

	*problems = ps[:n]
}

func (h *ProblemsHandler) LoadProblems(responseWriter http.ResponseWriter, request *http.Request) {
	topicIdStr := request.URL.Query().Get(QUERY_PARAM_TOPIC_ID)
	views.ExecuteTemplate(problemsTemplate, responseWriter, topicIdStr, nil)
}

func (h *ProblemsHandler) AddProblem(responseWriter http.ResponseWriter, request *http.Request) {
	topicIdStr := request.URL.Query().Get(QUERY_PARAM_TOPIC_ID)
	selectedTopicPtr, code, err := handlerutils.GetTopicById(topicIdStr, h.topicsService)
	if err != nil {
		http.Error(responseWriter, err.Error(), code)
		return
	}

	data := dto.HomeData{
		TopicPtr: selectedTopicPtr,
	}
	views.ExecuteTemplates(responseWriter, data, template.FuncMap{
		"joinInt16":      utils.JoinInt16,
		"add":            utils.Add,
		"stringToInt":    utils.StringToInt,
		"toJson":         utils.ToJson,
		"getConceptName": getConceptName,
	}, baseTemplate, addProblemTemplate, problemTypeOptionsTemplate,
		editorTemplate, inputTagsTemplate)
}

func (h *ProblemsHandler) AddConceptModal(responseWriter http.ResponseWriter, request *http.Request) {
	views.ExecuteTemplates(responseWriter, nil, nil, addConceptModalTemplate, curriculumGradeSelectsTemplate)
}

func (h *ProblemsHandler) CreateProblem(responseWriter http.ResponseWriter, request *http.Request) {
	reqBodyBytes, err := io.ReadAll(request.Body)
	if err != nil {
		http.Error(responseWriter, "Invalid input", http.StatusBadRequest)
		return
	}

	_, err = h.problemsService.AddObject(reqBodyBytes, problemsKey, resourcesEndPoint)
	if err != nil {
		http.Error(responseWriter, fmt.Sprintf("Error adding problem: %v", err), http.StatusInternalServerError)
		return
	}
}

func (h *ProblemsHandler) EditProblem(responseWriter http.ResponseWriter, request *http.Request) {
	selectedProblemPtr, code, err := h.getProblem(request.URL.Query())
	if err != nil {
		http.Error(responseWriter, err.Error(), code)
		return
	}

	data := dto.HomeData{
		ProblemPtr: selectedProblemPtr,
	}

	views.ExecuteTemplates(responseWriter, data, template.FuncMap{
		"joinInt16":      utils.JoinInt16,
		"add":            utils.Add,
		"stringToInt":    utils.StringToInt,
		"toJson":         utils.ToJson,
		"getConceptName": getConceptName,
	}, baseTemplate, addProblemTemplate, problemTypeOptionsTemplate, editorTemplate, inputTagsTemplate)
}

func (h *ProblemsHandler) UpdateProblem(responseWriter http.ResponseWriter, request *http.Request) {
	reqBodyBytes, err := io.ReadAll(request.Body)
	if err != nil {
		http.Error(responseWriter, "Invalid input", http.StatusBadRequest)
		return
	}

	problemIdStr := request.URL.Query().Get("id")
	problemId := utils.StringToInt(problemIdStr)

	_, err = h.problemsService.UpdateObject(problemIdStr, resourcesEndPoint, reqBodyBytes, problemsKey,
		func(problem *models.Problem) bool {
			return (*problem).ID == problemId
		})
	if err != nil {
		http.Error(responseWriter, fmt.Sprintf("Error updating problem: %v", err), http.StatusInternalServerError)
		return
	}
}

func (h *ProblemsHandler) ArchiveProblem(responseWriter http.ResponseWriter, request *http.Request) {
	problemIdStr := request.URL.Query().Get("id")
	problemId := utils.StringToInt(problemIdStr)
	body := map[string]any{
		"cms_status_id": constants.StatusArchived,
		"lang_code":     "en",
	}

	err := h.problemsService.ArchiveObject(problemIdStr, resourcesEndPoint, body, problemsKey,
		func(problem *models.Problem) bool {
			return problem.ID != problemId
		})
	if err != nil {
		http.Error(responseWriter, fmt.Sprintf("Error archiving problem: %v", err), http.StatusInternalServerError)
		return
	}
}
