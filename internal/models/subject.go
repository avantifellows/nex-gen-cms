package models

import "strings"

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

// MatchSubjectsByName returns the subjects whose English name is in names (case-insensitive), in
// the order of names. A name with no matching subject is skipped.
func MatchSubjectsByName(subjects []*Subject, names []string) []*Subject {
	matched := make([]*Subject, 0, len(names))
	for _, name := range names {
		for _, subject := range subjects {
			if strings.EqualFold(subject.GetNameByLang("en"), name) {
				matched = append(matched, subject)
				break
			}
		}
	}
	return matched
}
