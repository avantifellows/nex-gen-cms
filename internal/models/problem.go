package models

import "html/template"

type Problem struct {
	ID       int `json:"id"`
	Code     string
	LangCode string       `json:"lang_code"`
	MetaData ProbMetaData `json:"meta_data"`
}

type ProbMetaData struct {
	Question  template.HTML   `json:"text"`
	Options   []template.HTML `json:"options"`
	Answers   []string        `json:"answer"`
	Solutions []Solution      `json:"solutions"`
}

type Solution struct {
	Type  string `json:"type"`
	Value any    `json:"value"`
}
