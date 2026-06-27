package handlers

import (
	"fmt"
	"net/http"
	"slices"
	"strings"
	"text/template"

	"github.com/thoas/go-funk"

	"github.com/avantifellows/nex-gen-cms/internal/constants"
	"github.com/avantifellows/nex-gen-cms/internal/dto"
	"github.com/avantifellows/nex-gen-cms/internal/handlers/handlerutils"
	"github.com/avantifellows/nex-gen-cms/internal/models"
	"github.com/avantifellows/nex-gen-cms/internal/services"
	"github.com/avantifellows/nex-gen-cms/internal/views"
	"github.com/avantifellows/nex-gen-cms/utils"
)

const chaptersEndPoint = "chapter"

const chaptersKey = "chapters"

const chaptersTemplate = "chapters.html"
const chapterRowTemplate = "chapter_row.html"
const editChapterTemplate = "edit_chapter.html"
const updateSuccessTemplate = "update_success.html"
const chapterTemplate = "chapter.html"
const chapterDropdownTemplate = "chapter_dropdown.html"
const topicDropdownOptionalTemplate = "topic_dropdown_optional.html"

type ChaptersHandler struct {
	chaptersService *services.Service[models.Chapter]
	topicsService   *services.Service[models.Topic]
}

func NewChaptersHandler(chaptersService *services.Service[models.Chapter],
	topicsService *services.Service[models.Topic]) *ChaptersHandler {
	return &ChaptersHandler{
		chaptersService: chaptersService,
		topicsService:   topicsService,
	}
}

func (h *ChaptersHandler) LoadChapters(responseWriter http.ResponseWriter, _ *http.Request) {
	views.ExecuteTemplates(responseWriter, nil, nil, baseTemplate, chaptersTemplate)
}

func (h *ChaptersHandler) GetChapters(responseWriter http.ResponseWriter, request *http.Request) {
	urlVals := request.URL.Query()
	curriculumID, gradeID, subjectID := getCurriculumGradeSubjectIDs(urlVals)
	if curriculumID == 0 || gradeID == 0 || subjectID == 0 {
		return
	}

	queryParams := fmt.Sprintf("?"+QueryParamCurriculumID+"=%d&grade_id=%d&subject_id=%d", curriculumID, gradeID, subjectID)
	chapters, err := h.chaptersService.GetList(chaptersEndPoint+queryParams, chaptersKey, false, true)

	if err != nil {
		http.Error(responseWriter, fmt.Sprintf("Error fetching chapters: %v", err), http.StatusInternalServerError)
		return
	}
	*chapters = funk.Filter(*chapters, func(c *models.Chapter) bool {
		return c.StatusID != constants.StatusArchived
	}).([]*models.Chapter)

	for _, chapterPtr := range *chapters {
		chapterPtr.CurriculumID = curriculumID
	}

	h.getTopics(responseWriter, *chapters)

	sortColumn := urlVals.Get("sortColumn")
	sortOrder := urlVals.Get("sortOrder")
	sortChapters(*chapters, sortColumn, sortOrder)

	view := urlVals.Get("view")
	var filename string
	if view == "list" {
		filename = chapterRowTemplate
	} else {
		filename = chapterDropdownTemplate
	}
	views.ExecuteTemplate(filename, responseWriter, chapters, template.FuncMap{
		"getName": getChapterName,
	})
}

func getChapterName(ch models.Chapter, lang string) string {
	return ch.GetNameByLang(lang)
}

func (h *ChaptersHandler) getTopics(responseWriter http.ResponseWriter, chapterPtrs []*models.Chapter) {
	topics, err := h.topicsService.GetList(handlerutils.TopicsEndPoint, handlerutils.TopicsKey, false, false)
	if err != nil {
		http.Error(responseWriter, fmt.Sprintf("Error fetching topics: %v", err), http.StatusInternalServerError)
	} else {
		*topics = funk.Filter(*topics, func(t *models.Topic) bool {
			return t.StatusID != constants.StatusArchived
		}).([]*models.Topic)

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
		if chapterPtr, exists := chapterPtrsMap[topicPtr.ChapterID]; exists &&
			topicPtr.HasCurriculumID(chapterPtr.CurriculumID) {
			chapterPtr.Topics = append(chapterPtr.Topics, topicPtr)
		}
	}
}

func (h *ChaptersHandler) EditChapter(responseWriter http.ResponseWriter, request *http.Request) {
	selectedChapterPtr, code, err := h.getChapter(request)
	if err != nil {
		http.Error(responseWriter, err.Error(), code)
		return
	}

	data := dto.ChapterData{
		HomeData: dto.HomeData{
			CurriculumID: selectedChapterPtr.CurriculumID,
			GradeID:      selectedChapterPtr.GradeID,
			SubjectID:    selectedChapterPtr.SubjectID,
		},
		ChapterPtr: selectedChapterPtr,
	}
	views.ExecuteTemplates(responseWriter, data, template.FuncMap{
		"getName": getChapterName,
	}, baseTemplate, editChapterTemplate)
}

func (h *ChaptersHandler) UpdateChapter(responseWriter http.ResponseWriter, request *http.Request) {
	chapterIDStr := request.FormValue("id")
	chapterID, err := utils.StringToIntType[int16](chapterIDStr)
	if err != nil {
		http.Error(responseWriter, "Invalid Chapter ID", http.StatusBadRequest)
		return
	}

	chapterName := request.FormValue("name")
	chapterCode := request.FormValue("code")

	dummyChapterPtr := &models.Chapter{}
	chapterMap := dummyChapterPtr.BuildMap(chapterCode, chapterName)

	_, err = h.chaptersService.UpdateObject(chapterIDStr, chaptersEndPoint, chapterMap, chaptersKey,
		func(chapter *models.Chapter) bool {
			return (*chapter).ID == chapterID
		})
	if err != nil {
		http.Error(responseWriter, fmt.Sprintf("Error updating chapter: %v", err), http.StatusInternalServerError)
		return
	}

	views.ExecuteTemplate(updateSuccessTemplate, responseWriter, "Chapter", nil)
}

func (h *ChaptersHandler) AddChapter(responseWriter http.ResponseWriter, request *http.Request) {
	chapterCode := request.FormValue("code")
	chapterName := request.FormValue("name")
	curriculumIDStr := request.FormValue(CurriculumDropdownName)
	curriculumID, err := utils.StringToIntType[int16](curriculumIDStr)
	if err != nil {
		http.Error(responseWriter, "Invalid Curriculum ID", http.StatusBadRequest)
		return
	}
	gradeIDStr := request.FormValue(GradeDropdownName)
	gradeID, err := utils.StringToIntType[int8](gradeIDStr)
	if err != nil {
		http.Error(responseWriter, "Invalid Grade ID", http.StatusBadRequest)
		return
	}
	subjectIDStr := request.FormValue(SubjectDropdownName)
	subjectID, err := utils.StringToIntType[int8](subjectIDStr)
	if err != nil {
		http.Error(responseWriter, "Invalid Subject ID", http.StatusBadRequest)
		return
	}
	newChapterPtr := models.NewChapter(chapterCode, chapterName, curriculumID, gradeID, subjectID)

	newChapterPtr, err = h.chaptersService.AddObject(newChapterPtr, chaptersKey, chaptersEndPoint)
	if err != nil {
		http.Error(responseWriter, fmt.Sprintf("Error adding chapter: %v", err), http.StatusInternalServerError)
		return
	}

	chapterPtrs := []*models.Chapter{newChapterPtr}
	views.ExecuteTemplate(chapterRowTemplate, responseWriter, chapterPtrs, template.FuncMap{
		"getName": getChapterName,
	})
}

func (h *ChaptersHandler) ArchiveChapter(responseWriter http.ResponseWriter, request *http.Request) {
	chapterIDStr := request.URL.Query().Get("id")
	chapterID, err := utils.StringToIntType[int16](chapterIDStr)
	if err != nil {
		http.Error(responseWriter, "Invalid Chapter ID", http.StatusBadRequest)
		return
	}

	chapterMap := map[string]any{
		"cms_status_id": constants.StatusArchived,
	}

	err = h.chaptersService.ArchiveObject(chapterIDStr, chaptersEndPoint, chapterMap, chaptersKey,
		func(chapter *models.Chapter) bool {
			return (*chapter).ID != chapterID
		})

	// If http error is thrown from here then target row won't be removed by htmx code
	if err != nil {
		http.Error(responseWriter, err.Error(), http.StatusInternalServerError)
	}
}

func sortChapters(chapterPtrs []*models.Chapter, sortColumn string, sortOrder string) {
	slices.SortStableFunc(chapterPtrs, func(c1, c2 *models.Chapter) int {
		var sortResult int
		switch sortColumn {
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
			sortResult = strings.Compare(c1.GetNameByLang("en"), c2.GetNameByLang("en"))
		case "3":
			sortResult = int(c1.TopicCount() - c2.TopicCount())
		default:
			sortResult = 0
		}

		if constants.SortOrder(sortOrder) == constants.SortOrderDesc {
			sortResult = -sortResult
		}
		return sortResult
	})
}

func (h *ChaptersHandler) getChapter(request *http.Request) (*models.Chapter, int, error) {
	urlVals := request.URL.Query()
	chapterIDStr := urlVals.Get("id")

	if chapterIDStr == "" {
		chapterIDStr = urlVals.Get("chapter-dropdown")
	}
	chapterID, err := utils.StringToIntType[int16](chapterIDStr)
	if err != nil {
		return nil, http.StatusBadRequest, fmt.Errorf("invalid Chapter ID: %w", err)
	}

	selectedChapterPtr, err := h.chaptersService.GetObject(chapterIDStr,
		func(chapter *models.Chapter) bool {
			return (*chapter).ID == chapterID
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

	curriculumID, gradeID, subjectID := getCurriculumGradeSubjectIDs(request.URL.Query())
	if curriculumID == 0 || gradeID == 0 || subjectID == 0 {
		return
	}

	data := dto.ChapterData{
		HomeData: dto.HomeData{
			CurriculumID: curriculumID,
			GradeID:      gradeID,
			SubjectID:    subjectID,
		},
		ChapterPtr: selectedChapterPtr,
	}
	views.ExecuteTemplates(responseWriter, data, template.FuncMap{
		"getName": getChapterName,
	}, baseTemplate, chapterTemplate)
}

func (h *ChaptersHandler) LoadTopics(responseWriter http.ResponseWriter, request *http.Request) {
	chapterIDStr := request.URL.Query().Get("id")
	data := dto.TopicsData{
		ChapterID: chapterIDStr,
	}
	views.ExecuteTemplate(topicsTemplate, responseWriter, data, nil)
}

func (h *ChaptersHandler) LoadResources(responseWriter http.ResponseWriter, request *http.Request) {
	chapterIDStr := request.URL.Query().Get("chapterId")
	data := dto.ResourcesData{
		ChapterID: chapterIDStr,
	}
	views.ExecuteTemplate(resourcesTemplate, responseWriter, data, nil)
}

func (h *ChaptersHandler) GetTopics(responseWriter http.ResponseWriter, request *http.Request) {
	urlVals := request.URL.Query()
	view := urlVals.Get("view")
	var filename string
	switch view {
	case "list":
		filename = topicRowTemplate
	case "dropdown-optional":
		filename = topicDropdownOptionalTemplate
	default:
		filename = topicDropdownTemplate
	}

	selectedChapterPtr, code, err := h.getChapter(request)
	if err != nil {
		chapterDropdownVal := urlVals.Get("chapter-dropdown")
		/**
		 * if "Select Chapter" default option is selected or its value is blank (on coming back from
		 * single problem to add test screen sometimes it is blank) in add test screen,
		 * then just return empty response.
		 */
		if chapterDropdownVal == "Select Chapter" || chapterDropdownVal == "" {
			views.ExecuteTemplate(filename, responseWriter, nil, template.FuncMap{
				"getName": getTopicName,
			})
		} else {
			http.Error(responseWriter, err.Error(), code)
		}
		return
	}
	// Use a local copy so we never mutate the cached chapter pointer.
	localChapter := *selectedChapterPtr
	localChapter.Topics = nil
	curriculumID, _, _ := getCurriculumGradeSubjectIDs(urlVals)
	if curriculumID != 0 {
		localChapter.CurriculumID = curriculumID
	}
	h.getTopics(responseWriter, []*models.Chapter{&localChapter})

	sortColumn := urlVals.Get("sortColumn")
	sortOrder := urlVals.Get("sortOrder")
	sortTopics(localChapter.Topics, sortColumn, sortOrder)

	views.ExecuteTemplate(filename, responseWriter, localChapter.Topics, template.FuncMap{
		"getName": getTopicName,
	})
}
