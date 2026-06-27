package models

type Topic struct {
	ID           int16             `json:"id"`
	Name         []TopicLang       `json:"name"`
	Code         string            `json:"code"`
	ChapterID    int16             `json:"chapter_id"`
	CurriculumID int16             `json:"curriculum_id,omitempty"` // create API only
	Curriculums  []TopicCurriculum `json:"curriculums,omitempty"`   // get API only
	StatusID     int8              `json:"cms_status_id,omitempty"`
}

type TopicCurriculum struct {
	Priority     *int16  `json:"priority"`
	CurriculumID int16   `json:"curriculum_id"`
	PriorityText *string `json:"priority_text"`
}

type TopicLang struct {
	LangCode  string `json:"lang_code"`
	TopicName string `json:"topic"`
}

func NewTopic(code string, name string, chapterID int16, curriculumID int16) *Topic {
	return &Topic{
		Code:         code,
		Name:         []TopicLang{{LangCode: "en", TopicName: name}},
		ChapterID:    chapterID,
		CurriculumID: curriculumID,
	}
}

// NormalizeCurriculums fills Curriculums from CurriculumID when the create API
// response omits the curriculums array (GET responses include curriculums).
func (t *Topic) NormalizeCurriculums() {
	if len(t.Curriculums) > 0 {
		return
	}
	if t.CurriculumID != 0 {
		t.Curriculums = []TopicCurriculum{{CurriculumID: t.CurriculumID}}
	}
}

func (t *Topic) HasCurriculumID(curriculumID int16) bool {
	t.NormalizeCurriculums()
	for _, curriculum := range t.Curriculums {
		if curriculum.CurriculumID == curriculumID {
			return true
		}
	}
	return false
}

func (t *Topic) BuildMap(code string, name string) map[string]any {
	return map[string]any{
		"code": code,
		"name": []TopicLang{{TopicName: name, LangCode: "en"}},
	}
}

func (t *Topic) GetNameByLang(langCode string) string {
	for _, topicLang := range t.Name {
		if topicLang.LangCode == langCode {
			return topicLang.TopicName
		}
	}
	return ""
}
