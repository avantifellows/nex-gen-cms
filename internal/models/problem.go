package models

import "html/template"

type Problem struct {
	ID              int          `json:"id"`
	Code            string       `json:"code"`
	MetaData        ProbMetaData `json:"meta_data"`
	SkillIDs        []int16      `json:"skill_ids"`
	Skills          []Skill
	Subtype         string `json:"subtype"`
	CurriculumID    int16  `json:"curriculum_id"`
	GradeID         int8   `json:"grade_id"`
	SubjectID       int8   `json:"subject_id"`
	Subject         Subject
	DifficultyLevel string `json:"difficulty_level"`
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
