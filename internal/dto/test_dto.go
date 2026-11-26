package dto

import "github.com/avantifellows/nex-gen-cms/internal/models"

type TestData struct {
	HomeData
	TestPtr  *models.Test
	Problems map[int]*models.Problem // key = Problem ID
	TestRule *models.TestRule
	Syllabus SyllabusOptions `json:"-"`
	// Resolved from exams API (name JeeAdvancedExamName); 0 if not found.
	JeeAdvancedExamID int16 `json:"jee_advanced_exam_id"`
}

// SyllabusOptions is what the Test Syllabus dialog is built from.
type SyllabusOptions struct {
	CurriculumID int16
	TestGradeID  int8 // the grade ticked by default
	// Subjects the syllabus can contain, derived from the test's curriculum.
	Subjects []*models.Subject
	// Grades the syllabus can be limited to, ordered by grade number.
	Grades []*models.Grade
}
