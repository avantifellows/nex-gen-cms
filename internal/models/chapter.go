package models

type Chapter struct {
	ID           int16         `json:"id"`
	Code         string        `json:"code"`
	Name         []ChapterLang `json:"name"`
	CurriculumID int16         `json:"curriculum_id"`
	GradeID      int8          `json:"grade_id"`
	SubjectID    int8          `json:"subject_id"`
	/**
	 * []*Topic is used instead of []Topic so that updates applied in centrally cached Topic objects
	 * are also visible inside these Topic objects
	 */
	Topics []*Topic
}

type ChapterLang struct {
	ChapterName string `json:"chapter"`
	LangCode    string `json:"lang_code"`
}

func NewChapter(code string, name string, curriculum_id int16, grade_id int8, subject_id int8) *Chapter {
	return &Chapter{
		Code:         code,
		Name:         []ChapterLang{{ChapterName: name, LangCode: "en"}},
		CurriculumID: curriculum_id,
		GradeID:      grade_id,
		SubjectID:    subject_id,
	}
}

func (chapter Chapter) TopicCount() int8 {
	return int8(len(chapter.Topics))
}

func (chapterPtr *Chapter) BuildMap(code string, name string) map[string]any {
	return map[string]any{
		"code": code,
		"name": []ChapterLang{{ChapterName: name, LangCode: "en"}},
	}
}

func (ch *Chapter) GetNameByLang(langCode string) string {
	for _, chapterLang := range ch.Name {
		if chapterLang.LangCode == langCode {
			return chapterLang.ChapterName
		}
	}
	return ""
}
