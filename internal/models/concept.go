package models

type Concept struct {
	ID      int16         `json:"id"`
	Name    []ConceptLang `json:"name"`
	TopicID int16         `json:"topic_id"`
}

type ConceptLang struct {
	LangCode    string `json:"lang_code"`
	ConceptName string `json:"concept"`
}

func (c *Concept) GetNameByLang(langCode string) string {
	for _, conceptLang := range c.Name {
		if conceptLang.LangCode == langCode {
			return conceptLang.ConceptName
		}
	}
	return ""
}
