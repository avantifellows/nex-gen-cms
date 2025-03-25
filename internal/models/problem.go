package models

import "html/template"

type Problem struct {
	ID       int `json:"id"`
	Code     string
	LangID   int8         `json:"lang_id"`
	MetaData ProbMetaData `json:"meta_data"`
}

type ProbMetaData struct {
	Question  template.HTML `json:"text"`
	Options   []string      `json:"options"`
	Answers   []string      `json:"answer"`
	Solutions []Solution    `json:"solutions"`
}

type Solution struct {
	Type  string `json:"type"`
	Value any    `json:"value"`
}
