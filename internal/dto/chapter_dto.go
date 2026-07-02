package dto

import "github.com/avantifellows/nex-gen-cms/internal/models"

type ChapterData struct {
	HomeData
	ChapterPtr *models.Chapter
}

type ResourcesData struct {
	ChapterId string
	TopicId   string
}

// ChapterTestsData feeds the chapter view's Tests sub-tab shell; the tbody uses these to
// fetch the chapter's tests via /api/chapter-tests.
type ChapterTestsData struct {
	ChapterId    string
	CurriculumId string
	GradeId      string
}
