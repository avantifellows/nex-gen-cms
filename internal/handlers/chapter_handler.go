package handlers

import (
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strings"

	"github.com/avantifellows/nex-gen-cms/internal/constants"
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

type ChaptersHandler struct {
	chaptersService *services.ChapterService
	topicsService   *services.Service[models.Topic]
}

// NewChaptersHandler creates a new instance of ChaptersHandler
func NewChaptersHandler(chaptersService *services.ChapterService,
	topicsService *services.Service[models.Topic]) *ChaptersHandler {
	return &ChaptersHandler{
		chaptersService: chaptersService,
		topicsService:   topicsService,
	}
}

type HomeChapterData struct {
	InitialLoad bool
	ChapterPtr  *models.Chapter
}

type SortState struct {
	Column string
	Order  constants.SortOrder
}

var sortState = SortState{
	Column: "0",
	Order:  constants.SortOrderAsc,
}

func (h *ChaptersHandler) LoadChapters(w http.ResponseWriter, r *http.Request) {
	sortColumn := r.URL.Query().Get("sortColumn")

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
	local_repo.ExecuteTemplate(chaptersTemplate, w, sortState)
}

func (h *ChaptersHandler) GetChapters(w http.ResponseWriter, r *http.Request) {
	curriculumId, gradeId, subjectId := getCurriculumGradeSubjectIds(r.URL.Query())
	if curriculumId == 0 || gradeId == 0 || subjectId == 0 {
		return
	}

	chapters, err := h.chaptersService.GetList(chaptersEndPoint, chaptersKey, false)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error fetching chapters: %v", err), http.StatusInternalServerError)
		return
	}

	filteredChapters := funk.Filter(*chapters, func(chapter *models.Chapter) bool {
		return (*chapter).CurriculumID == curriculumId && (*chapter).GradeId == gradeId &&
			(*chapter).SubjectID == subjectId
	})
	typecastedChapters := filteredChapters.([]*models.Chapter)

	topics, err := h.topicsService.GetList(getTopicsEndPoint, topicsKey, false)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error fetching topics: %v", err), http.StatusInternalServerError)
	} else {
		associateTopicsWithChapters(typecastedChapters, *topics)
	}
	sortChapters(typecastedChapters)

	local_repo.ExecuteTemplate(chapterRowTemplate, w, typecastedChapters)
}

func (h *ChaptersHandler) EditChapter(w http.ResponseWriter, r *http.Request) {
	// Check if the request is from HTMX (using the HX-Request header)
	if r.Header.Get("HX-Request") == "" {
		// If the request is NOT from HTMX, redirect to the home page
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	chapterIdStr := r.URL.Query().Get("id")
	chapterId, err := utils.StringToIntType[int16](chapterIdStr)
	if err != nil {
		http.Error(w, "Invalid Chapter ID", http.StatusBadRequest)
		return
	}

	selectedChapterPtr, err := h.chaptersService.GetObject(chapterIdStr,
		func(chapter *models.Chapter) bool {
			return (*chapter).ID == chapterId
		}, chaptersKey, chaptersEndPoint)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error fetching chapter: %v", err), http.StatusInternalServerError)
		return
	}

	data := HomeChapterData{
		false,
		selectedChapterPtr,
	}
	local_repo.ExecuteTemplates(baseTemplate, editChapterTemplate, w, data)
}

func (h *ChaptersHandler) UpdateChapter(w http.ResponseWriter, r *http.Request) {
	chapterIdStr := r.FormValue("id")
	chapterId, err := utils.StringToIntType[int16](chapterIdStr)
	if err != nil {
		http.Error(w, "Invalid Chapter ID", http.StatusBadRequest)
		return
	}

	chapterName := r.FormValue("name")
	chapterCode := r.FormValue("code")

	_, err = h.chaptersService.UpdateChapter(chapterIdStr, chapterCode, chapterName, chaptersKey,
		func(chapter *models.Chapter) bool {
			return (*chapter).ID == chapterId
		}, chaptersEndPoint)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error updating chapter: %v", err), http.StatusInternalServerError)
		return
	}

	local_repo.ExecuteTemplate(updateSuccessTemplate, w, nil)
}

func (h *ChaptersHandler) AddChapter(w http.ResponseWriter, r *http.Request) {
	chapterCode := r.FormValue("code")
	chapterName := r.FormValue("name")
	curriculumIdStr := r.FormValue(CURRICULUM_DROPDOWN_NAME)
	curriculumId, err := utils.StringToIntType[int16](curriculumIdStr)
	if err != nil {
		http.Error(w, "Invalid Curriculum ID", http.StatusBadRequest)
		return
	}
	gradeIdStr := r.FormValue(GRADE_DROPDOWN_NAME)
	gradeId, err := utils.StringToIntType[int8](gradeIdStr)
	if err != nil {
		http.Error(w, "Invalid Grade ID", http.StatusBadRequest)
		return
	}
	subjectIdStr := r.FormValue(SUBJECT_DROPDOWN_NAME)
	subjectId, err := utils.StringToIntType[int8](subjectIdStr)
	if err != nil {
		http.Error(w, "Invalid Subject ID", http.StatusBadRequest)
		return
	}
	newChapterPtr := models.NewChapter(chapterCode, chapterName, curriculumId, gradeId, subjectId)

	newChapterPtr, err = h.chaptersService.AddObject(newChapterPtr, chaptersKey, chaptersEndPoint)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error adding chapter: %v", err), http.StatusInternalServerError)
		return
	}

	chapterPtrs := []*models.Chapter{newChapterPtr}
	local_repo.ExecuteTemplate(chapterRowTemplate, w, chapterPtrs)
}

func (h *ChaptersHandler) DeleteChapter(w http.ResponseWriter, r *http.Request) {
	chapterIdStr := r.URL.Query().Get("id")
	chapterId, err := utils.StringToIntType[int16](chapterIdStr)
	if err != nil {
		http.Error(w, "Invalid Chapter ID", http.StatusBadRequest)
		return
	}
	err = h.chaptersService.DeleteObject(chapterIdStr, func(c *models.Chapter) bool {
		return c.ID != chapterId
	}, chaptersKey, chaptersEndPoint)

	// If http error is thrown from here then target row won't be removed by htmx code
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
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
		switch sortState.Column {
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

		if sortState.Order == constants.SortOrderDesc {
			sortResult = -sortResult
		}
		return sortResult
	})
}
