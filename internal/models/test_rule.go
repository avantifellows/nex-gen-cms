package models

import "html/template"

type TestRule struct {
	ExamID   int8   `json:"exam_id"`
	TestType string `json:"test_type"`
	Config   Config `json:"config"`
}

type Config struct {
	Subjects      []SubjectRule `json:"subjects"`
	Duration      int16         `json:"duration"`
	MarkingScheme MarkingScheme `json:"marking_scheme"`
	Instructions  template.HTML `json:"instructions"`
	SingleSubject bool          `json:"single_subject,omitempty"`
}

type SubjectRule struct {
	SubjectID    int8          `json:"subject_id"`
	Rules        RuleDetails   `json:"rules"`
	Instructions template.HTML `json:"instructions,omitempty"`
}

type RuleDetails struct {
	Marks      int16      `json:"marks"`
	Questions  int16      `json:"questions"`
	Sections   []Section  `json:"sections"`
	Difficulty Difficulty `json:"difficulty"`
}

type Section struct {
	Name  string `json:"name"`
	Type  string `json:"type"`
	Count int    `json:"count"`
}

type Difficulty struct {
	Easy   int `json:"easy"`
	Medium int `json:"medium"`
	Hard   int `json:"hard"`
}

type MarkingScheme struct {
	PosMarks []int `json:"pos_marks"`
	NegMarks []int `json:"neg_marks"`
}
