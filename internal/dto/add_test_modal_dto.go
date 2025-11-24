package dto

import "github.com/avantifellows/nex-gen-cms/internal/models"

type AddTestDialogData struct {
	Subtype          string                   `json:"subtype"`
	CurriculumGrades []models.CurriculumGrade `json:"curriculum_grades"`
	ExamID           int8                     `json:"exam_id"`
}
