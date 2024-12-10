package models

type Topic struct {
	ID           int16  `json:"id"`
	Name         string `json:"name"`
	Code         string `json:"code"`
	ChapterID    int16  `json:"chapter_id"`
	CurriculumID int16  `json:"curriculum_id"`
}

func NewTopic(code string, name string, chapter_id int16, curriculum_id int16) *Topic {
	return &Topic{
		Code:         code,
		Name:         name,
		ChapterID:    chapter_id,
		CurriculumID: curriculum_id,
	}
}

func (topicPtr *Topic) BuildMap(code string, name string) map[string]any {
	return map[string]any{
		"code": code,
		"name": name,
	}
}
