package dto

import "github.com/avantifellows/nex-gen-cms/internal/models"

type HomeData struct {
	CurriculumID int16
	GradeID      int8
	SubjectID    int8
	ChapterPtr   *models.Chapter
	TestPtr      *models.Test
	ProblemPtr   *models.Problem
	Problems     map[int]*models.Problem // key = Problem ID
	TopicPtr     *models.Topic
	TestRule     *models.TestRule
	// Resolved from exams API (name JeeAdvancedExamName); 0 if not found.
	JeeAdvancedExamID int16 `json:"jee_advanced_exam_id"`
}
