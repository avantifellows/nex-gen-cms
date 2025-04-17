package models

import "html/template"

type Problem struct {
	ID              int `json:"id"`
	Code            string
	LangCode        string       `json:"lang_code"`
	MetaData        ProbMetaData `json:"meta_data"`
	SkillIDs        []int16      `json:"skill_ids"`
	Skills          []Skill
	Subtype         string `json:"subtype"`
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

func (p Problem) DisplaySubtype() string {
	switch p.Subtype {
	case "mcq_single_answer":
		return "MCQ Single Answer"
	case "numerical_answer":
		return "Numerical Answer"
	default:
		return "Unknown"
	}
}

func (p Problem) DisplayDifficulty() int8 {
	switch p.DifficultyLevel {
	case "easy":
		return 1
	case "medium":
		return 2
	default:
		return 3
	}
}
