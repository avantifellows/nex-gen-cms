package handlers

import (
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strings"

	"github.com/avantifellows/nex-gen-cms/internal/constants"
	"github.com/avantifellows/nex-gen-cms/internal/dto"
	"github.com/avantifellows/nex-gen-cms/internal/models"
	local_repo "github.com/avantifellows/nex-gen-cms/internal/repositories/local"
	"github.com/avantifellows/nex-gen-cms/internal/services"
	"github.com/avantifellows/nex-gen-cms/utils"
	"github.com/thoas/go-funk"
)

const CURRICULUM_DROPDOWN_NAME = "curriculum-dropdown"
const GRADE_DROPDOWN_NAME = "grade-dropdown"
const SUBJECT_DROPDOWN_NAME = "subject-dropdown"

const chaptersEndPoint = "/chapter"

const chaptersKey = "chapters"

const chaptersTemplate = "chapters.html"
const chapterRowTemplate = "chapter_row.html"
const baseTemplate = "home.html"
const editChapterTemplate = "edit_chapter.html"
const updateSuccessTemplate = "update_success.html"
const chapterTemplate = "chapter.html"

type ChaptersHandler struct {
	chaptersService *services.Service[models.Chapter]
	topicsService   *services.Service[models.Topic]
}

// NewChaptersHandler creates a new instance of ChaptersHandler
func NewChaptersHandler(chaptersService *services.Service[models.Chapter],
	topicsService *services.Service[models.Topic]) *ChaptersHandler {
	return &ChaptersHandler{
		chaptersService: chaptersService,
		topicsService:   topicsService,
	}
}

var chapterSortState = dto.SortState{
	Column: "0",
	Order:  constants.SortOrderAsc,
}

var topicSortState = dto.SortState{
	Column: "0",
	Order:  constants.SortOrderAsc,
}

func (h *ChaptersHandler) LoadChapters(responseWriter http.ResponseWriter, request *http.Request) {
	updateSortState(request, &chapterSortState)
	local_repo.ExecuteTemplate(chaptersTemplate, responseWriter, chapterSortState)
}

func updateSortState(request *http.Request, sortState *dto.SortState) {
	urlVals := request.URL.Query()
	const queryParam = "sortColumn"

	// change sort state if it is called due to click on any column header
	if urlVals.Has(queryParam) {
		sortColumn := urlVals.Get(queryParam)

		// if same column is clicked, toggle the order
		if sortColumn == sortState.Column {
			if sortState.Order == constants.SortOrderAsc {
				sortState.Order = constants.SortOrderDesc
			} else {
				sortState.Order = constants.SortOrderAsc
			}
		} else {
			// If a new column is clicked, default to ascending order
			sortState.Column = sortColumn
			sortState.Order = constants.SortOrderAsc
		}
	}
}

func (h *ChaptersHandler) GetChapters(responseWriter http.ResponseWriter, request *http.Request) {
	curriculumId, gradeId, subjectId := getCurriculumGradeSubjectIds(request.URL.Query())
	if curriculumId == 0 || gradeId == 0 || subjectId == 0 {
		return
	}

	queryParams := fmt.Sprintf("?curriculum_id=%d&grade_id=%d&subject_id=%d", curriculumId, gradeId, subjectId)
	chapters, err := h.chaptersService.GetList(chaptersEndPoint+queryParams, chaptersKey, false, true)

	// set curriculum id on each chapter
	for _, ch := range *chapters {
		ch.CurriculumID = curriculumId
	}

	if err != nil {
		http.Error(responseWriter, fmt.Sprintf("Error fetching chapters: %v", err), http.StatusInternalServerError)
		return
	}

	filteredChapters := funk.Filter(*chapters, func(chapter *models.Chapter) bool {
		return (*chapter).CurriculumID == curriculumId && (*chapter).GradeID == gradeId &&
			(*chapter).SubjectID == subjectId
	})
	typecastedChapters := filteredChapters.([]*models.Chapter)

	h.getTopics(responseWriter, typecastedChapters)
	sortChapters(typecastedChapters)

	local_repo.ExecuteTemplate(chapterRowTemplate, responseWriter, typecastedChapters)
}

func (h *ChaptersHandler) getTopics(responseWriter http.ResponseWriter, chapterPtrs []*models.Chapter) {
	topics, err := h.topicsService.GetList(topicsEndPoint, topicsKey, false, false)
	if err != nil {
		http.Error(responseWriter, fmt.Sprintf("Error fetching topics: %v", err), http.StatusInternalServerError)
	} else {
		associateTopicsWithChapters(chapterPtrs, *topics)
	}
}

func associateTopicsWithChapters(chapterPtrs []*models.Chapter, topicPtrs []*models.Topic) {
	// Create a map to quickly lookup chapters by their ID
	chapterPtrsMap := make(map[int16]*models.Chapter)

	// Fill the map with the address of each chapter
	for _, chapterPtr := range chapterPtrs {
		chapterPtrsMap[chapterPtr.ID] = chapterPtr
		// clear topics data, because it will be refilled in next step based on latest data
		chapterPtr.Topics = chapterPtr.Topics[:0]
	}

	// Loop through each topic and assign it to the corresponding chapter
	for _, topicPtr := range topicPtrs {
		if chapterPtr, exists := chapterPtrsMap[topicPtr.ChapterID]; exists {
			chapterPtr.Topics = append(chapterPtr.Topics, *topicPtr)
		}
	}
}

func (h *ChaptersHandler) EditChapter(responseWriter http.ResponseWriter, request *http.Request) {
	selectedChapterPtr, code, err := h.getChapter(request)
	if err != nil {
		http.Error(responseWriter, err.Error(), code)
		return
	}

	data := dto.HomeChapterData{
		InitialLoad: false,
		ChapterPtr:  selectedChapterPtr,
	}
	local_repo.ExecuteTemplates(baseTemplate, editChapterTemplate, responseWriter, data)
}

func (h *ChaptersHandler) UpdateChapter(responseWriter http.ResponseWriter, request *http.Request) {
	chapterIdStr := request.FormValue("id")
	chapterId, err := utils.StringToIntType[int16](chapterIdStr)
	if err != nil {
		http.Error(responseWriter, "Invalid Chapter ID", http.StatusBadRequest)
		return
	}

	chapterName := request.FormValue("name")
	chapterCode := request.FormValue("code")

	dummyChapterPtr := &models.Chapter{}
	chapterMap := dummyChapterPtr.BuildMap(chapterCode, chapterName)

	_, err = h.chaptersService.UpdateObject(chapterIdStr, chaptersEndPoint, chapterMap, chaptersKey,
		func(chapter *models.Chapter) bool {
			return (*chapter).ID == chapterId
		})
	if err != nil {
		http.Error(responseWriter, fmt.Sprintf("Error updating chapter: %v", err), http.StatusInternalServerError)
		return
	}

	local_repo.ExecuteTemplate(updateSuccessTemplate, responseWriter, "Chapter")
}

func (h *ChaptersHandler) AddChapter(responseWriter http.ResponseWriter, request *http.Request) {
	chapterCode := request.FormValue("code")
	chapterName := request.FormValue("name")
	curriculumIdStr := request.FormValue(CURRICULUM_DROPDOWN_NAME)
	curriculumId, err := utils.StringToIntType[int16](curriculumIdStr)
	if err != nil {
		http.Error(responseWriter, "Invalid Curriculum ID", http.StatusBadRequest)
		return
	}
	gradeIdStr := request.FormValue(GRADE_DROPDOWN_NAME)
	gradeId, err := utils.StringToIntType[int8](gradeIdStr)
	if err != nil {
		http.Error(responseWriter, "Invalid Grade ID", http.StatusBadRequest)
		return
	}
	subjectIdStr := request.FormValue(SUBJECT_DROPDOWN_NAME)
	subjectId, err := utils.StringToIntType[int8](subjectIdStr)
	if err != nil {
		http.Error(responseWriter, "Invalid Subject ID", http.StatusBadRequest)
		return
	}
	newChapterPtr := models.NewChapter(chapterCode, chapterName, curriculumId, gradeId, subjectId)

	newChapterPtr, err = h.chaptersService.AddObject(newChapterPtr, chaptersKey, chaptersEndPoint)
	if err != nil {
		http.Error(responseWriter, fmt.Sprintf("Error adding chapter: %v", err), http.StatusInternalServerError)
		return
	}

	chapterPtrs := []*models.Chapter{newChapterPtr}
	local_repo.ExecuteTemplate(chapterRowTemplate, responseWriter, chapterPtrs)
}

func (h *ChaptersHandler) DeleteChapter(responseWriter http.ResponseWriter, request *http.Request) {
	chapterIdStr := request.URL.Query().Get("id")
	chapterId, err := utils.StringToIntType[int16](chapterIdStr)
	if err != nil {
		http.Error(responseWriter, "Invalid Chapter ID", http.StatusBadRequest)
		return
	}
	err = h.chaptersService.DeleteObject(chapterIdStr, func(c *models.Chapter) bool {
		return c.ID != chapterId
	}, chaptersKey, chaptersEndPoint)

	// If http error is thrown from here then target row won't be removed by htmx code
	if err != nil {
		http.Error(responseWriter, err.Error(), http.StatusInternalServerError)
	}
}

func getCurriculumGradeSubjectIds(urlValues url.Values) (int16, int8, int8) {
	// these query parameters can be queried by element names only, not ids
	curriculumId, err := utils.StringToIntType[int16](urlValues.Get(CURRICULUM_DROPDOWN_NAME))
	if err != nil {
		fmt.Println("Selected Curriculum is invalid")
	}
	gradeId, err := utils.StringToIntType[int8](urlValues.Get(GRADE_DROPDOWN_NAME))
	if err != nil {
		fmt.Println("Selected Grade is invalid")
	}
	subjectId, err := utils.StringToIntType[int8](urlValues.Get(SUBJECT_DROPDOWN_NAME))
	if err != nil {
		fmt.Println("Selected Subject is invalid")
	}
	return curriculumId, gradeId, subjectId
}

func sortChapters(chapterPtrs []*models.Chapter) {
	slices.SortStableFunc(chapterPtrs, func(c1, c2 *models.Chapter) int {
		var sortResult int
		switch chapterSortState.Column {
		case "1":
			c1Suffix := utils.ExtractNumericSuffix(c1.Code)
			c2Suffix := utils.ExtractNumericSuffix(c2.Code)
			// if numeric suffix found for both chapters then perform their integer comparison
			if c1Suffix > 0 && c2Suffix > 0 {
				sortResult = c1Suffix - c2Suffix
			} else {
				// perform string comparison of codes, because numeric suffixes could not be found
				sortResult = strings.Compare(c1.Code, c2.Code)
			}
		case "2":
			sortResult = strings.Compare(c1.Name, c2.Name)
		case "3":
			sortResult = int(c1.TopicCount() - c2.TopicCount())
		default:
			sortResult = 0
		}

		if chapterSortState.Order == constants.SortOrderDesc {
			sortResult = -sortResult
		}
		return sortResult
	})
}

func (h *ChaptersHandler) getChapter(request *http.Request) (*models.Chapter, int, error) {
	chapterIdStr := request.URL.Query().Get("id")
	chapterId, err := utils.StringToIntType[int16](chapterIdStr)
	if err != nil {
		return nil, http.StatusBadRequest, fmt.Errorf("invalid Chapter ID: %w", err)
	}

	selectedChapterPtr, err := h.chaptersService.GetObject(chapterIdStr,
		func(chapter *models.Chapter) bool {
			return (*chapter).ID == chapterId
		}, chaptersKey, chaptersEndPoint)
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("error fetching chapter: %v", err)
	}

	return selectedChapterPtr, http.StatusOK, nil
}

func (h *ChaptersHandler) GetChapter(responseWriter http.ResponseWriter, request *http.Request) {
	selectedChapterPtr, code, err := h.getChapter(request)
	if err != nil {
		http.Error(responseWriter, err.Error(), code)
		return
	}

	data := dto.HomeChapterData{
		InitialLoad: false,
		ChapterPtr:  selectedChapterPtr,
	}
	local_repo.ExecuteTemplates(baseTemplate, chapterTemplate, responseWriter, data)
}

func (h *ChaptersHandler) LoadTopics(responseWriter http.ResponseWriter, request *http.Request) {
	chapterIdStr := request.URL.Query().Get("id")
	updateSortState(request, &topicSortState)

	data := dto.TopicsData{
		ChapterId:       chapterIdStr,
		TopicsSortState: topicSortState,
	}
	local_repo.ExecuteTemplate(topicsTemplate, responseWriter, data)
}

func (h *ChaptersHandler) GetTopics(responseWriter http.ResponseWriter, request *http.Request) {
	selectedChapterPtr, code, err := h.getChapter(request)
	if err != nil {
		http.Error(responseWriter, err.Error(), code)
		return
	}
	if len(selectedChapterPtr.Topics) == 0 {
		h.getTopics(responseWriter, []*models.Chapter{selectedChapterPtr})
	}
	sortTopics(selectedChapterPtr.Topics)
	local_repo.ExecuteTemplate(topicRowTemplate, responseWriter, selectedChapterPtr.Topics)
}
