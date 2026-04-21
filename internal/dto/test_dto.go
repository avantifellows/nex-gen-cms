package dto

import "github.com/avantifellows/nex-gen-cms/internal/models"

type TestData struct {
	HomeData
	TestPtr  *models.Test
	Problems map[int]*models.Problem // key = Problem ID
	TestRule *models.TestRule
	// Resolved from exams API (name JeeAdvancedExamName); 0 if not found.
	JeeAdvancedExamID int16 `json:"jee_advanced_exam_id"`
}
