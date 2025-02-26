package models

type Test struct {
	ID           int16  `json:"id"`
	Name         string `json:"name"`
	Code         string `json:"code"`
	ChapterID    int16  `json:"chapter_id"`
	CurriculumID int16  `json:"curriculum_id"`
}
