package handlers

import (
	"fmt"
	"net/http"

	"github.com/avantifellows/nex-gen-cms/internal/constants"
	"github.com/avantifellows/nex-gen-cms/internal/dto"
	"github.com/avantifellows/nex-gen-cms/internal/models"
	local_repo "github.com/avantifellows/nex-gen-cms/internal/repositories/local"
	"github.com/avantifellows/nex-gen-cms/internal/services"
)

const TESTTYPE_DROPDOWN_NAME = "testtype-dropdown"

const testsTemplate = "tests.html"
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

var testSortState = dto.SortState{
	Column: "0",
	Order:  constants.SortOrderAsc,
}

func (h *TestsHandler) LoadTests(responseWriter http.ResponseWriter, request *http.Request) {
	updateSortState(request, &testSortState)
	local_repo.ExecuteTemplate(testsTemplate, responseWriter, testSortState)
}

func (h *TestsHandler) GetTests(responseWriter http.ResponseWriter, request *http.Request) {
	urlValues := request.URL.Query()
	curriculumId, gradeId, _ := getCurriculumGradeSubjectIds(urlValues)
	if curriculumId == 0 || gradeId == 0 {
		return
	}
	testSubtype := urlValues.Get(TESTTYPE_DROPDOWN_NAME)
	fmt.Println("curr = ", curriculumId, gradeId, testSubtype)

	queryParams := fmt.Sprintf("?curriculum_id=%d&grade_id=%d&subtype=%s", curriculumId, gradeId, testSubtype)
	tests, err := h.service.GetList(resourcesCurriculumEndPoint+queryParams, testsKey, false, true)

	if err != nil {
		http.Error(responseWriter, fmt.Sprintf("Error fetching tests: %v", err), http.StatusInternalServerError)
		return
	}

	local_repo.ExecuteTemplate(testRowTemplate, responseWriter, tests)
}
