package models

type Test struct {
	ID               int               `json:"id"`
	Name             []ResName         `json:"name"`
	Code             string            `json:"code"`
	Type             string            `json:"type"`
	Subtype          string            `json:"subtype"`
	SkillIDs         []int16           `json:"skill_ids"`
	CurriculumGrades []CurriculumGrade `json:"curriculum_grades"`
	TypeParams       ResTypeParams     `json:"type_params"`
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
	PosMarks []int8       `json:"pos_marks"`
	NegMarks []int8       `json:"neg_marks"`
	Subjects []ResSubject `json:"subjects"`
}

type ResSubject struct {
	SubjectID int8 `json:"subject_id"`
	Name      string
	Marks     int          `json:"marks"`
	PosMarks  []int8       `json:"pos_marks"`
	NegMarks  []int8       `json:"neg_marks"`
	Sections  []ResSection `json:"sections"`
}

type ResSection struct {
	Name       string        `json:"name"`
	Marks      int16         `json:"marks"`
	PosMarks   []int8        `json:"pos_marks"`
	NegMarks   []int8        `json:"neg_marks"`
	Compulsory ResCompulsory `json:"compulsory"`
	Optional   ResOptional   `json:"optional"`
}

type ResCompulsory struct {
	Problems []ResProblem `json:"problems"`
}

type ResOptional struct {
	MandatoryCount int8         `json:"mandatory_count"`
	Problems       []ResProblem `json:"problems"`
}

type ResProblem struct {
	ID       int    `json:"id"`
	PosMarks []int8 `json:"pos_marks"`
	NegMarks []int8 `json:"neg_marks"`
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
			total += int(section.Optional.MandatoryCount)
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
