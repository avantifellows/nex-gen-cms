package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/avantifellows/nex-gen-cms/internal/constants"
	"github.com/avantifellows/nex-gen-cms/internal/dto"
	"github.com/avantifellows/nex-gen-cms/internal/handlers/handlerutils"
	"github.com/avantifellows/nex-gen-cms/internal/models"
	"github.com/avantifellows/nex-gen-cms/internal/services"
	"github.com/avantifellows/nex-gen-cms/internal/views"
	"github.com/avantifellows/nex-gen-cms/utils"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
	"github.com/thoas/go-funk"
)

const TESTTYPE_DROPDOWN_NAME = "testtype-dropdown"

const testsTemplate = "tests.html"
const testRowTemplate = "test_row.html"
const testTemplate = "test.html"
const testProblemRowTemplate = "test_problem_row.html"
const addTestTemplate = "add_test.html"
const testTypeOptionsTemplate = "test_type_options.html"
const testChipEditorTemplate = "test_chip_editor.html"
const addTestDestProblemRowWithoutHeadersTemplate = "dest_problem_row_without_headers.html"
const addTestDestProblemRowWithSubtypeTemplate = "dest_problem_row_with_subtype.html"
const addTestDestProblemRowWithHeadersTemplate = "dest_problem_row_with_headers.html"
const addTestDestProblemRowTemplate = "dest_problem_row.html"
const addTestDestSubtypeRowTemplate = "dest_subtype_row.html"
const addTestDestSubjectRowTemplate = "dest_subject_row.html"
const chipBoxCellTemplate = "chip_box_cells.html"
const addTestModalTemplate = "add_test_modal.html"
const curriculumGradeSelectsTemplate = "curriculum_grade_selects.html"
const addCurriculumGradeSelectsTemplate = "add_curriculum_grade_selects.html"
const questionPaperTemplate = "question_paper.html"
const answerSolutionSheetTemplate = "answer_sheet.html"

const resourcesEndPoint = "resource"
const resourcesCurriculumEndPoint = "resources/curriculum"
const testProblemsEndPoint = "resource/test/%d/problems?lang_code=en&" + QUERY_PARAM_CURRICULUM_ID + "=%s"
const testRulesEndPoint = "test-rule"

const testsKey = "tests"
const testRulesKey = "testRules"

type TestsHandler struct {
	testsService     *services.Service[models.Test]
	subjectsService  *services.Service[models.Subject]
	problemsService  *services.Service[models.Problem]
	testRulesService *services.Service[models.TestRule]
}

func NewTestsHandler(testsService *services.Service[models.Test], subjectsService *services.Service[models.Subject],
	problemsService *services.Service[models.Problem], testRulesService *services.Service[models.TestRule]) *TestsHandler {
	return &TestsHandler{
		testsService:     testsService,
		subjectsService:  subjectsService,
		problemsService:  problemsService,
		testRulesService: testRulesService,
	}
}

func (h *TestsHandler) LoadTests(responseWriter http.ResponseWriter, request *http.Request) {
	views.ExecuteTemplates(responseWriter, nil, template.FuncMap{
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

	filtered := (*tests)[:0] // zero-length slice, same backing array
	// set curriculum & grade id on each test
	for _, test := range *tests {
		// skip archived tests
		if test.Status == constants.ResourceStatusArchived {
			continue
		}
		test.SetCurriculumGrade(curriculumId, gradeId)
		filtered = append(filtered, test)
	}
	*tests = filtered // assign filtered slice back to original

	sortTests(*tests, sortColumn, sortOrder)
	views.ExecuteTemplate(testRowTemplate, responseWriter, tests, nil)
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
		CurriculumID: selectedTestPtr.CurriculumGrades[0].CurriculumID,
		GradeID:      selectedTestPtr.CurriculumGrades[0].GradeID,
		TestPtr:      selectedTestPtr,
	}

	views.ExecuteTemplates(responseWriter, data, nil, baseTemplate, testTemplate)
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
	if err == nil {
		gradeId, err := utils.StringToIntType[int8](urlVals.Get("grade_id"))
		if err == nil {
			selectedTestPtr.SetCurriculumGrade(curriculumId, gradeId)
		}
	}

	// Fill subject names in test
	h.fillSubjectNames(responseWriter, selectedTestPtr)

	return selectedTestPtr, http.StatusOK, nil
}

func (h *TestsHandler) fillSubjectNames(responseWriter http.ResponseWriter, testPtr *models.Test) {
	subjectPtrs, err := h.subjectsService.GetList(handlerutils.SubjectsEndPoint, handlerutils.SubjectsKey, false, false)
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

	urlVals := request.URL.Query()
	subjectId, err := utils.StringToIntType[int8](urlVals.Get("subject_id"))
	if err != nil {
		fmt.Println("invalid subject id")
		return
	}

	*problems = funk.Filter(*problems, func(p *models.Problem) bool {
		return p.SubjectID == subjectId
	}).([]*models.Problem)

	// Passing custom function add to use in template for serial number by adding 1 to index
	views.ExecuteTemplate(testProblemRowTemplate, responseWriter, problems, template.FuncMap{
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
	*problems = funk.Filter(*problems, func(p *models.Problem) bool {
		return p.Status != constants.ResourceStatusArchived
	}).([]*models.Problem)

	h.fillProblemSubjects(responseWriter, problems)

	return problems
}

func (h *TestsHandler) fillProblemSubjects(responseWriter http.ResponseWriter, problems *[]*models.Problem) {
	subjectPtrs, err := h.subjectsService.GetList(handlerutils.SubjectsEndPoint, handlerutils.SubjectsKey, false, false)
	if err != nil {
		http.Error(responseWriter, fmt.Sprintf("Error fetching subjects: %v", err), http.StatusInternalServerError)
	} else {
		// Create a map to quickly lookup subjects by their ID
		subjectIdToSubMap := make(map[int8]models.Subject)

		// fill the map with each subject
		for _, subjectPtr := range *subjectPtrs {
			subjectIdToSubMap[subjectPtr.ID] = *subjectPtr
		}
		// loop through problems and update subject inside it
		for _, problem := range *problems {
			problem.Subject = subjectIdToSubMap[problem.SubjectID]
		}
	}
}

func (h *TestsHandler) AddTest(responseWriter http.ResponseWriter, request *http.Request) {
	if err := request.ParseForm(); err != nil {
		http.Error(responseWriter, "Invalid form data", http.StatusBadRequest)
		return
	}

	curriculums := request.Form["curriculum[]"]
	grades := request.Form["grade[]"]
	testType := request.FormValue("modal-testType")

	var curriculumGrades []models.CurriculumGrade
	for i := range curriculums {
		curriculumId, err := utils.StringToIntType[int16](curriculums[i])
		if err != nil {
			fmt.Printf("invalid curriculum id at index %d", i)
			return
		}

		gradeId, err := utils.StringToIntType[int8](grades[i])
		if err != nil {
			fmt.Printf("invalid grade id at index %d", i)
			return
		}

		curriculumGrades = append(curriculumGrades, models.CurriculumGrade{
			CurriculumID: curriculumId,
			GradeID:      gradeId,
		})
	}

	examId, err := utils.StringToIntType[int8](request.FormValue("modal-examType"))
	if err != nil {
		fmt.Println("invalid exam id")
		return
	}

	testRule, err := h.getTestRule(testType, examId)
	if err != nil {
		fmt.Println(err.Error())
	}
	data := dto.HomeData{
		TestPtr: &models.Test{
			ExamIDs:          []int8{examId},
			Subtype:          testType,
			CurriculumGrades: curriculumGrades,
		},
		TestRule: testRule,
	}

	views.ExecuteTemplates(responseWriter, data, template.FuncMap{
		"split":             strings.Split,
		"slice":             utils.Slice,
		"seq":               utils.Seq,
		"getName":           getTestName,
		"add":               utils.Add,
		"joinInt16":         utils.JoinInt16,
		"dict":              utils.Dict,
		"getDisplaySubtype": utils.DisplaySubtype,
		"toJson":            utils.ToJson,
		"getParentId":       getParentSubjectId,
	}, baseTemplate, addTestTemplate, problemTypeOptionsTemplate, testTypeOptionsTemplate, testChipEditorTemplate,
		addTestDestSubjectRowTemplate, addTestDestSubtypeRowTemplate, addTestDestProblemRowTemplate, chipBoxCellTemplate)
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

	subjectPtr, statusCode, err := handlerutils.FetchSelectedSubject(utils.IntToString(problemPtr.SubjectID),
		h.subjectsService)
	if err != nil {
		http.Error(responseWriter, err.Error(), statusCode)
		return
	}
	// set subject as its name is required to be displayed under right hand side table for add/edit test screen
	problemPtr.Subject = *subjectPtr

	insertAfterId := request.FormValue("insert-after-id")
	subjectExists := request.FormValue("subject-exists") == "true"
	subtypeExists := request.FormValue("subtype-exists") == "true"
	readOnlyMarks := request.FormValue("read-only-marks") == "true"

	var filename string
	var data any

	switch {
	case !subjectExists && !subtypeExists:
		// Need subject + subtype header
		filename = addTestDestProblemRowWithHeadersTemplate
		data = map[string]any{
			"Problem":       problemPtr,
			"ReadOnlyMarks": readOnlyMarks,
		}

	case subjectExists && !subtypeExists:
		// Only subtype header needed
		filename = addTestDestProblemRowWithSubtypeTemplate
		data = map[string]any{
			"Problem":       problemPtr,
			"InsertAfterId": insertAfterId,
			"ReadOnlyMarks": readOnlyMarks,
		}

	case subtypeExists:
		// Just problem row
		filename = addTestDestProblemRowWithoutHeadersTemplate
		data = map[string]any{
			"Problem":       problemPtr,
			"InsertAfterId": insertAfterId,
			"ReadOnlyMarks": readOnlyMarks,
		}
	}

	views.ExecuteTemplates(responseWriter, data, template.FuncMap{
		"getParentName":     getParentSubjectName,
		"getParentId":       getParentSubjectId,
		"joinInt16":         utils.JoinInt16,
		"dict":              utils.Dict,
		"getDisplaySubtype": utils.DisplaySubtype,
	}, filename, addTestDestSubjectRowTemplate, addTestDestSubtypeRowTemplate, addTestDestProblemRowTemplate, chipBoxCellTemplate)
}

func (h *TestsHandler) CreateTest(responseWriter http.ResponseWriter, request *http.Request) {
	// Declare a variable to hold the parsed JSON
	var testObj models.Test

	// Decode the JSON body into the testData map
	err := json.NewDecoder(request.Body).Decode(&testObj)
	if err != nil {
		http.Error(responseWriter, "Error parsing JSON", http.StatusBadRequest)
		return
	}

	_, err = h.testsService.AddObject(testObj, testsKey, resourcesEndPoint)
	if err != nil {
		http.Error(responseWriter, fmt.Sprintf("Error adding test: %v", err), http.StatusInternalServerError)
		return
	}
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

	var testRule *models.TestRule // nil by default
	if len(selectedTestPtr.ExamIDs) > 0 {
		tr, err := h.getTestRule(selectedTestPtr.Subtype, selectedTestPtr.ExamIDs[0])
		if err != nil {
			fmt.Println(err.Error())
		} else {
			testRule = tr
		}
	}

	data := dto.HomeData{
		TestPtr:  selectedTestPtr,
		Problems: problemsMap,
		TestRule: testRule,
	}

	views.ExecuteTemplates(responseWriter, data, template.FuncMap{
		"split":             strings.Split,
		"slice":             utils.Slice,
		"seq":               utils.Seq,
		"getName":           getTestName,
		"add":               utils.Add,
		"joinInt16":         utils.JoinInt16,
		"dict":              utils.Dict,
		"getDisplaySubtype": utils.DisplaySubtype,
		"toJson":            utils.ToJson,
		"getParentId":       getParentSubjectId,
	}, baseTemplate, addTestTemplate, problemTypeOptionsTemplate, testTypeOptionsTemplate, testChipEditorTemplate,
		addTestDestSubjectRowTemplate, addTestDestSubtypeRowTemplate, addTestDestProblemRowTemplate, chipBoxCellTemplate)
}

func (h *TestsHandler) UpdateTest(responseWriter http.ResponseWriter, request *http.Request) {
	// Declare a variable to hold the parsed JSON
	var testObj models.Test

	// Decode the JSON body into the testData map
	err := json.NewDecoder(request.Body).Decode(&testObj)
	if err != nil {
		http.Error(responseWriter, "Error parsing JSON", http.StatusBadRequest)
		return
	}

	testIdStr := request.URL.Query().Get("id")
	testId := utils.StringToInt(testIdStr)

	_, err = h.testsService.UpdateObject(testIdStr, resourcesEndPoint, testObj, testsKey,
		func(test *models.Test) bool {
			return (*test).ID == testId
		})
	if err != nil {
		http.Error(responseWriter, fmt.Sprintf("Error updating test: %v", err), http.StatusInternalServerError)
		return
	}
}

func (h *TestsHandler) ArchiveTest(responseWriter http.ResponseWriter, request *http.Request) {
	testIdStr := request.URL.Query().Get("id")
	testId := utils.StringToInt(testIdStr)
	body := map[string]string{
		"cms_status": constants.ResourceStatusArchived,
	}

	err := h.testsService.ArchiveObject(testIdStr, resourcesEndPoint, body, testsKey,
		func(test *models.Test) bool {
			return test.ID != testId
		})
	if err != nil {
		http.Error(responseWriter, fmt.Sprintf("Error archiving test: %v", err), http.StatusInternalServerError)
		return
	}
}

func getTestName(t models.Test, lang string) string {
	return t.GetNameByLang(lang)
}

func (h *TestsHandler) AddTestModal(responseWriter http.ResponseWriter, request *http.Request) {
	var data dto.AddTestDialogData

	query := request.URL.Query()

	if len(query) > 0 {
		subtype := query.Get("subtype")
		examIdStr := query.Get("exam_id")
		curriculumGradesStr := query.Get("curriculum_grades")

		// Decode exam_id
		examId, _ := utils.StringToIntType[int8](examIdStr)

		// Decode curriculum_grades JSON string
		var curriculumGrades []models.CurriculumGrade
		if err := json.Unmarshal([]byte(curriculumGradesStr), &curriculumGrades); err != nil {
			log.Println("Error decoding curriculum_grades:", err)
		}

		data = dto.AddTestDialogData{
			Subtype:          subtype,
			CurriculumGrades: curriculumGrades,
			ExamID:           examId,
		}

	} else {
		data = dto.AddTestDialogData{}
	}

	views.ExecuteTemplates(responseWriter, data, template.FuncMap{
		"slice": utils.Slice,
		"add":   utils.Add,
		"dict":  utils.Dict,
	}, addTestModalTemplate, testTypeOptionsTemplate, curriculumGradeSelectsTemplate)
}

func (h *TestsHandler) AddCurriculumGradeDropdowns(responseWriter http.ResponseWriter, request *http.Request) {
	views.ExecuteTemplates(responseWriter, nil, nil, addCurriculumGradeSelectsTemplate, curriculumGradeSelectsTemplate)
}

func (h *TestsHandler) getTestRule(testType string, examId int8) (*models.TestRule, error) {
	testRules, err := h.testRulesService.GetList(testRulesEndPoint, testRulesKey, false, false)
	if err != nil {
		return nil, fmt.Errorf("error fetching test rules: %v", err)
	}

	for _, rule := range *testRules {
		if rule.ExamID == examId && rule.TestType == testType {
			return rule, nil
		}
	}

	return nil, fmt.Errorf("no matching test rule found for examID=%d and testType=%s", examId, testType)
}

func (h *TestsHandler) DownloadPdf(responseWriter http.ResponseWriter, request *http.Request) {
	selectedTestPtr, code, err := h.getTest(responseWriter, request)
	if err != nil {
		http.Error(responseWriter, err.Error(), code)
		return
	}
	problems := h.getTestProblems(responseWriter, request)

	problemsMap := make(map[int]*models.Problem)
	for _, p := range *problems {
		problemsMap[p.ID] = p
	}

	pdfType := request.URL.Query().Get("type") // "questions" or "answers"

	var pdfTemplate, headerTxt, pdfSuffix string
	var testRule *models.TestRule
	if pdfType == "questions" {
		pdfTemplate = questionPaperTemplate
		headerTxt = selectedTestPtr.DisplaySubtype()
		pdfSuffix = "Question Paper"

		if len(selectedTestPtr.ExamIDs) > 0 {
			testRule, err = h.getTestRule(selectedTestPtr.Subtype, selectedTestPtr.ExamIDs[0])
			if err != nil {
				fmt.Println(err.Error())
			}
		}

	} else if pdfType == "answers" {
		pdfTemplate = answerSolutionSheetTemplate
		headerTxt = selectedTestPtr.DisplaySubtype() + " - Answer Sheet"
		pdfSuffix = "Answer Sheet"
	}

	// Load template
	tmplPath := filepath.Join(constants.GetHtmlFolderPath(), pdfTemplate)
	tmpl, err := template.New(pdfTemplate).Funcs(template.FuncMap{
		"getName":               getTestName,
		"add":                   utils.Add,
		"labels":                optionLabels,
		"capitalize":            utils.Capitalize,
		"problemDisplaySubtype": utils.DisplaySubtype,
		"stringToInt":           utils.StringToInt,
		"trim":                  strings.TrimSpace,
	}).ParseFiles(tmplPath)
	if err != nil {
		http.Error(responseWriter, "Template parsing error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	data := dto.PaperData{
		TestPtr:     selectedTestPtr,
		ProblemsMap: problemsMap,
		TestRule:    testRule,
	}

	// Render HTML to buffer
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		http.Error(responseWriter, "Template execution error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	htmlContent := buf.String()

	// for tailwind css lib. Including it from here, because chromedp is unable to resolve it using relative path in html <link>
	cssBytes, err := os.ReadFile("web/static/css/output.css")
	if err != nil {
		http.Error(responseWriter, "CSS read error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	htmlContent = strings.Replace(htmlContent, "</head>", "<style>"+string(cssBytes)+"</style></head>", 1)

	headerHTML := fmt.Sprintf(`
		<div style="width:100%%; font-size:12px; font-family:Arial; text-align:center; padding:0 40px;">
			<div style="margin-bottom:4px;">%s</div>
			<hr style="border:0; border-top:1px solid #000; margin:4px 0 0 0;">
		</div>`, headerTxt)

	// Create Chrome context
	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	// Set a global timeout (for safety)
	ctx, cancel = context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	var pdfData []byte

	tasks := chromedp.Tasks{
		// Set page content
		chromedp.Navigate("data:text/html," + url.PathEscape(htmlContent)),

		// Inject HTML into page
		chromedp.ActionFunc(func(ctx context.Context) error {
			script := `document.documentElement.innerHTML = ` + strconv.Quote(htmlContent)
			return chromedp.Evaluate(script, nil).Do(ctx)
		}),

		// Wait for MathJax to render fully
		chromedp.ActionFunc(func(ctx context.Context) error {
			js := `
            new Promise(resolve => {
                function check() {
                    if (window.MathJax && MathJax.typesetPromise) {
                        MathJax.typesetPromise().then(() => resolve(true));
                    } else {
                        setTimeout(check, 200);
                    }
                }
                check();
            });
            `
			return chromedp.Evaluate(js, nil).Do(ctx)
		}),

		// Generate PDF using CDP low-level API
		chromedp.ActionFunc(func(ctx context.Context) error {
			var err error
			pdfData, _, err = page.PrintToPDF().
				WithPrintBackground(true).
				WithPaperWidth(8.27).   // A4 width in inches
				WithPaperHeight(11.69). // A4 height in inches
				WithMarginTop(0.5).
				WithMarginBottom(1.0).
				WithMarginLeft(0.3).
				WithMarginRight(0.3).
				WithDisplayHeaderFooter(true).
				WithHeaderTemplate(headerHTML).
				WithFooterTemplate(`
				<div style="width:100%; font-size:12px; font-family:Arial; position:relative; height:30px; padding:0 40px;">
					<div style="position:absolute; top:0; left:40px; right:40px;">
						<hr style="border:0; border-top:1px solid #000; margin:0;">
					</div>
					<div style="display:flex; justify-content:space-between; align-items:flex-end; height:100%; color:#444;">
						<span></span>
						<span>Avanti Fellows. All rights reserved.</span>
						<span>Page - <span class="pageNumber"></span> / <span class="totalPages"></span></span>
					</div>
				</div>
				`).
				Do(ctx)
			return err
		}),
	}

	if err := chromedp.Run(ctx, tasks); err != nil {
		http.Error(responseWriter, "PDF generation failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Send as response
	responseWriter.Header().Set("Content-Type", "application/pdf")
	responseWriter.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s - %s.pdf"`,
		selectedTestPtr.GetNameByLang("en"), pdfSuffix))
	_, _ = responseWriter.Write(pdfData)
}

func optionLabels() []string {
	return []string{"A)", "B)", "C)", "D)", "E)", "F)", "G)", "H)", "I)", "J)"}
}

func (h *TestsHandler) CopyTest(responseWriter http.ResponseWriter, request *http.Request) {
	selectedTestPtr, code, err := h.getTest(responseWriter, request)
	if err != nil {
		http.Error(responseWriter, err.Error(), code)
		return
	}

	// Make a copy so the original is not mutated
	copiedTest := *selectedTestPtr
	copiedTest.ID = 0

	problems := h.getTestProblems(responseWriter, request)
	if problems == nil {
		return
	}
	problemsMap := make(map[int]*models.Problem)
	for _, p := range *problems {
		problemsMap[p.ID] = p
	}

	data := dto.HomeData{
		TestPtr:  &copiedTest,
		Problems: problemsMap,
	}

	views.ExecuteTemplates(responseWriter, data, template.FuncMap{
		"split":             strings.Split,
		"slice":             utils.Slice,
		"seq":               utils.Seq,
		"getName":           getTestName,
		"add":               utils.Add,
		"joinInt16":         utils.JoinInt16,
		"dict":              utils.Dict,
		"getDisplaySubtype": utils.DisplaySubtype,
		"toJson":            utils.ToJson,
		"getParentId":       getParentSubjectId,
	}, baseTemplate, addTestTemplate, problemTypeOptionsTemplate, testTypeOptionsTemplate, testChipEditorTemplate,
		addTestDestSubjectRowTemplate, addTestDestSubtypeRowTemplate, addTestDestProblemRowTemplate, chipBoxCellTemplate)
}
