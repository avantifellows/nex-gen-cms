package models

type Subject struct {
	ID   int8          `json:"id"`
	Name []SubjectLang `json:"name"`
	Code string        `json:"code"`
}

type SubjectLang struct {
	LangCode string `json:"lang_code"`
	SubName  string `json:"subject"`
}

func (s *Subject) GetNameByLang(langCode string) string {
	for _, subLang := range s.Name {
		if subLang.LangCode == langCode {
			return subLang.SubName
		}
	}
	return ""
}
