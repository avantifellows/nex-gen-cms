package models

type Subject struct {
	ID         int8          `json:"id"`
	Name       []SubjectLang `json:"name"`
	Code       string        `json:"code"`
	ParentID   int8          `json:"parent_id"`
	ParentName []SubjectLang
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

func (s *Subject) GetParentNameByLang(langCode string) string {
	for _, subLang := range s.ParentName {
		if subLang.LangCode == langCode {
			return subLang.SubName
		}
	}
	return ""
}
