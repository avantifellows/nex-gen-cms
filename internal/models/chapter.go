package models

type Chapter struct {
	ID           int16  `json:"id"`
	Code         string `json:"code"`
	Name         string `json:"name"`
	CurriculumID int16  `json:"curriculum_id"`
	GradeId      int8   `json:"grade_id"`
	SubjectID    int8   `json:"subject_id"`
	Topics       []Topic
}

func NewChapter(code string, name string, curriculum_id int16, grade_id int8, subject_id int8) *Chapter {
	return &Chapter{
		Code:         code,
		Name:         name,
		CurriculumID: curriculum_id,
		GradeId:      grade_id,
		SubjectID:    subject_id,
	}
}

func (chapter Chapter) TopicCount() int8 {
	return int8(len(chapter.Topics))
}

func (chapterPtr *Chapter) UpdateProperties(code string, name string) {
	chapterPtr.Name = name
	chapterPtr.Code = code
}

func (chapterPtr *Chapter) BuildMap(code string, name string) map[string]any {
	return map[string]any{
		"code": code,
		"name": name,
	}
}
