package models

import (
	"encoding/json"
	"html/template"
)

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

// UnmarshalJSON migrates old subject_ids array format to new subject_id scalar format.
// Remove once all DB records are migrated to the new format.
func (c *Config) UnmarshalJSON(data []byte) error {
	type Alias Config
	var raw struct {
		Alias
		Subjects []struct {
			SubjectIDs   []int8        `json:"subject_ids"`
			SubjectID    int8          `json:"subject_id"`
			Rules        RuleDetails   `json:"rules"`
			Instructions template.HTML `json:"instructions,omitempty"`
		} `json:"subjects"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	*c = Config(raw.Alias)

	isOldFormat := false
	var subjects []SubjectRule
	for _, s := range raw.Subjects {
		if len(s.SubjectIDs) > 0 {
			isOldFormat = true
			for _, id := range s.SubjectIDs {
				subjects = append(subjects, SubjectRule{SubjectID: id, Rules: s.Rules, Instructions: s.Instructions})
			}
		} else {
			subjects = append(subjects, SubjectRule{SubjectID: s.SubjectID, Rules: s.Rules, Instructions: s.Instructions})
		}
	}
	c.Subjects = subjects

	// In the old format, a single subjects[] entry implicitly meant single-subject
	if isOldFormat && len(raw.Subjects) == 1 {
		c.SingleSubject = true
	}
	return nil
}
