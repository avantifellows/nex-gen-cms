package models

type Topic struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	Code      string `json:"code"`
	ChapterID int16  `json:"chapter_id"`
}
