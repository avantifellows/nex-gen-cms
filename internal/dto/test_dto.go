package dto

import (
	"slices"

	"github.com/avantifellows/nex-gen-cms/internal/models"
)

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
	// Saved is the test's saved syllabus; nil if it has none.
	Saved *models.TestSyllabus
}

// GradeTicked reports whether a grade starts ticked: the saved grades, or else the test's own grade.
func (o SyllabusOptions) GradeTicked(gradeID int8) bool {
	if o.Saved == nil {
		return gradeID == o.TestGradeID
	}
	return slices.Contains(o.Saved.GradeIDs, gradeID)
}

// SubjectTicked reports whether a subject starts as part of the syllabus.
func (o SyllabusOptions) SubjectTicked(subjectID int8) bool {
	if o.Saved == nil {
		return false
	}
	return slices.ContainsFunc(o.Saved.Subjects, func(s models.SyllabusSubject) bool { return s.SubjectID == subjectID })
}

// SavedChapterIDs returns the chapters of the saved syllabus, across subjects.
func (o SyllabusOptions) SavedChapterIDs() []int16 {
	ids := []int16{}
	if o.Saved != nil {
		for _, subject := range o.Saved.Subjects {
			ids = append(ids, subject.ChapterIDs...)
		}
	}
	return ids
}

// SyllabusChapters is what the chapter rows of the Test Syllabus dialog are built from.
type SyllabusChapters struct {
	Chapters    []*models.Chapter
	SelectedIDs map[int16]bool // chapters that start ticked
}
