package handlers

import (
	"fmt"
	"net/http"
	"slices"
	"strings"

	"github.com/avantifellows/nex-gen-cms/internal/constants"
	"github.com/avantifellows/nex-gen-cms/internal/models"
	local_repo "github.com/avantifellows/nex-gen-cms/internal/repositories/local"
	"github.com/avantifellows/nex-gen-cms/internal/services"
	"github.com/avantifellows/nex-gen-cms/utils"
)

const TESTTYPE_DROPDOWN_NAME = "testtype-dropdown"

const testRowTemplate = "test_row.html"

const resourcesCurriculumEndPoint = "/resources/curriculum"

const testsKey = "tests"

type TestsHandler struct {
	service *services.Service[models.Test]
}

func NewTestsHandler(service *services.Service[models.Test]) *TestsHandler {
	return &TestsHandler{
		service: service,
	}
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
	tests, err := h.service.GetList(resourcesCurriculumEndPoint+queryParams, testsKey, false, true)

	if err != nil {
		http.Error(responseWriter, fmt.Sprintf("Error fetching tests: %v", err), http.StatusInternalServerError)
		return
	}

	sortTests(*tests, sortColumn, sortOrder)
	local_repo.ExecuteTemplate(testRowTemplate, responseWriter, tests)
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
