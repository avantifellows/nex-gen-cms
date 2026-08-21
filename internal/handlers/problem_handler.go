package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
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
const problemEndPoint = "resource/problem/%d"
const searchProblemsEndPoint = "problems/search"
const testsContainingProblemsEndPoint = "resources/tests-containing-problems"
const batchProblemsEndPoint = "resources/problems/batch"
const similarSearchEndPoint = "problems/similar-search"

const problemsTemplate = "problems.html"
const problemTemplate = "problem.html"
const srcProblemRowParentTemplate = "src_problem_row_parent.html"
const srcProblemRowTemplate = "src_problem_row.html"
const topicProblemsTemplate = "topic_problems.html"
const topicProblemRowTemplate = "topic_problem_row.html"
const searchProblemRowTemplate = "search_problem_row.html"
const addProblemTemplate = "add_problem.html"
const problemTypeOptionsTemplate = "problem_type_options.html"
const addConceptModalTemplate = "add_concept_modal.html"
const editorTemplate = "editor.html"
const problemAnswerNumericalTemplate = "problem_answer_numerical.html"
const inputTagsTemplate = "input_tags.html"
const problemTestAssociationTemplate = "problem_test_association_modal.html"
const moveProblemsTemplate = "move_problems_modal.html"
const copyProblemModalTemplate = "copy_problem_modal.html"
const duplicateProblemsModalTemplate = "duplicate_problems_modal.html"

// confirmDuplicatesParam, when set to "true" on a save request, skips the similarity check and
// saves directly — used when the editor has already seen the duplicate-warning modal and chose
// to save anyway.
const confirmDuplicatesParam = "confirmDuplicates"

type ProblemsHandler struct {
	problemsService *services.Service[models.Problem]
	skillsService   *services.Service[models.Skill]
	subjectsService *services.Service[models.Subject]
	topicsService   *services.Service[models.Topic]
	chaptersService *services.Service[models.Chapter]
	tagsService     *services.Service[models.Tag]
}

func NewProblemsHandler(problemsService *services.Service[models.Problem],
	skillsService *services.Service[models.Skill], subjectsService *services.Service[models.Subject],
	topicsService *services.Service[models.Topic], chaptersService *services.Service[models.Chapter],
	tagsService *services.Service[models.Tag]) *ProblemsHandler {
	return &ProblemsHandler{problemsService: problemsService, skillsService: skillsService,
		subjectsService: subjectsService, topicsService: topicsService, chaptersService: chaptersService,
		tagsService: tagsService}
}

func (h *ProblemsHandler) GetProblem(responseWriter http.ResponseWriter, request *http.Request) {
	selectedProblemPtr, code, err := h.getProblem(request.URL.Query())
	if err != nil {
		http.Error(responseWriter, err.Error(), code)
		return
	}

	topicIDStr := strconv.Itoa(int(selectedProblemPtr.TopicID))
	selectedTopicPtr, _, _ := handlerutils.GetTopicByID(topicIDStr, h.topicsService)

	var selectedChapterPtr *models.Chapter
	if selectedTopicPtr != nil {
		chapterIDStr := strconv.Itoa(int(selectedTopicPtr.ChapterID))
		selectedChapterPtr, _, _ = handlerutils.GetChapterByID(chapterIDStr, h.chaptersService)
	}

	data := dto.ProblemData{
		HomeData: dto.HomeData{
			CurriculumID: selectedProblemPtr.CurriculumID,
			GradeID:      selectedProblemPtr.GradeID,
			SubjectID:    selectedProblemPtr.SubjectID,
		},
		ProblemPtr: selectedProblemPtr,
		TopicPtr:   selectedTopicPtr,
		ChapterPtr: selectedChapterPtr,
	}

	views.ExecuteTemplates(responseWriter, data, template.FuncMap{
		"add":         utils.Add,
		"stringToInt": utils.StringToInt,
		"seq":         utils.Seq,
		"getName":     getConceptName,
		"langName":    utils.LangName,
	}, baseTemplate, problemTemplate)
}

func (h *ProblemsHandler) getProblem(urlValues url.Values) (*models.Problem, int, error) {
	problemIdStr := urlValues.Get("id")
	problemId := utils.StringToInt(problemIdStr)
	endPointWithID := fmt.Sprintf(problemEndPoint, problemId)

	// In problemEndPoint problem id is already included in path segment, hence passing blank as first argument
	selectedProblemPtr, err := h.problemsService.GetObject("",
		func(problem *models.Problem) bool {
			return problem.ID == problemId
		}, problemsKey, endPointWithID)
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("error fetching problem: %v", err)
	}

	skills, err := h.skillsService.GetList(skillsEndPoint, skillsKey, false, false)
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("error fetching skills: %v", err)
	}

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

	if err = h.enrichProblemTagNames(selectedProblemPtr); err != nil {
		return nil, http.StatusInternalServerError, err
	}
	return selectedProblemPtr, http.StatusOK, nil
}

func (h *ProblemsHandler) enrichProblemTagNames(problem *models.Problem) error {
	if len(problem.TagIDs) == 0 || len(problem.TagNames) > 0 {
		return nil
	}

	tagsMap, err := h.getTagsMap()
	if err != nil {
		return err
	}

	for _, tagId := range problem.TagIDs {
		problem.TagNames = append(problem.TagNames, tagsMap[tagId])
	}
	return nil
}

func (h *ProblemsHandler) GetTopicProblems(responseWriter http.ResponseWriter, request *http.Request) {
	const includeParagraphSiblingsParam = "include_paragraph_siblings"

	urlValues := request.URL.Query()
	topicIdStr := urlValues.Get("topic-dropdown")
	topicId, err := utils.StringToIntType[int16](topicIdStr)
	if err != nil {
		http.Error(responseWriter, "Invalid Topic ID", http.StatusBadRequest)
		return
	}

	queryParams := fmt.Sprintf("?topic_id=%d", topicId)
	if urlValues.Has(includeParagraphSiblingsParam) {
		queryParams += "&" + includeParagraphSiblingsParam + "=" + urlValues.Get(includeParagraphSiblingsParam)
	}
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

	tagsMap, err := h.getTagsMap()
	if err != nil {
		http.Error(responseWriter, err.Error(), http.StatusInternalServerError)
		return
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

	if urlValues.Has("ptype-dropdown") {
		// for add/edit test screen
		views.ExecuteTemplates(responseWriter, problems, nil, srcProblemRowParentTemplate, srcProblemRowTemplate)

	} else {
		// for topic screen's Problems tab
		views.ExecuteTemplate(topicProblemRowTemplate, responseWriter, problems, nil)
	}
}

func (h *ProblemsHandler) getTagsMap() (map[int]string, error) {
	// true is passed for onlyRemote, so that new tags inserted via create problem api can also be fetched
	tags, err := h.tagsService.GetList(tagsEndPoint, tagsKey, false, true)
	if err != nil {
		return nil, fmt.Errorf("error fetching tags: %v", err)
	}

	// Create a map to quickly lookup tag names by their ID
	tagsMap := make(map[int]string)
	// Fill the map with the string name of each tag
	for _, tagPtr := range *tags {
		tagsMap[tagPtr.ID] = tagPtr.Name
	}

	return tagsMap, nil
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

func (h *ProblemsHandler) LoadProblems(responseWriter http.ResponseWriter, _ *http.Request) {
	views.ExecuteTemplates(responseWriter, nil, nil, baseTemplate, problemsTemplate)
}

func (h *ProblemsHandler) LoadTopicProblems(responseWriter http.ResponseWriter, request *http.Request) {
	topicIdStr := request.URL.Query().Get(QUERY_PARAM_TOPIC_ID)
	views.ExecuteTemplate(topicProblemsTemplate, responseWriter, topicIdStr, nil)
}

func (h *ProblemsHandler) AddProblem(responseWriter http.ResponseWriter, request *http.Request) {
	topicIDStr := request.URL.Query().Get(QUERY_PARAM_TOPIC_ID)
	selectedTopicPtr, code, err := handlerutils.GetTopicByID(topicIDStr, h.topicsService)
	if err != nil {
		http.Error(responseWriter, err.Error(), code)
		return
	}

	data := dto.ProblemData{
		TopicPtr: selectedTopicPtr,
	}
	views.ExecuteTemplates(responseWriter, data, template.FuncMap{
		"joinInt16":        utils.JoinInt16,
		"add":              utils.Add,
		"stringToInt":      utils.StringToInt,
		"toJson":           utils.ToJson,
		"getConceptName":   getConceptName,
		"dict":             utils.Dict,
		"emptyStringSlice": utils.EmptyStringSlice,
	}, baseTemplate, addProblemTemplate, problemTypeOptionsTemplate,
		editorTemplate, problemAnswerNumericalTemplate, inputTagsTemplate)
}

func (h *ProblemsHandler) AddConceptModal(responseWriter http.ResponseWriter, _ *http.Request) {
	views.ExecuteTemplates(responseWriter, nil, nil, addConceptModalTemplate, curriculumGradeSelectsTemplate)
}

func (h *ProblemsHandler) CreateProblem(responseWriter http.ResponseWriter, request *http.Request) {
	reqBodyBytes, err := io.ReadAll(request.Body)
	if err != nil {
		http.Error(responseWriter, "Invalid input", http.StatusBadRequest)
		return
	}

	var problem models.Problem
	_ = json.Unmarshal(reqBodyBytes, &problem) // best-effort: an unparsable body just skips the similarity check

	if h.duplicatesFound(responseWriter, request, extractSimilarityLanguages(problem), 0) {
		return
	}

	_, err = h.problemsService.AddObject(reqBodyBytes, problemsKey, resourcesEndPoint)
	if err != nil {
		http.Error(responseWriter, fmt.Sprintf("Error adding problem: %v", err), http.StatusInternalServerError)
		return
	}
}

func (h *ProblemsHandler) CreateProblems(responseWriter http.ResponseWriter, request *http.Request) {
	reqBodyBytes, err := io.ReadAll(request.Body)
	if err != nil {
		http.Error(responseWriter, "Invalid input", http.StatusBadRequest)
		return
	}

	// Endpoint expects a JSON object with "problems" array & "paragraph".
	var batch struct {
		Problems []models.Problem `json:"problems"`
	}
	_ = json.Unmarshal(reqBodyBytes, &batch) // best-effort: an unparsable body just skips the similarity check

	var languages []dto.SimilarSearchLanguage
	for _, problem := range batch.Problems {
		languages = append(languages, extractSimilarityLanguages(problem)...)
	}
	if h.duplicatesFound(responseWriter, request, languages, 0) {
		return
	}

	var result any
	err = h.problemsService.Post(batchProblemsEndPoint, json.RawMessage(reqBodyBytes), &result)
	if err != nil {
		http.Error(responseWriter, fmt.Sprintf("Error adding problems: %v", err), http.StatusInternalServerError)
		return
	}
}

// duplicatesFound runs the similarity check (unless bypassed by confirmDuplicatesParam) and, if
// matches turn up, writes the 409 + confirmation-modal response itself. Returns true when it has
// written a response — the caller should return without saving. Also writes an error response
// and returns true if the check itself fails, so a flaky similarity check never masks the save.
//
// excludeProblemID drops a match against that problem's own id — needed on edit, where the
// problem's own already-saved text otherwise self-matches at ~100%. Pass 0 on create, where
// there's no id yet.
func (h *ProblemsHandler) duplicatesFound(responseWriter http.ResponseWriter, request *http.Request,
	languages []dto.SimilarSearchLanguage, excludeProblemID int) bool {
	if request.URL.Query().Get(confirmDuplicatesParam) == "true" || len(languages) == 0 {
		return false
	}

	var resp dto.SimilarSearchResponse
	if err := h.problemsService.Post(similarSearchEndPoint, dto.SimilarSearchRequest{Languages: languages}, &resp); err != nil {
		http.Error(responseWriter, fmt.Sprintf("Error checking similar problems: %v", err), http.StatusInternalServerError)
		return true
	}

	matches := make([]dto.SimilarProblemMatch, 0, len(resp.Problems))
	for _, match := range resp.Problems {
		if match.ID != excludeProblemID {
			matches = append(matches, match)
		}
	}
	if len(matches) == 0 {
		return false
	}

	renderDuplicateProblemsModal(responseWriter, matches)
	return true
}

// extractSimilarityLanguages returns the {lang_code, text} pairs the similarity check needs from
// a problem's language versions, skipping any with blank question text.
func extractSimilarityLanguages(problem models.Problem) []dto.SimilarSearchLanguage {
	languages := make([]dto.SimilarSearchLanguage, 0, len(problem.LangVersions))
	for _, langVersion := range problem.LangVersions {
		text := string(langVersion.MetaData.Question)
		if text == "" {
			continue
		}
		languages = append(languages, dto.SimilarSearchLanguage{
			LangCode: langVersion.LangCode,
			Text:     text,
		})
	}
	return languages
}

// renderDuplicateProblemsModal writes a 409 so the frontend's fetch handler can tell this apart
// from a successful save, then renders the confirmation modal as the response body.
func renderDuplicateProblemsModal(responseWriter http.ResponseWriter, matches []dto.SimilarProblemMatch) {
	duplicateMatches := make([]dto.DuplicateProblemMatch, len(matches))
	for i, match := range matches {
		duplicateMatches[i] = dto.DuplicateProblemMatch{
			ID:           match.ID,
			Code:         match.Code,
			LangCode:     match.LangCode,
			MatchPercent: int(match.MatchScore*100 + 0.5),
		}
	}

	responseWriter.WriteHeader(http.StatusConflict)
	views.ExecuteTemplate(duplicateProblemsModalTemplate, responseWriter, duplicateMatches, template.FuncMap{
		"langName": utils.LangName,
	})
}

func (h *ProblemsHandler) EditProblem(responseWriter http.ResponseWriter, request *http.Request) {
	selectedProblemPtr, code, err := h.getProblem(request.URL.Query())
	if err != nil {
		http.Error(responseWriter, err.Error(), code)
		return
	}

	data := dto.ProblemData{
		HomeData: dto.HomeData{
			CurriculumID: selectedProblemPtr.CurriculumID,
			GradeID:      selectedProblemPtr.GradeID,
			SubjectID:    selectedProblemPtr.SubjectID,
		},
		ProblemPtr: selectedProblemPtr,
	}

	views.ExecuteTemplates(responseWriter, data, template.FuncMap{
		"joinInt16":        utils.JoinInt16,
		"add":              utils.Add,
		"stringToInt":      utils.StringToInt,
		"toJson":           utils.ToJson,
		"getConceptName":   getConceptName,
		"dict":             utils.Dict,
		"emptyStringSlice": utils.EmptyStringSlice,
	}, baseTemplate, addProblemTemplate, problemTypeOptionsTemplate, editorTemplate,
		problemAnswerNumericalTemplate, inputTagsTemplate)
}

func (h *ProblemsHandler) UpdateProblem(responseWriter http.ResponseWriter, request *http.Request) {
	reqBodyBytes, err := io.ReadAll(request.Body)
	if err != nil {
		http.Error(responseWriter, "Invalid input", http.StatusBadRequest)
		return
	}

	problemIdStr := request.URL.Query().Get("id")
	problemId := utils.StringToInt(problemIdStr)

	var problem models.Problem
	_ = json.Unmarshal(reqBodyBytes, &problem) // best-effort: an unparsable body just skips the similarity check

	if h.duplicatesFound(responseWriter, request, extractSimilarityLanguages(problem), problemId) {
		return
	}

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

func (h *ProblemsHandler) GetSearchProblems(responseWriter http.ResponseWriter, request *http.Request) {
	urlVals := request.URL.Query()
	search := urlVals.Get("problem-search")

	limit := utils.StringToIntOrDefault(urlVals.Get("limit"), 10, 1)  // min = 1
	offset := utils.StringToIntOrDefault(urlVals.Get("offset"), 0, 0) // min = 0
	queryParams := "?search=" + url.QueryEscape(search) + "&limit=" + strconv.Itoa(limit) + "&offset=" + strconv.Itoa(offset)

	subjectId := utils.StringToInt(urlVals.Get("problems-subject-dropdown"))
	if subjectId != 0 {
		queryParams += "&subject_id=" + strconv.Itoa(subjectId)
	}

	problems, err := h.problemsService.GetList(searchProblemsEndPoint+queryParams, "", false, true)
	if err != nil {
		http.Error(responseWriter, fmt.Sprintf("Error fetching problems: %v", err), http.StatusInternalServerError)
		return
	}

	subjects, err := h.subjectsService.GetList(handlerutils.SubjectsEndPoint, handlerutils.SubjectsKey, false, false)
	if err != nil {
		http.Error(responseWriter, "error fetching subjects", http.StatusInternalServerError)
		return
	}

	subjectMap := make(map[int8]*models.Subject)
	for i := range *subjects {
		sub := (*subjects)[i]
		subjectMap[sub.ID] = sub
	}

	for _, sub := range subjectMap {
		if sub.ParentID != 0 {
			if parent, ok := subjectMap[sub.ParentID]; ok {
				sub.ParentName = parent.Name
			}
		}
	}

	tagsMap, err := h.getTagsMap()
	if err != nil {
		http.Error(responseWriter, err.Error(), http.StatusInternalServerError)
		return
	}

	// set subject & tag names on each problem
	for _, problemPtr := range *problems {
		subjectPtr, ok := subjectMap[problemPtr.SubjectID]
		if !ok {
			http.Error(responseWriter, "subject not found", http.StatusInternalServerError)
			return
		}

		problemPtr.Subject = *subjectPtr

		// Loop through tag ids and add corresponding tag names
		for _, tagId := range problemPtr.TagIDs {
			problemPtr.TagNames = append(problemPtr.TagNames, tagsMap[tagId])
		}
	}

	// Decide hasMore BEFORE filtering
	hasMore := len(*problems) >= limit // true if more pages should exist
	if !hasMore {
		responseWriter.Header().Set("hasMore", "false")
	}

	filterProblems(problems, nil, "", "")
	views.ExecuteTemplate(searchProblemRowTemplate, responseWriter, problems, nil)
}

func (h *ProblemsHandler) LoadTestAssociations(responseWriter http.ResponseWriter, request *http.Request) {
	if err := request.ParseForm(); err != nil {
		http.Error(responseWriter, fmt.Sprintf("Error parsing form: %v", err), http.StatusBadRequest)
		return
	}
	problemIDsStr := request.Form["select-problem"]
	problemIDs := utils.StringSliceToIntSlice(problemIDsStr)

	req := dto.TestsContainingProblemsRequest{
		ProblemIDs: problemIDs,
	}
	var resp dto.TestsContainingProblemsResponse

	err := h.problemsService.Post(testsContainingProblemsEndPoint, req, &resp)
	if err != nil {
		http.Error(responseWriter, fmt.Sprintf("Error fetching linked tests: %v", err), http.StatusInternalServerError)
		return
	}
	views.ExecuteTemplate(problemTestAssociationTemplate, responseWriter, resp.ProblemTests, nil)
}

func (h *ProblemsHandler) LoadMoveProblems(responseWriter http.ResponseWriter, request *http.Request) {
	idsStr := request.FormValue("problem_ids")
	views.ExecuteTemplate(moveProblemsTemplate, responseWriter, idsStr, nil)
}

func (h *ProblemsHandler) LoadCopyProblemDialog(responseWriter http.ResponseWriter, request *http.Request) {
	problemIdStr := request.URL.Query().Get("id")
	if problemIdStr == "" {
		http.Error(responseWriter, "Missing problem ID", http.StatusBadRequest)
		return
	}

	views.ExecuteTemplate(copyProblemModalTemplate, responseWriter, problemIdStr, nil)
}

func (h *ProblemsHandler) CopyProblem(responseWriter http.ResponseWriter, request *http.Request) {
	urlValues := request.URL.Query()

	problemQuery := url.Values{}
	problemQuery.Set("id", urlValues.Get("id"))

	selectedProblemPtr, code, err := h.getProblem(problemQuery)
	if err != nil {
		http.Error(responseWriter, err.Error(), code)
		return
	}

	topicIDStr := urlValues.Get(QUERY_PARAM_TOPIC_ID)
	selectedTopicPtr, code, err := handlerutils.GetTopicByID(topicIDStr, h.topicsService)
	if err != nil {
		http.Error(responseWriter, err.Error(), code)
		return
	}

	curriculumId, gradeId, subjectId := getCurriculumGradeSubjectIds(urlValues)
	if curriculumId == 0 || gradeId == 0 || subjectId == 0 {
		http.Error(responseWriter, "Invalid curriculum, grade or subject ID", http.StatusBadRequest)
		return
	}

	copiedProblem := selectedProblemPtr.CopyTo(selectedTopicPtr, curriculumId, gradeId, subjectId)

	data := dto.ProblemData{
		HomeData: dto.HomeData{
			CurriculumID: curriculumId,
			GradeID:      gradeId,
			SubjectID:    subjectId,
		},
		ProblemPtr: &copiedProblem,
		TopicPtr:   selectedTopicPtr,
	}

	views.ExecuteTemplates(responseWriter, data, template.FuncMap{
		"joinInt16":        utils.JoinInt16,
		"add":              utils.Add,
		"stringToInt":      utils.StringToInt,
		"toJson":           utils.ToJson,
		"getConceptName":   getConceptName,
		"dict":             utils.Dict,
		"emptyStringSlice": utils.EmptyStringSlice,
	}, baseTemplate, addProblemTemplate, problemTypeOptionsTemplate, editorTemplate,
		problemAnswerNumericalTemplate, inputTagsTemplate)
}

func (h *ProblemsHandler) MoveProblems(responseWriter http.ResponseWriter, request *http.Request) {
	err := request.ParseForm()
	if err != nil {
		http.Error(responseWriter, "Invalid form", http.StatusBadRequest)
		return
	}

	curriculumId, gradeId, subjectId := getCurriculumGradeSubjectIds(request.Form)
	if curriculumId == 0 || gradeId == 0 || subjectId == 0 {
		http.Error(responseWriter, "Invalid curriculum, grade or subject ID", http.StatusBadRequest)
		return
	}

	chapterIdStr := request.Form.Get("chapter-dropdown")
	chapterId, err := utils.StringToIntType[int16](chapterIdStr)
	if err != nil {
		http.Error(responseWriter, fmt.Sprintf("Invalid Chapter ID: %v", err), http.StatusBadRequest)
		return
	}

	topicIdStr := request.Form.Get("topic_id")
	topicId, err := utils.StringToIntType[int16](topicIdStr)
	if err != nil {
		http.Error(responseWriter, fmt.Sprintf("Invalid Topic ID: %v", err), http.StatusBadRequest)
		return
	}
	topicIdPtr := &topicId

	problemIdsStr := request.Form.Get("problem_ids")
	problemIds := utils.StringSliceToIntSlice(strings.Split(problemIdsStr, ","))

	reqBody := dto.MoveResourcesRequest{
		ResourceIDs: problemIds,
		CurriculumGrades: []models.CurriculumGrade{
			{
				CurriculumID: curriculumId,
				GradeID:      gradeId,
			},
		},
		SubjectID: subjectId,
		ChapterID: chapterId,
		TopicID:   topicIdPtr,
		LangCode:  "en",
	}

	var result any

	err = h.problemsService.Post(moveResourceEndPoint, reqBody, &result)
	if err != nil {
		log.Println("move problems error:", err)
		http.Error(responseWriter, "Failed to move problems", http.StatusInternalServerError)
		return
	}
}
