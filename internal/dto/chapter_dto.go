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
