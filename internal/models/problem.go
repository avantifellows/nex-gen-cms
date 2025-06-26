package models

import "html/template"

type Problem struct {
	ID         int            `json:"id,omitempty"`
	Code       string         `json:"code,omitempty"`
	Type       string         `json:"type"`
	Subtype    string         `json:"subtype"`
	TypeParams ProbTypeParams `json:"type_params"`
	MetaData   ProbMetaData   `json:"meta_data"`
	SkillIDs   []int16        `json:"skill_ids"`
	// As we don’t want it in the JSON output while marshaling (Problem struct in Go --> JSON conversion in
	// api_repository on executing json.Marshal(body)), explicitly tell the JSON encoder to skip it using `json:"-"`
	Skills          []Skill  `json:"-"`
	CurriculumID    int16    `json:"curriculum_id"`
	GradeID         int8     `json:"grade_id"`
	SubjectID       int8     `json:"subject_id"`
	Subject         Subject  `json:"-"`
	TopicId         int16    `json:"topic_id"`
	ChapterId       int16    `json:"chapter_id"`
	ConceptIds      []int16  `json:"concept_ids"`
	DifficultyLevel string   `json:"difficulty_level"`
	Tags            []string `json:"tags"`
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
