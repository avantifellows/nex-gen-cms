package dto

import "github.com/avantifellows/nex-gen-cms/internal/models"

// Modes of the add-test dialog, which also serves copying a test and changing the setup
// (curriculum-grade, test type, exam) of an existing one.
const (
	AddTestDialogModeNew   = "new"
	AddTestDialogModeCopy  = "copy"
	AddTestDialogModeSetup = "setup"
)

type AddTestDialogData struct {
	Mode             string                   `json:"mode"`
	Subtype          string                   `json:"subtype"`
	CurriculumGrades []models.CurriculumGrade `json:"curriculum_grades"`
	ExamID           int8                     `json:"exam_id"`
}
