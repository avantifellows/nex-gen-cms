package handlers

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"slices"
	"strings"

	"github.com/avantifellows/nex-gen-cms/internal/constants"
	"github.com/avantifellows/nex-gen-cms/internal/dto"
	"github.com/avantifellows/nex-gen-cms/internal/handlers/handlerutils"
	"github.com/avantifellows/nex-gen-cms/internal/models"
	local_repo "github.com/avantifellows/nex-gen-cms/internal/repositories/local"
	"github.com/avantifellows/nex-gen-cms/internal/services"
	"github.com/avantifellows/nex-gen-cms/utils"
)

const TESTTYPE_DROPDOWN_NAME = "testtype-dropdown"

const testsTemplate = "tests.html"
const testRowTemplate = "test_row.html"
const testTemplate = "test.html"
const problemRowTemplate = "problem_row.html"
const addTestTemplate = "add_test.html"
const testTypeOptionsTemplate = "test_type_options.html"
const addTestDestProblemRowWithoutHeadersTemplate = "dest_problem_row_without_headers.html"
const addTestDestProblemRowWithSubtypeTemplate = "dest_problem_row_with_subtype.html"
const addTestDestProblemRowWithHeadersTemplate = "dest_problem_row_with_headers.html"
const addTestDestProblemRowTemplate = "dest_problem_row.html"
const addTestDestSubtypeRowTemplate = "dest_subtype_row.html"
const addTestDestSubjectRowTemplate = "dest_subject_row.html"
const chipBoxCellTemplate = "chip_box_cells.html"

const resourcesEndPoint = "/resource"
const resourcesCurriculumEndPoint = "/resources/curriculum"
const testProblemsEndPoint = "/resource/test/%d/problems?lang_code=en&" + QUERY_PARAM_CURRICULUM_ID + "=%s"

const testsKey = "tests"

type TestsHandler struct {
	testsService    *services.Service[models.Test]
	subjectsService *services.Service[models.Subject]
	problemsService *services.Service[models.Problem]
}

func NewTestsHandler(testsService *services.Service[models.Test], subjectsService *services.Service[models.Subject],
	problemsService *services.Service[models.Problem]) *TestsHandler {
	return &TestsHandler{
		testsService:    testsService,
		subjectsService: subjectsService,
		problemsService: problemsService,
	}
}

func (h *TestsHandler) LoadTests(responseWriter http.ResponseWriter, request *http.Request) {
	local_repo.ExecuteTemplates(responseWriter, nil, template.FuncMap{
		"slice": utils.Slice,
		"add":   utils.Add,
	}, baseTemplate, testsTemplate, testTypeOptionsTemplate)
}

func (h *TestsHandler) GetTests(responseWriter http.ResponseWriter, request *http.Request) {
	urlValues := request.URL.Query()
	curriculumId, gradeId, _ := getCurriculumGradeSubjectIds(urlValues)
	if curriculumId == 0 || gradeId == 0 {
		return
	}
	testtype := urlValues.Get(TESTTYPE_DROPDOWN_NAME)
	sortColumn := urlValues.Get("sortColumn")
	sortOrder := urlValues.Get("sortOrder")

	queryParams := fmt.Sprintf("?"+QUERY_PARAM_CURRICULUM_ID+"=%d&grade_id=%d&type=test&subtype=%s", curriculumId, gradeId, testtype)
	tests, err := h.testsService.GetList(resourcesCurriculumEndPoint+queryParams, testsKey, false, true)

	if err != nil {
		http.Error(responseWriter, fmt.Sprintf("Error fetching tests: %v", err), http.StatusInternalServerError)
		return
	}

	// set curriculum & grade id on each test
	for _, test := range *tests {
		test.CurriculumID = curriculumId
		test.GradeID = gradeId
	}

	sortTests(*tests, sortColumn, sortOrder)
	local_repo.ExecuteTemplate(testRowTemplate, responseWriter, tests, nil)
}

func sortTests(testPtrs []*models.Test, sortColumn string, sortOrder string) {
	slices.SortStableFunc(testPtrs, func(t1, t2 *models.Test) int {
		var sortResult int
		switch sortColumn {
		case "1":
			c1Suffix := utils.ExtractNumericSuffix(t1.Code)
			c2Suffix := utils.ExtractNumericSuffix(t2.Code)
			// if numeric suffix found for both tests then perform their integer comparison
			if c1Suffix > 0 && c2Suffix > 0 {
				sortResult = c1Suffix - c2Suffix
			} else {
				// perform string comparison of codes, because numeric suffixes could not be found
				sortResult = strings.Compare(t1.Code, t2.Code)
			}
		case "2":
			sortResult = strings.Compare(t1.Name[0].Resource, t2.Name[0].Resource)
		case "3":
			sortResult = int(t1.ProblemCount() - t2.ProblemCount())
		case "4":
			sortResult = int(t1.TypeParams.Marks - t2.TypeParams.Marks)
		case "5":
			sortResult = int(utils.StringToInt(t1.TypeParams.Duration) - utils.StringToInt(t2.TypeParams.Duration))
		default:
			sortResult = 0
		}

		if constants.SortOrder(sortOrder) == constants.SortOrderDesc {
			sortResult = -sortResult
		}
		return sortResult
	})
}

func (h *TestsHandler) GetTest(responseWriter http.ResponseWriter, request *http.Request) {
	selectedTestPtr, code, err := h.getTest(responseWriter, request)
	if err != nil {
		http.Error(responseWriter, err.Error(), code)
		return
	}

	data := dto.HomeData{
		CurriculumID: selectedTestPtr.CurriculumID,
		GradeID:      selectedTestPtr.GradeID,
		TestPtr:      selectedTestPtr,
	}

	local_repo.ExecuteTemplates(responseWriter, data, nil, baseTemplate, testTemplate)
}

func (h *TestsHandler) getTest(responseWriter http.ResponseWriter, request *http.Request) (*models.Test, int, error) {
	urlVals := request.URL.Query()
	testIdStr := urlVals.Get("id")
	testId := utils.StringToInt(testIdStr)

	selectedTestPtr, err := h.testsService.GetObject(testIdStr,
		func(test *models.Test) bool {
			return (*test).ID == testId
		}, testsKey, resourcesEndPoint)
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("error fetching test: %v", err)
	}

	curriculumId, err := utils.StringToIntType[int16](urlVals.Get(QUERY_PARAM_CURRICULUM_ID))
	if err != nil {
		return nil, http.StatusBadRequest, fmt.Errorf("invalid Curriculum ID: %v", err)
	}

	selectedTestPtr.CurriculumID = curriculumId

	// Fill subject names in test
	h.fillSubjectNames(responseWriter, selectedTestPtr)

	return selectedTestPtr, http.StatusOK, nil
}

func (h *TestsHandler) fillSubjectNames(responseWriter http.ResponseWriter, testPtr *models.Test) {
	subjectPtrs, err := h.subjectsService.GetList(subjectsEndPoint, subjectsKey, false, false)
	if err != nil {
		http.Error(responseWriter, fmt.Sprintf("Error fetching subjects: %v", err), http.StatusInternalServerError)
	} else {
		// Create a map to quickly lookup subject names by their ID
		subjectIdToNameMap := make(map[int8]string)

		// fill the map with the address of each subject
		for _, subjectPtr := range *subjectPtrs {
			subjectIdToNameMap[subjectPtr.ID] = subjectPtr.GetNameByLang("en")
		}

		// loop through subjects of test and update subject name
		for i, testSubject := range testPtr.TypeParams.Subjects {
			/**
			updating name directly on testSubject will change it in copy of actual subjects instead of
			original subjects under testPtr, hence we are assigning it to testPtr.TypeParams.Subjects[i].Name
			*/
			testPtr.TypeParams.Subjects[i].Name = subjectIdToNameMap[testSubject.SubjectID]
		}
	}
}

func (h *TestsHandler) GetTestProblems(responseWriter http.ResponseWriter, request *http.Request) {
	problems := h.getTestProblems(responseWriter, request)
	if problems == nil {
		return
	}

	// Passing custom function add to use in template for serial number by adding 1 to index
	local_repo.ExecuteTemplate(problemRowTemplate, responseWriter, problems, template.FuncMap{
		"add": utils.Add,
	})
}

func (h *TestsHandler) getTestProblems(responseWriter http.ResponseWriter, request *http.Request) *[]*models.Problem {
	urlVals := request.URL.Query()
	testIdStr := urlVals.Get("id")
	testId := utils.StringToInt(testIdStr)

	endPointWithId := fmt.Sprintf(testProblemsEndPoint, testId, urlVals.Get(QUERY_PARAM_CURRICULUM_ID))
	problems, err := h.problemsService.GetList(endPointWithId, problemsKey, false, true)

	if err != nil {
		http.Error(responseWriter, fmt.Sprintf("Error fetching problems: %v", err), http.StatusInternalServerError)
	}
	return problems
}

func (h *TestsHandler) AddTest(responseWriter http.ResponseWriter, request *http.Request) {
	local_repo.ExecuteTemplates(responseWriter, nil, template.FuncMap{
		"split":             strings.Split,
		"slice":             utils.Slice,
		"seq":               utils.Seq,
		"getName":           getTestName,
		"add":               utils.Add,
		"joinInt16":         utils.JoinInt16,
		"dict":              utils.Dict,
		"getDisplaySubtype": utils.DisplaySubtype,
	}, baseTemplate, addTestTemplate, testTypeOptionsTemplate, addTestDestSubjectRowTemplate,
		addTestDestSubtypeRowTemplate, addTestDestProblemRowTemplate, chipBoxCellTemplate)
}

func (h *TestsHandler) AddQuestionToTest(responseWriter http.ResponseWriter, request *http.Request) {
	problemIdStr := request.FormValue("id")
	problemId := utils.StringToInt(problemIdStr)

	endPointWithId := fmt.Sprintf(problemEndPoint, problemId, request.FormValue("curriculum-id"))

	// In problemEndPoint problem id is already included in path segment, hence passing blank as first argument
	problemPtr, err := h.problemsService.GetObject("",
		func(problem *models.Problem) bool {
			return problem.ID == problemId
		}, problemsKey, endPointWithId)
	if err != nil {
		http.Error(responseWriter, err.Error(), http.StatusInternalServerError)
	}

	subjectPtr, statusCode, err := handlerutils.FetchSelectedSubject(request.FormValue("subject-id"),
		h.subjectsService, subjectsKey, subjectsEndPoint)
	if err != nil {
		http.Error(responseWriter, err.Error(), statusCode)
		return
	}
	/**
	 * Following properties might not be coming with problem response as these are not in resource &
	 * problem_lang tables
	 */
	problemPtr.Subject = *subjectPtr
	problemPtr.DifficultyLevel = request.FormValue("difficulty")

	insertAfterId := request.FormValue("insert-after-id")
	subjectExists := request.FormValue("subject-exists") == "true"
	subtypeExists := request.FormValue("subtype-exists") == "true"

	var filename string
	var data any

	switch {
	case !subjectExists && !subtypeExists:
		// Need subject + subtype header
		filename = addTestDestProblemRowWithHeadersTemplate
		data = problemPtr

	case subjectExists && !subtypeExists:
		// Only subtype header needed
		filename = addTestDestProblemRowWithSubtypeTemplate
		data = map[string]any{
			"Problem":       problemPtr,
			"InsertAfterId": insertAfterId,
		}

	case subtypeExists:
		// Just problem row
		filename = addTestDestProblemRowWithoutHeadersTemplate
		data = map[string]any{
			"Problem":       problemPtr,
			"InsertAfterId": insertAfterId,
		}
	}

	local_repo.ExecuteTemplates(responseWriter, data, template.FuncMap{
		"getName":           getSubjectName,
		"joinInt16":         utils.JoinInt16,
		"dict":              utils.Dict,
		"getDisplaySubtype": utils.DisplaySubtype,
	}, filename, addTestDestSubjectRowTemplate, addTestDestSubtypeRowTemplate, addTestDestProblemRowTemplate, chipBoxCellTemplate)
}

func (h *TestsHandler) CreateTest(responseWriter http.ResponseWriter, request *http.Request) {
	// Declare a variable to hold the parsed JSON
	var testData map[string]interface{}

	// Decode the JSON body into the testData map
	err := json.NewDecoder(request.Body).Decode(&testData)
	if err != nil {
		http.Error(responseWriter, "Error parsing JSON", http.StatusBadRequest)
		return
	}

	// Print the parsed JSON
	fmt.Println("Received test data:", testData)
}

func (h *TestsHandler) EditTest(responseWriter http.ResponseWriter, request *http.Request) {
	selectedTestPtr, code, err := h.getTest(responseWriter, request)
	if err != nil {
		http.Error(responseWriter, err.Error(), code)
		return
	}

	problems := h.getTestProblems(responseWriter, request)
	if problems == nil {
		return
	}
	problemsMap := make(map[int]*models.Problem)
	for _, p := range *problems {
		problemsMap[p.ID] = p
	}

	data := dto.HomeData{
		TestPtr:  selectedTestPtr,
		Problems: problemsMap,
	}

	local_repo.ExecuteTemplates(responseWriter, data, template.FuncMap{
		"split":             strings.Split,
		"slice":             utils.Slice,
		"seq":               utils.Seq,
		"getName":           getTestName,
		"add":               utils.Add,
		"joinInt16":         utils.JoinInt16,
		"dict":              utils.Dict,
		"getDisplaySubtype": utils.DisplaySubtype,
	}, baseTemplate, addTestTemplate, testTypeOptionsTemplate, addTestDestSubjectRowTemplate,
		addTestDestSubtypeRowTemplate, addTestDestProblemRowTemplate, chipBoxCellTemplate)
}

func getTestName(t models.Test, lang string) string {
	return t.GetNameByLang(lang)
}
