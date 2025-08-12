package models

type Topic struct {
	ID           int16       `json:"id"`
	Name         []TopicLang `json:"name"`
	Code         string      `json:"code"`
	ChapterID    int16       `json:"chapter_id"`
	CurriculumID int16       `json:"curriculum_id"`
}

type TopicLang struct {
	LangCode  string `json:"lang_code"`
	TopicName string `json:"topic"`
}

func NewTopic(code string, name string, chapter_id int16, curriculum_id int16) *Topic {
	return &Topic{
		Code:         code,
		Name:         []TopicLang{{LangCode: "en", TopicName: name}},
		ChapterID:    chapter_id,
		CurriculumID: curriculum_id,
	}
}

func (topicPtr *Topic) BuildMap(code string, name string) map[string]any {
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
