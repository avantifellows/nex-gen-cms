package models

import "html/template"

type Test struct {
	ID               int               `json:"id,omitempty"`
	Name             []ResName         `json:"name"`
	Code             string            `json:"code"`
	Type             string            `json:"type"`
	Subtype          string            `json:"subtype"`
	ExamIDs          []int8            `json:"exam_ids"`
	SkillIDs         []int16           `json:"skill_ids,omitempty"`
	CurriculumGrades []CurriculumGrade `json:"curriculum_grades,omitempty"`
	TypeParams       ResTypeParams     `json:"type_params"`
	StatusID         int8              `json:"cms_status_id,omitempty"`
}

type CurriculumGrade struct {
	CurriculumID int16 `json:"curriculum_id"`
	GradeID      int8  `json:"grade_id,omitempty"`
}

type ResName struct {
	LangCode string `json:"lang_code"`
	Resource string `json:"resource"`
}

type ResTypeParams struct {
	Duration string `json:"duration"`
	Marks    int16  `json:"marks"`
	PosMarks []int8 `json:"pos_marks,omitempty"`
	NegMarks []int8 `json:"neg_marks,omitempty"`
	// ChapterID links a chapter_test to its chapter (only meaningful for chapter_test;
	// stored in type_params rather than a shared resource column, and consumed by the
	// service list route so af_lms can filter "tests for a chapter"). Nullable: orphan
	// tests whose problems don't resolve to a single chapter leave it unset.
	ChapterID    *int16        `json:"chapter_id,omitempty"`
	Subjects     []ResSubject  `json:"subjects,omitempty"`
	Instructions template.HTML `json:"instructions,omitempty"`
	// InstructionLangVersions is the source of truth going forward, and includes an "en"
	// entry alongside any regional ones. Instructions above is kept in sync for now so
	// older consumers still work; once everything reads from the array it can be dropped.
	InstructionLangVersions []InstructionLangVersion `json:"instruction_lang_versions,omitempty"`
}

// InstructionLangVersion holds one language's instructions text, including English.
type InstructionLangVersion struct {
	LangCode     string        `json:"lang_code"`
	Instructions template.HTML `json:"instructions"`
}

// FindInstructionLangVersion returns the version matching langCode, or nil if absent.
func FindInstructionLangVersion(versions []InstructionLangVersion, langCode string) *InstructionLangVersion {
	for i := range versions {
		if versions[i].LangCode == langCode {
			return &versions[i]
		}
	}
	return nil
}

// ResolveInstructions looks up langCode in versions; if not found, falls back to the
// legacy singular field for "en" only (covers rows not yet migrated to the array).
func ResolveInstructions(legacy template.HTML, versions []InstructionLangVersion, langCode string) template.HTML {
	if v := FindInstructionLangVersion(versions, langCode); v != nil {
		return v.Instructions
	}
	if langCode == "en" {
		return legacy
	}
	return ""
}

type ResSubject struct {
	SubjectID int8         `json:"subject_id"`
	Name      string       `json:",omitempty"`
	Marks     int          `json:"marks"`
	PosMarks  []int8       `json:"pos_marks,omitempty"`
	NegMarks  []int8       `json:"neg_marks,omitempty"`
	Sections  []ResSection `json:"sections"`
}

type ResSection struct {
	Type       string        `json:"type"` // system identifier
	Name       string        `json:"name"` // display name (customizable)
	Marks      int16         `json:"marks"`
	PosMarks   []int8        `json:"pos_marks,omitempty"`
	NegMarks   []int8        `json:"neg_marks,omitempty"`
	Compulsory ResCompulsory `json:"compulsory"`
	Optional   *ResOptional  `json:"optional,omitempty"`
}

type ResCompulsory struct {
	Problems []ResProblem `json:"problems"`
}

type ResOptional struct {
	MandatoryCount int8         `json:"mandatory_count,omitempty"`
	Problems       []ResProblem `json:"problems,omitempty"`
}

type ResProblem struct {
	ID              int    `json:"id"`
	PosMarks        []int8 `json:"pos_marks"`
	NegMarks        []int8 `json:"neg_marks,omitempty"`
	DifficultyLevel string `json:"difficulty_level"`
	// struct is never empty and omitempty is ignored without pointer,
	// so we need to use a pointer to make it optional
	OptionLayout *OptionLayout `json:"option_layout,omitempty"`
}

type OptionLayout struct {
	Rows int8 `json:"rows"`
	Cols int8 `json:"cols"`
}

// Method to count total problems
func (t Test) ProblemCount() int {
	total := 0

	// Iterate over subjects
	for _, subject := range t.TypeParams.Subjects {
		// Iterate over sections
		for _, section := range subject.Sections {
			// Count compulsory problems
			total += len(section.Compulsory.Problems)
			// Count optional problems
			optionalSection := section.Optional
			if optionalSection != nil {
				total += int(optionalSection.MandatoryCount)
			}
		}
	}

	return total
}

func (test *Test) GetNameByLang(langCode string) string {
	for _, testLang := range test.Name {
		if testLang.LangCode == langCode {
			return testLang.Resource
		}
	}
	return ""
}

func (t *Test) DisplaySubtype() string {
	switch t.Subtype {
	case "chapter_test":
		return "Chapter Test"
	case "part_test":
		return "Part Test"
	case "major_test":
		return "Major Test"
	case "full_syllabus_test":
		return "Full Syllabus Test"
	case "evaluation_test":
		return "Evaluation Test"
	case "mock_test":
		return "Mock Test"
	case "nta_test":
		return "NTA Test"
	case "homework_assignment":
		return "Homework Assignment"
	default:
		return "Unknown"
	}
}

// ChapterTestSubjectID returns the subject a chapter_test's problems belong to, derived from
// TypeParams.Subjects rather than stored separately - a chapter_test's problems all come from
// one chapter, and a chapter belongs to exactly one subject, so a well-formed chapter_test
// should carry exactly one entry. Returns nil before any problems are added (Subjects is empty)
// or if that single-subject invariant is ever violated, rather than guessing which one applies.
func (t *Test) ChapterTestSubjectID() *int8 {
	if len(t.TypeParams.Subjects) != 1 {
		return nil
	}
	return &t.TypeParams.Subjects[0].SubjectID
}

func (t *Test) RecalculateTotalMarksFromSubjects() {
	var total int16

	for _, subject := range t.TypeParams.Subjects {
		total += int16(subject.Marks)
	}

	t.TypeParams.Marks = total
}
