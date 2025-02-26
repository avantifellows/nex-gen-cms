package dto

import (
	"github.com/avantifellows/nex-gen-cms/internal/constants"
	"github.com/avantifellows/nex-gen-cms/internal/models"
)

type HomeChapterData struct {
	InitialLoad bool
	ChapterPtr  *models.Chapter
}

type SortState struct {
	Column string
	Order  constants.SortOrder
}

type TopicsData struct {
	ChapterId       string
	TopicsSortState SortState
}
