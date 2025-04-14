package handlers

import (
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
const addTestDestProblemRowTemplate = "dest_problem_row.html"
const addTestDestProblemRowWithHeadersTemplate = "dest_problem_row_with_headers.html"

const resourcesEndPoint = "/resource"
const resourcesCurriculumEndPoint = "/resources/curriculum"
const testProblemsEndPoint = "/resource/test/%d/problems?lang_code=en"

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
	local_repo.ExecuteTemplates(responseWriter, nil, nil, baseTemplate, testsTemplate, testTypeOptionsTemplate)
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

	queryParams := fmt.Sprintf("?curriculum_id=%d&grade_id=%d&type=test&subtype=%s", curriculumId, gradeId, testtype)
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
	testIdStr := request.URL.Query().Get("id")
	testId := utils.StringToInt(testIdStr)

	selectedTestPtr, err := h.testsService.GetObject(testIdStr,
		func(test *models.Test) bool {
			return (*test).ID == testId
		}, testsKey, resourcesEndPoint)
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("error fetching test: %v", err)
	}

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
	testIdStr := request.URL.Query().Get("test_id")
	testId := utils.StringToInt(testIdStr)

	endPointWithId := fmt.Sprintf(testProblemsEndPoint, testId)
	problems, err := h.problemsService.GetList(endPointWithId, problemsKey, false, true)

	if err != nil {
		http.Error(responseWriter, fmt.Sprintf("Error fetching problems: %v", err), http.StatusInternalServerError)
		return
	}

	// Passing custom function add to use in template for serial number by adding 1 to index
	local_repo.ExecuteTemplate(problemRowTemplate, responseWriter, problems, template.FuncMap{
		"add": utils.Add,
	})
}

func (h *TestsHandler) AddTest(responseWriter http.ResponseWriter, request *http.Request) {
	local_repo.ExecuteTemplates(responseWriter, nil, nil, baseTemplate, addTestTemplate, testTypeOptionsTemplate)
}

func (h *TestsHandler) AddQuestionToTest(responseWriter http.ResponseWriter, request *http.Request) {
	subjectPtr, statusCode, err := handlerutils.FetchSelectedSubject(request.FormValue("subject-id"),
		h.subjectsService, subjectsKey, subjectsEndPoint)
	if err != nil {
		http.Error(responseWriter, err.Error(), statusCode)
		return
	}

	problem := models.Problem{
		ID:   utils.StringToInt(request.FormValue("id")),
		Code: request.FormValue("code"),
		MetaData: models.ProbMetaData{
			Question: template.HTML(request.FormValue("question")),
		},
		Subject: *subjectPtr,
	}
	insertAfterId := request.FormValue("insert-after-id")
	var filename string
	if insertAfterId == "" {
		filename = addTestDestProblemRowWithHeadersTemplate
	} else {
		filename = addTestDestProblemRowTemplate
	}

	data := map[string]interface{}{
		"Problem":       problem,
		"InsertAfterId": insertAfterId,
	}
	responseWriter.Header().Set("Content-Type", "text/html")

	local_repo.ExecuteTemplate(filename, responseWriter, data, template.FuncMap{
		"getName": getSubjectName,
	})
}
