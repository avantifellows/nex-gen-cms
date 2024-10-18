package handlers

import (
	"fmt"
	"net/http"
	"net/url"

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
const chaptersTemplate = "chapter_row.html"
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

	local_repo.ExecuteTemplate(chaptersTemplate, w, typecastedChapters)
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
	local_repo.ExecuteTemplate(chaptersTemplate, w, chapterPtrs)
}

func (h *ChaptersHandler) DeleteChapter(w http.ResponseWriter, r *http.Request) {
	chapterIdStr := r.URL.Query().Get("id")
	chapterId, err := utils.StringToIntType[int16](chapterIdStr)
	if err != nil {
		http.Error(w, "Invalid Chapter ID", http.StatusBadRequest)
		return
	}
	h.chaptersService.DeleteObject(chapterIdStr, func(c *models.Chapter) bool {
		return c.ID != chapterId
	}, chaptersKey, chaptersEndPoint)
}

func associateTopicsWithChapters(chapterPtrs []*models.Chapter, topicPtrs []*models.Topic) {
	// Create a map to quickly lookup chapters by their ID
	chapterPtrsMap := make(map[int16]*models.Chapter)

	// Fill the map with the address of each chapter
	for _, chapterPtr := range chapterPtrs {
		chapterPtrsMap[chapterPtr.ID] = chapterPtr
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
