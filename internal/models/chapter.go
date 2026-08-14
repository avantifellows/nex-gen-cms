package models

type Chapter struct {
	ID           int16         `json:"id"`
	Code         string        `json:"code"`
	Name         []ChapterLang `json:"name"`
	CurriculumID int16         `json:"curriculum_id"`
	GradeID      int8          `json:"grade_id,omitempty"`
	SubjectID    int8          `json:"subject_id"`
	StatusID     int8          `json:"cms_status_id,omitempty"`
	Priority     int8          `json:"priority,omitempty"`
	PriorityText string        `json:"priority_text,omitempty"`
	/**
	 * []*Topic is used instead of []Topic so that updates applied in centrally cached Topic objects
	 * are also visible inside these Topic objects
	 */
	Topics []*Topic
}

// priorityByText maps the CMS's priority dropdown value to the numeric priority sent to db-service.
// 1 is the highest priority (P1-style ranking), matching db-service's expected ordering.
var priorityByText = map[string]int8{
	"high":   1,
	"medium": 2,
	"low":    3,
}

func PriorityFromText(priorityText string) int8 {
	return priorityByText[priorityText]
}

type ChapterLang struct {
	ChapterName string `json:"chapter"`
	LangCode    string `json:"lang_code"`
}

func NewChapter(code string, name string, curriculum_id int16, grade_id int8, subject_id int8, priorityText string) *Chapter {
	return &Chapter{
		Code:         code,
		Name:         []ChapterLang{{ChapterName: name, LangCode: "en"}},
		CurriculumID: curriculum_id,
		GradeID:      grade_id,
		SubjectID:    subject_id,
		Priority:     PriorityFromText(priorityText),
		PriorityText: priorityText,
	}
}

func (chapter Chapter) TopicCount() int8 {
	return int8(len(chapter.Topics))
}

func (chapterPtr *Chapter) BuildMap(code string, name string, priorityText string) map[string]any {
	return map[string]any{
		"code":          code,
		"name":          []ChapterLang{{ChapterName: name, LangCode: "en"}},
		"priority":      PriorityFromText(priorityText),
		"priority_text": priorityText,
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
