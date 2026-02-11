package models

type Test struct {
	ID               int               `json:"id,omitempty"`
	Name             []ResName         `json:"name"`
	Code             string            `json:"code"`
	Type             string            `json:"type"`
	Subtype          string            `json:"subtype"`
	ExamIDs          []int8            `json:"exam_ids"`
	SkillIDs         []int16           `json:"skill_ids,omitempty"`
	CurriculumGrades []CurriculumGrade `json:"curriculum_grades"`
	TypeParams       ResTypeParams     `json:"type_params"`
	Status           string            `json:"cms_status"`
}

type CurriculumGrade struct {
	CurriculumID int16 `json:"curriculum_id"`
	GradeID      int8  `json:"grade_id"`
}

type ResName struct {
	LangCode string `json:"lang_code"`
	Resource string `json:"resource"`
}

type ResTypeParams struct {
	Duration string       `json:"duration"`
	Marks    int16        `json:"marks"`
	PosMarks []int8       `json:"pos_marks,omitempty"`
	NegMarks []int8       `json:"neg_marks,omitempty"`
	Subjects []ResSubject `json:"subjects,omitempty"`
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
	Name       string        `json:"name"`
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
	ID       int    `json:"id"`
	PosMarks []int8 `json:"pos_marks"`
	NegMarks []int8 `json:"neg_marks,omitempty"`
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

func (test *Test) SetCurriculumGrade(curriculumID int16, gradeID int8) {
	newPair := CurriculumGrade{
		CurriculumID: curriculumID,
		GradeID:      gradeID,
	}

	// If nil or empty, initialize with the new pair
	if test.CurriculumGrades == nil || len(test.CurriculumGrades) == 0 {
		test.CurriculumGrades = []CurriculumGrade{newPair}
		return
	}

	// Check if the pair already exists
	for _, cg := range test.CurriculumGrades {
		if cg.CurriculumID == curriculumID && cg.GradeID == gradeID {
			return // Already exists, do nothing
		}
	}

	// Append if not found
	test.CurriculumGrades = append(test.CurriculumGrades, newPair)
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

func (t *Test) RecalculateTotalMarksFromSubjects() {
	var total int16 = 0

	for _, subject := range t.TypeParams.Subjects {
		total += int16(subject.Marks)
	}

	t.TypeParams.Marks = total
}
