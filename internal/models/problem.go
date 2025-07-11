package models

import "html/template"

type Problem struct {
	ID              int            `json:"id,omitempty"`
	Code            string         `json:"code,omitempty"`
	Type            string         `json:"type"`
	Subtype         string         `json:"subtype"`
	TypeParams      ProbTypeParams `json:"type_params"`
	MetaData        ProbMetaData   `json:"meta_data"`
	SkillIDs        []int16        `json:"skill_ids"`
	Skills          []Skill
	CurriculumID    int16 `json:"curriculum_id"` // used with only get call
	GradeID         int8  `json:"grade_id"`      // used with only get call
	SubjectID       int8  `json:"subject_id"`
	Subject         Subject
	TopicID         int16     `json:"topic_id"`
	ChapterID       int16     `json:"chapter_id"`
	Concepts        []Concept `json:"concepts"` // used with only get call
	DifficultyLevel string    `json:"difficulty_level"`
	TagIDs          []int     `json:"tag_ids"`
	TagNames        []string
	Status          string `json:"cms_status"`
}

type ProbTypeParams struct {
	TestIds []int `json:"test_ids"`
}

type ProbMetaData struct {
	Question  template.HTML   `json:"text"`
	Options   []template.HTML `json:"options"`
	Answers   []string        `json:"answer"`
	Solutions []Solution      `json:"solutions"`
}

type Solution struct {
	Type  string        `json:"type"`
	Value template.HTML `json:"value"`
}

func (p Problem) DisplayDifficulty() int8 {
	switch p.DifficultyLevel {
	case "hard":
		return 3
	case "medium":
		return 2
	default:
		return 1
	}
}
